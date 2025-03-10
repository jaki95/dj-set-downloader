package djset

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// Define base directories for local storage - these can be configured globally
var (
	BaseDir      = "storage"           // Base directory for all storage
	ProcessesDir = "storage/processes" // Directory for process-specific storage
	OutputDir    = "output"            // Directory for final output
)

type processor struct {
	tracklistImporter tracklist.Importer
	setDownloader     downloader.Downloader
	audioProcessor    audio.Processor
}

// New creates a new processor instance
func New(tracklistImporter tracklist.Importer, setDownloader downloader.Downloader, audioProcessor audio.Processor) *processor {
	return &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
	}
}

type ProcessingOptions struct {
	TracklistPath      string
	DJSetURL           string
	FileExtension      string
	MaxConcurrentTasks int
}

// processingContext encapsulates all state and options needed during processing
type processingContext struct {
	processID        string
	opts             *ProcessingOptions
	progressCallback func(int, string)
	set              *domain.Tracklist
	setLength        int

	// File information
	fileName        string
	downloadedFile  string
	actualExtension string
	coverArtPath    string

	// Paths
	processDir       string
	downloadDir      string
	tempDir          string
	processOutputDir string
}

// newProcessingContext creates a new processing context with the given options
func newProcessingContext(opts *ProcessingOptions, progressCallback func(int, string)) *processingContext {
	// Generate a unique process ID
	processID := uuid.New().String()

	// Create paths
	processDir := filepath.Join(ProcessesDir, processID)
	downloadDir := filepath.Join(processDir, "download")
	tempDir := filepath.Join(processDir, "temp")

	return &processingContext{
		processID:        processID,
		opts:             opts,
		progressCallback: progressCallback,
		setLength:        0, // Will be set after importing tracklist
		processDir:       processDir,
		downloadDir:      downloadDir,
		tempDir:          tempDir,
	}
}

func (p *processor) ProcessTracks(opts *ProcessingOptions, progressCallback func(int, string)) ([]string, error) {
	// Setup a processing context
	ctx := newProcessingContext(opts, progressCallback)

	// Create base directories
	if err := ensureBaseDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create base directories: %w", err)
	}

	// Create process-specific directories
	if err := ensureProcessDirectories(ctx); err != nil {
		return nil, fmt.Errorf("failed to create process directories: %w", err)
	}

	// Step 1: Import tracklist
	if err := p.importTracklist(ctx); err != nil {
		return nil, err
	}

	// Step 2: Download set
	if err := p.downloadSet(ctx); err != nil {
		return nil, err
	}

	// Step 3: Prepare for processing (extract cover art)
	if err := p.prepareForProcessing(ctx); err != nil {
		return nil, err
	}

	// Step 4: Split tracks
	results, err := p.processTracks(ctx)
	if err != nil {
		return nil, err
	}

	// Cleanup process directories unless there was an error
	if err == nil {
		go cleanupProcessDir(ctx.processDir)
	}

	return results, nil
}

// cleanupProcessDir removes process directories after a delay
func cleanupProcessDir(processDir string) {
	// Wait 5 minutes before cleanup to ensure files aren't still in use
	time.Sleep(5 * time.Minute)
	if err := os.RemoveAll(processDir); err != nil {
		slog.Error("failed to cleanup process directory", "dir", processDir, "error", err)
	}
}

// ensureBaseDirectories creates the necessary base directories
func ensureBaseDirectories() error {
	dirs := []string{BaseDir, ProcessesDir, OutputDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// ensureProcessDirectories creates process-specific directories
func ensureProcessDirectories(ctx *processingContext) error {
	dirs := []string{ctx.processDir, ctx.downloadDir, ctx.tempDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

func (p *processor) importTracklist(ctx *processingContext) error {
	set, err := p.tracklistImporter.Import(ctx.opts.TracklistPath)
	if err != nil {
		return err
	}

	// Store the tracklist in the context
	ctx.set = set
	ctx.setLength = len(set.Tracks)

	// Create a specific output directory for this set
	sanitizedSetName := sanitizeTitle(set.Name)
	ctx.processOutputDir = filepath.Join(OutputDir, sanitizedSetName)
	if err := os.MkdirAll(ctx.processOutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory for set: %w", err)
	}

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

	// Download the set to the download directory with a predictable name
	sanitizedSetName := sanitizeTitle(ctx.set.Name)

	// Download the set - we'll let the downloader determine the extension
	err = p.setDownloader.Download(url, sanitizedSetName, ctx.downloadDir, func(progress int, message string) {
		// Adjust the progress calculation to ensure it never goes negative
		// Scale the original 0-100% to 10-50% range
		adjustedProgress := 10 + ((progress * 40) / 100)
		ctx.progressCallback(adjustedProgress, message)
	})
	if err != nil {
		return err
	}

	// Get all files in the download directory
	files, err := os.ReadDir(ctx.downloadDir)
	if err != nil {
		return fmt.Errorf("failed to read download directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in download directory %s", ctx.downloadDir)
	}

	// Use the first file (should be our downloaded set)
	downloadedFile := filepath.Join(ctx.downloadDir, files[0].Name())
	ctx.downloadedFile = downloadedFile
	ctx.fileName = downloadedFile

	// Get the extension from the filename
	ctx.actualExtension = strings.TrimPrefix(filepath.Ext(downloadedFile), ".")
	if ctx.actualExtension == "" {
		// Default to mp3 if no extension
		ctx.actualExtension = "mp3"
	}

	slog.Info("Downloaded set", "file", downloadedFile, "extension", ctx.actualExtension)
	return nil
}

func (p *processor) prepareForProcessing(ctx *processingContext) error {
	slog.Info("Preparing for processing", "set", ctx.set.Name, "file", ctx.fileName)

	// Create a specific path for the cover art in the temp directory
	coverArtPath := filepath.Join(ctx.tempDir, "cover.jpg")
	ctx.coverArtPath = coverArtPath

	// Extract cover art
	if err := p.audioProcessor.ExtractCoverArt(ctx.fileName, coverArtPath); err != nil {
		return fmt.Errorf("failed to extract cover art: %w", err)
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

	return p.splitTracks(ctx, updatedOpts, bar)
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-", " ", "_")
	return replacer.Replace(title)
}

func (p *processor) splitTracks(
	ctx *processingContext,
	opts ProcessingOptions,
	bar *progressbar.ProgressBar,
) ([]string, error) {
	// Configure worker pool
	maxWorkers := p.getMaxWorkers(opts.MaxConcurrentTasks)

	// Setup processing context and channels
	processingCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup channels for communication
	errCh := make(chan error, 1)
	filePathCh := make(chan string, ctx.setLength)
	semaphore := make(chan struct{}, maxWorkers)

	// Start initial progress
	slog.Info("Splitting tracks", "count", ctx.setLength, "extension", opts.FileExtension)
	ctx.progressCallback(50, fmt.Sprintf("Processing %d tracks", ctx.setLength))

	// Start track processing with a pool of workers
	var wg sync.WaitGroup
	var completedTracks int32 // Atomic counter for completed tracks

	// Launch worker goroutines for each track
	for i, t := range ctx.set.Tracks {
		wg.Add(1)
		t := t // Capture variable
		i := i

		go func(i int, t *domain.Track) {
			defer wg.Done()

			// Check if processing should be canceled
			select {
			case <-processingCtx.Done():
				return
			default:
			}

			// Acquire semaphore slot or abort if canceled
			select {
			case semaphore <- struct{}{}:
			case <-processingCtx.Done():
				return
			}
			defer func() { <-semaphore }()

			// Process the track and handle errors
			p.processTrack(
				processingCtx, cancel,
				i, t,
				ctx,
				opts,
				bar,
				&completedTracks,
				errCh,
				filePathCh,
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
	filePaths := make([]string, 0, ctx.setLength)
	for path := range filePathCh {
		filePaths = append(filePaths, path)
	}

	// Check if any errors occurred
	if err := <-errCh; err != nil {
		return nil, err
	}

	slog.Info("Splitting completed")
	ctx.progressCallback(100, "Processing completed")
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
	processingCtx *processingContext,
	opts ProcessingOptions,
	bar *progressbar.ProgressBar,
	completedTracks *int32,
	errCh chan<- error,
	filePathCh chan<- string,
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
	trackName := fmt.Sprintf("%02d-%s", trackNumber, safeTitle)

	// Create a track-specific temp directory for processing
	trackTempDir := filepath.Join(processingCtx.tempDir, fmt.Sprintf("track_%02d", trackNumber))
	if err := os.MkdirAll(trackTempDir, os.ModePerm); err != nil {
		select {
		case errCh <- fmt.Errorf("failed to create track temp directory for track %d (%s): %w",
			trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Create output paths
	tempOutputPath := filepath.Join(trackTempDir, trackName)
	finalOutputPath := filepath.Join(processingCtx.processOutputDir, trackName)

	// Create parameters for audio processor
	splitParams := audio.SplitParams{
		InputPath:     processingCtx.fileName,
		OutputPath:    tempOutputPath,
		FileExtension: opts.FileExtension,
		Track:         *track,
		TrackCount:    processingCtx.setLength,
		Artist:        processingCtx.set.Artist,
		Name:          processingCtx.set.Name,
		CoverArtPath:  processingCtx.coverArtPath,
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

	// Move the processed file to the final output directory
	tempFilePath := fmt.Sprintf("%s.%s", tempOutputPath, opts.FileExtension)
	finalFilePath := fmt.Sprintf("%s.%s", finalOutputPath, opts.FileExtension)

	// Read the processed file
	fileData, err := os.ReadFile(tempFilePath)
	if err != nil {
		select {
		case errCh <- fmt.Errorf("failed to read processed track file %d (%s): %w",
			trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Write to the final destination
	if err := os.WriteFile(finalFilePath, fileData, 0644); err != nil {
		select {
		case errCh <- fmt.Errorf("failed to write final track file %d (%s): %w",
			trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Update progress
	p.updateTrackProgress(bar, completedTracks, processingCtx.setLength, processingCtx.progressCallback)

	// Add the final path to the result channel
	filePathCh <- finalFilePath
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
