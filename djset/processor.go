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

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/storage"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
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

// processingContext encapsulates all state and options needed during processing
type processingContext struct {
	opts             *ProcessingOptions
	progressCallback func(int, string)
	set              *domain.Tracklist
	setLength        int

	// File information
	fileName        string
	downloadedFile  string
	actualExtension string
	coverArtPath    string
}

// newProcessingContext creates a new processing context with the given options
func newProcessingContext(opts *ProcessingOptions, progressCallback func(int, string)) *processingContext {
	return &processingContext{
		opts:             opts,
		progressCallback: progressCallback,
		setLength:        0, // Will be set after importing tracklist
	}
}

func (p *processor) ProcessTracks(opts *ProcessingOptions, progressCallback func(int, string)) ([]string, error) {
	// Setup a processing context
	ctx := newProcessingContext(opts, progressCallback)

	// Step 1: Import tracklist
	if err := p.importTracklist(ctx); err != nil {
		return nil, err
	}

	// Step 2: Download set if needed
	if err := p.downloadSet(ctx); err != nil {
		return nil, err
	}

	// Step 3: Prepare for processing (find files, extract cover art)
	if err := p.prepareForProcessing(ctx); err != nil {
		return nil, err
	}

	// Step 4: Split tracks
	results, err := p.processTracks(ctx)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (p *processor) importTracklist(ctx *processingContext) error {
	set, err := p.tracklistImporter.Import(ctx.opts.TracklistPath)
	if err != nil {
		return err
	}

	// Store the tracklist in the context
	ctx.set = set
	ctx.setLength = len(set.Tracks)

	// Report progress
	ctx.progressCallback(10, "Got tracklist")
	return nil
}

func (p *processor) downloadSet(ctx *processingContext) error {
	url := ctx.opts.DJSetURL
	var err error

	if url == "" {
		// Try to find the URL using the user's search query if available, otherwise use metadata
		var input string
		if ctx.opts.TracklistPath != "" && strings.HasPrefix(ctx.opts.TracklistPath, "http") {
			// If the tracklist path looks like a URL, use it directly
			input = ctx.opts.TracklistPath
		} else {
			// Otherwise, use the set name and artist
			input = ctx.set.Name
			if ctx.set.Artist != "" {
				input = fmt.Sprintf("%s %s", ctx.set.Artist, input)
			}
		}

		input = strings.TrimSpace(input)
		url, err = p.setDownloader.FindURL(input)
		if err != nil {
			return err
		}
	}
	slog.Debug("found match", "url", url)

	// Get the download directory from storage
	downloadPath, err := p.getDownloadPath()
	if err != nil {
		return fmt.Errorf("failed to get download path: %w", err)
	}

	slog.Debug("downloading set", "url", url, "downloadPath", downloadPath)

	err = p.setDownloader.Download(url, ctx.set.Name, downloadPath, func(progress int, message string) {
		// Adjust the progress calculation to ensure it never goes negative
		// Scale the original 0-100% to 10-50% range
		adjustedProgress := 10 + (progress / 2) // This ensures progress starts at 10% and goes up to 50%
		ctx.progressCallback(adjustedProgress, message)
	})
	if err != nil {
		return err
	}

	return nil
}

func (p *processor) prepareForProcessing(ctx *processingContext) error {
	// Find files in the data directory that match the set name
	// This will be encapsulated in storage later
	files, err := p.storage.ListFiles("", ctx.set.Name)
	if err != nil {
		return fmt.Errorf("error listing files: %w", err)
	}

	slog.Debug("found files", "files", files)

	ctx.downloadedFile = ""
	ctx.actualExtension = ""

	for _, filePath := range files {
		fileName := filepath.Base(filePath)
		if strings.HasPrefix(fileName, ctx.set.Name) {
			ctx.downloadedFile = fileName
			parts := strings.Split(ctx.downloadedFile, ".")
			if len(parts) > 1 {
				ctx.actualExtension = parts[len(parts)-1]
			} else {
				// If no extension found, fall back to config
				ctx.actualExtension = ctx.opts.FileExtension
			}
			break
		}
	}

	// Get the file path from storage
	ctx.fileName = p.storage.GetSetPath(ctx.set.Name, ctx.actualExtension)
	ctx.coverArtPath = p.storage.GetCoverArtPath()

	// Check if the file exists before attempting to extract cover art
	if !p.storage.FileExists(ctx.fileName) {
		// Try to find the file using its full path
		files, err := p.storage.ListFiles("", "")
		if err != nil {
			return fmt.Errorf("error listing files: %w", err)
		}

		var matchingFiles []string
		for _, file := range files {
			if strings.Contains(strings.ToLower(file), strings.ToLower(ctx.set.Name)) {
				matchingFiles = append(matchingFiles, file)
			}
		}

		if len(matchingFiles) > 0 {
			// Found potential matches
			errorMsg := fmt.Sprintf("Audio file not found: %s\nPotential matches found:\n", ctx.fileName)
			for i, match := range matchingFiles {
				errorMsg += fmt.Sprintf("  %d. %s\n", i+1, match)
			}
			return fmt.Errorf("%s", errorMsg)
		}

		return fmt.Errorf("audio file not found: %s (make sure file exists in the download directory)", ctx.fileName)
	}

	slog.Info("Extracting cover art", "set", ctx.set.Name, "file", ctx.fileName)
	if err := p.audioProcessor.ExtractCoverArt(ctx.fileName, ctx.coverArtPath); err != nil {
		return fmt.Errorf("failed to extract cover art: %w", err)
	}
	defer func() {
		if err := p.storage.Cleanup(); err != nil {
			slog.Error("failed to cleanup storage", "error", err)
		}
	}()

	// Create output directory for the set
	if err := p.storage.CreateSetOutputDir(ctx.set.Name); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return nil
}

func (p *processor) processTracks(ctx *processingContext) ([]string, error) {
	bar := progressbar.NewOptions(
		ctx.setLength,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Processing tracks..."),
	)

	// Use the updated ProcessingOptions with the actual extension
	updatedOpts := *ctx.opts
	updatedOpts.FileExtension = ctx.actualExtension

	return p.splitTracks(*ctx.set, updatedOpts, ctx.setLength, ctx.fileName, ctx.coverArtPath, bar, ctx.progressCallback)
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
	// Configure worker pool
	maxWorkers := p.getMaxWorkers(opts.MaxConcurrentTasks)

	// Setup processing context and channels
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup channels for communication
	errCh := make(chan error, 1)
	filePathCh := make(chan string, len(set.Tracks))
	semaphore := make(chan struct{}, maxWorkers)

	// Start initial progress
	slog.Info("Splitting tracks", "count", len(set.Tracks), "extension", opts.FileExtension)
	progressCallback(50, fmt.Sprintf("Processing %d tracks", len(set.Tracks)))

	// Start track processing with a pool of workers
	var wg sync.WaitGroup
	var completedTracks int32 // Atomic counter for completed tracks

	// Launch worker goroutines for each track
	for i, t := range set.Tracks {
		wg.Add(1)
		t := t // Capture variable
		i := i

		go func(i int, t *domain.Track) {
			defer wg.Done()

			// Check if processing should be canceled
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Acquire semaphore slot or abort if canceled
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()

			// Process the track and handle errors
			p.processTrack(
				ctx, cancel,
				i, t,
				set, opts,
				setLength,
				fileName, coverArtPath,
				bar,
				&completedTracks,
				errCh,
				filePathCh,
				progressCallback,
			)
		}(i, t)
	}

	// Wait for all workers to complete and gather results
	go func() {
		wg.Wait()
		close(filePathCh)
		close(errCh)
	}()

	// Collect file paths from successful operations
	filePaths := make([]string, 0, len(set.Tracks))
	for path := range filePathCh {
		filePaths = append(filePaths, path)
	}

	// Check if any errors occurred
	if err := <-errCh; err != nil {
		return nil, err
	}

	slog.Info("Splitting completed")
	progressCallback(100, "Processing completed")
	return filePaths, nil
}

// getMaxWorkers returns a valid number of concurrent workers
func (p *processor) getMaxWorkers(requested int) int {
	if requested < 1 || requested > 10 {
		return 4 // Default value
	}
	return requested
}

// processTrack handles the processing of a single track
func (p *processor) processTrack(
	ctx context.Context,
	cancel context.CancelFunc,
	trackIndex int,
	track *domain.Track,
	set domain.Tracklist,
	opts ProcessingOptions,
	setLength int,
	fileName string,
	coverArtPath string,
	bar *progressbar.ProgressBar,
	completedTracks *int32,
	errCh chan<- error,
	filePathCh chan<- string,
	progressCallback func(int, string),
) {
	// Set up monitoring for long-running operations
	processingStartTime := time.Now()
	trackNumber := trackIndex + 1
	done := make(chan struct{})

	// Monitor for slow track processing
	go p.monitorTrackProcessing(done, processingStartTime, trackNumber, track.Title)
	defer close(done)

	// Create track name with proper numbering
	safeTitle := sanitizeTitle(track.Title)
	trackName := fmt.Sprintf("%02d - %s", trackNumber, safeTitle)

	// Get the output path from the storage layer
	outputPath, err := p.storage.SaveTrack(set.Name, trackName, opts.FileExtension)
	if err != nil {
		select {
		case errCh <- fmt.Errorf("failed to create output path for track %d (%s): %w",
			trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Remove the extension as the audio processor will add it
	outputPath = strings.TrimSuffix(outputPath, "."+opts.FileExtension)

	// Create parameters for audio processor
	splitParams := audio.SplitParams{
		InputPath:     fileName,
		OutputPath:    outputPath,
		FileExtension: opts.FileExtension,
		Track:         *track,
		TrackCount:    setLength,
		Artist:        set.Artist,
		Name:          set.Name,
		CoverArtPath:  coverArtPath,
	}

	// Process the track
	if err := p.audioProcessor.Split(splitParams); err != nil {
		errorMsg := fmt.Sprintf("Failed to split track %d (%s): %v", trackNumber, track.Title, err)
		slog.Error(errorMsg)
		select {
		case errCh <- fmt.Errorf("failed to process track %d (%s): %w", trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Update progress
	p.updateTrackProgress(bar, completedTracks, setLength, progressCallback)

	// Add the full path to the result channel
	finalPath := fmt.Sprintf("%s.%s", outputPath, opts.FileExtension)
	filePathCh <- finalPath
}

// monitorTrackProcessing logs a warning if track processing takes too long
func (p *processor) monitorTrackProcessing(
	done <-chan struct{},
	startTime time.Time,
	trackNumber int,
	trackTitle string,
) {
	warningThreshold := 2 * time.Minute
	ticker := time.NewTicker(warningThreshold)
	defer ticker.Stop()

	select {
	case <-done:
		return
	case <-ticker.C:
		elapsed := time.Since(startTime)
		slog.Warn("Track processing is taking longer than expected",
			"track", trackNumber,
			"title", trackTitle,
			"elapsed", elapsed.String(),
		)
	}
}

// updateTrackProgress updates the progress indicators for track processing
func (p *processor) updateTrackProgress(
	bar *progressbar.ProgressBar,
	completedTracks *int32,
	setLength int,
	progressCallback func(int, string),
) {
	// Increment completed tracks atomically
	newCount := atomic.AddInt32(completedTracks, 1)
	trackProgress := int((float64(newCount) / float64(setLength)) * 100)
	totalProgress := 50 + (trackProgress / 2) // Scale to 50-100%
	_ = bar.Add(1)
	progressCallback(totalProgress, fmt.Sprintf("Processed %d/%d tracks", newCount, setLength))
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
