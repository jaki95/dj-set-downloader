package djset

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"

	"github.com/jaki95/dj-set-downloader/config"
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

func New(cfg *config.Config) (*processor, error) {
	tracklistImporter, err := tracklist.NewImporter(cfg)
	if err != nil {
		return nil, err
	}
	setDownloader, err := downloader.NewDownloader(cfg)
	if err != nil {
		return nil, err
	}
	audioProcessor := audio.NewFFMPEGEngine()

	// Initialize storage
	fileStorage, err := storage.NewLocalFileStorage("data", "output", "data")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
		storage:           fileStorage,
	}, nil
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

	setLength := len(set.Tracks)

	url := opts.DJSetURL
	if url == "" {
		findQuery := fmt.Sprintf("%s %s", set.Name, set.Artist)
		url, err = p.setDownloader.FindURL(findQuery)
		if err != nil {
			fmt.Println("could not find match, input name of set:")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			input = strings.TrimSpace(input)
			url, err = p.setDownloader.FindURL(input)
			if err != nil {
				return nil, err
			}
		}
		slog.Debug("found match", "url", url)
	}

	err = p.setDownloader.Download(url, set.Name, func(progress int, message string) {
		totalProgress := progress / 2 // Scale to 0-50%
		progressCallback(totalProgress, message)
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

	if downloadedFile == "" {
		slog.Warn("could not find downloaded file, using configured extension", "extension", opts.FileExtension)
		downloadedFile = fmt.Sprintf("%s.%s", set.Name, opts.FileExtension)
		actualExtension = opts.FileExtension
	} else {
		slog.Debug("found downloaded file", "file", downloadedFile, "extension", actualExtension)
	}

	// Make sure we have a non-empty extension
	if actualExtension == "" {
		slog.Warn("empty file extension detected, falling back to configured extension", "extension", opts.FileExtension)
		actualExtension = opts.FileExtension
	}

	// Get the file path from storage
	fileName := p.storage.GetSetPath(set.Name, actualExtension)
	coverArtPath := p.storage.GetCoverArtPath()

	if err := p.audioProcessor.ExtractCoverArt(fileName, coverArtPath); err != nil {
		return nil, err
	}
	defer p.storage.Cleanup() // This will clean up the cover art

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
		slog.Warn("invalid max workers, defaulting to 1", "maxWorkers", opts.MaxConcurrentTasks)
		maxWorkers = 1
	}
	semaphore := make(chan struct{}, maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	filePathCh := make(chan string, len(set.Tracks))

	slog.Info("Starting track splitting", "trackCount", setLength, "fileExtension", opts.FileExtension)
	progressCallback(50, "Processing tracks...")

	for i, t := range set.Tracks {
		wg.Add(1)
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
				select {
				case errCh <- fmt.Errorf("track %d (%s): %w", i+1, t.Title, err):
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
