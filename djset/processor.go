package djset

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/storage"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

type processor struct {
	tracklistImporter tracklist.Importer
	setDownloader     downloader.Downloader
	audioProcessor    audio.Processor
	storage           storage.Storage
}

type ProcessingOptions struct {
	TracklistPath      string
	DJSetURL           string
	FileExtension      string
	MaxConcurrentTasks int
}

func (p *processor) ProcessTracks(
	opts *ProcessingOptions,
	progressCallback func(int, string),
) ([]string, error) {
	set, err := p.tracklistImporter.Import(opts.TracklistPath)
	if err != nil {
		return nil, err
	}

	progressCallback(10, "Got tracklist")

	setLength := len(set.Tracks)

	url := opts.DJSetURL
	if url == "" {
		// Try to find the URL using the user's search query if available, otherwise use metadata
		var input string
		if !strings.Contains(opts.TracklistPath, "https://") {
			// Use the user's search query
			input = opts.TracklistPath
			slog.Debug("using user-provided search query", "query", input)
		} else {
			// Fall back to constructing query from tracklist metadata
			input = fmt.Sprintf("%s %s", set.Artist, set.Name)
			slog.Debug("using tracklist metadata for search", "artist", set.Artist, "name", set.Name)
		}

		input = strings.TrimSpace(input)
		url, err = p.setDownloader.FindURL(input)
		if err != nil {
			return nil, err
		}
	}
	slog.Debug("found match", "url", url)

	// Get the download directory from storage
	downloadPath, err := p.getDownloadPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get download path: %w", err)
	}

	slog.Debug("downloading set", "url", url, "downloadPath", downloadPath)

	err = p.setDownloader.Download(url, set.Name, downloadPath, func(progress int, message string) {
		// Adjust the progress calculation to ensure it never goes negative
		// Scale the original 0-100% to 10-50% range
		adjustedProgress := 10 + (progress / 2)
		progressCallback(adjustedProgress, message)
	})
	if err != nil {
		return nil, err
	}

	// Find files in the data directory that match the set name
	// This will be encapsulated in storage later
	files, err := p.storage.ListFiles("", set.Name)
	if err != nil {
		return nil, fmt.Errorf("error listing files: %w", err)
	}

	slog.Debug("found files", "files", files)

	var downloadedFile string
	var actualExtension string

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		if strings.HasPrefix(fileName, set.Name) {
			downloadedFile = fileName
			parts := strings.Split(downloadedFile, ".")
			if len(parts) > 1 {
				actualExtension = parts[len(parts)-1]
			} else {
				// If no extension found, fall back to config
				actualExtension = opts.FileExtension
			}
			break
		}
	}

	// Get the file path from storage
	fileName := p.storage.GetSetPath(set.Name, actualExtension)
	coverArtPath := p.storage.GetCoverArtPath()

	// Check if the file exists before attempting to extract cover art
	if !p.storage.FileExists(fileName) {
		// Try to find the file using its full path
		files, err := p.storage.ListFiles("", "")
		if err != nil {
			return nil, fmt.Errorf("error listing files: %w", err)
		}

		var matchingFiles []string
		for _, file := range files {
			if strings.Contains(strings.ToLower(file), strings.ToLower(set.Name)) {
				matchingFiles = append(matchingFiles, file)
			}
		}

		if len(matchingFiles) > 0 {
			// Found potential matches
			errorMsg := fmt.Sprintf("Audio file not found: %s\nPotential matches found:\n", fileName)
			for i, match := range matchingFiles {
				errorMsg += fmt.Sprintf("  %d. %s\n", i+1, match)
			}
			return nil, fmt.Errorf("%s", errorMsg)
		}

		return nil, fmt.Errorf("audio file not found: %s (make sure file exists in the download directory)", fileName)
	}

	slog.Info("Extracting cover art", "set", set.Name, "file", fileName)
	if err := p.audioProcessor.ExtractCoverArt(fileName, coverArtPath); err != nil {
		return nil, fmt.Errorf("failed to extract cover art: %w", err)
	}
	defer func() {
		if err := p.storage.Cleanup(); err != nil {
			slog.Error("failed to cleanup storage", "error", err)
		}
	}()

	// Create output directory for the set
	if err := p.storage.CreateSetOutputDir(set.Name); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	bar := progressbar.NewOptions(
		setLength,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Processing tracks..."),
	)

	// Use the updated ProcessingOptions with the actual extension
	updatedOpts := opts
	updatedOpts.FileExtension = actualExtension

	return p.splitTracks(*set, *updatedOpts, setLength, fileName, coverArtPath, bar, progressCallback)
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}

func (p *processor) splitTracks(
	set domain.Tracklist,
	opts ProcessingOptions,
	setLength int,
	fileName string,
	coverArtPath string,
	bar *progressbar.ProgressBar,
	progressCallback func(int, string),
) ([]string, error) {
	var wg sync.WaitGroup
	var completedTracks int32 // Atomic counter for completed tracks
	maxWorkers := opts.MaxConcurrentTasks
	if maxWorkers < 1 || maxWorkers > 10 {
		maxWorkers = 4
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	filePathCh := make(chan string, len(set.Tracks))
	semaphore := make(chan struct{}, maxWorkers)

	slog.Info("Splitting tracks", "count", len(set.Tracks), "extension", opts.FileExtension)
	progressCallback(50, fmt.Sprintf("Processing %d tracks", len(set.Tracks)))

	for i, t := range set.Tracks {
		wg.Add(1)
		t := t // Capture variable
		i := i

		go func(i int, t *domain.Track) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()

			// Add a monitoring goroutine to log if processing takes too long
			processingStartTime := time.Now()
			trackNumber := i + 1
			done := make(chan struct{})

			go func() {
				warningThreshold := 2 * time.Minute
				ticker := time.NewTicker(warningThreshold)
				defer ticker.Stop()

				select {
				case <-done:
					return
				case <-ticker.C:
					elapsed := time.Since(processingStartTime)
					slog.Warn("Track processing is taking longer than expected",
						"track", trackNumber,
						"title", t.Title,
						"elapsed", elapsed.String(),
					)
				}
			}()

			defer close(done)

			safeTitle := sanitizeTitle(t.Title)
			trackName := fmt.Sprintf("%02d - %s", i+1, safeTitle)

			// Get the output path from the storage layer
			outputPath, err := p.storage.SaveTrack(set.Name, trackName, opts.FileExtension)
			if err != nil {
				select {
				case errCh <- fmt.Errorf("failed to create output path for track %d (%s): %w", i+1, t.Title, err):
					cancel()
				default:
				}
				return
			}

			// Remove the extension as the audio processor will add it
			outputPath = strings.TrimSuffix(outputPath, "."+opts.FileExtension)

			splitParams := audio.SplitParams{
				InputPath:     fileName,
				OutputPath:    outputPath,
				FileExtension: opts.FileExtension,
				Track:         *t,
				TrackCount:    setLength,
				Artist:        set.Artist,
				Name:          set.Name,
				CoverArtPath:  coverArtPath,
			}

			if err := p.audioProcessor.Split(splitParams); err != nil {
				errorMsg := fmt.Sprintf("Failed to split track %d (%s): %v", i+1, t.Title, err)
				slog.Error(errorMsg)
				select {
				case errCh <- fmt.Errorf("failed to process track %d (%s): %w", i+1, t.Title, err):
					cancel()
				default:
				}
				return
			}

			// Increment completed tracks atomically
			newCount := atomic.AddInt32(&completedTracks, 1)
			trackProgress := int((float64(newCount) / float64(setLength)) * 100)
			totalProgress := 50 + (trackProgress / 2) // Scale to 50-100%
			_ = bar.Add(1)
			progressCallback(totalProgress, fmt.Sprintf("Processed %d/%d tracks", newCount, setLength))

			// Add the full path to the result channel
			finalPath := fmt.Sprintf("%s.%s", outputPath, opts.FileExtension)
			filePathCh <- finalPath
		}(i, t)
	}

	go func() {
		wg.Wait()
		close(filePathCh)
		close(errCh)
	}()

	filePaths := make([]string, 0, len(set.Tracks))
	for path := range filePathCh {
		filePaths = append(filePaths, path)
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	slog.Info("Splitting completed")
	progressCallback(100, "Processing completed")
	return filePaths, nil
}

// Helper method to get the download path
func (p *processor) getDownloadPath() (string, error) {
	// For GCS, this will return the temp directory
	// For local storage, it will return the data directory
	tempFile, err := p.storage.SaveDownloadedSet("temp", "tmp")
	if err != nil {
		return "", err
	}

	// Return just the directory path
	return filepath.Dir(tempFile), nil
}
