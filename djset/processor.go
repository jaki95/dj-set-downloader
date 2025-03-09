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
	"github.com/jaki95/dj-set-downloader/internal/storage"
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

// ConfigureDirs allows setting custom directory paths (useful for testing)
func ConfigureDirs(baseDir, processesDir, outputDir string) (restoreFunc func()) {
	// Save original values
	origBaseDir := BaseDir
	origProcessesDir := ProcessesDir
	origOutputDir := OutputDir

	// Set new values
	BaseDir = baseDir
	ProcessesDir = processesDir
	OutputDir = outputDir

	// Return a function to restore the original values
	return func() {
		BaseDir = origBaseDir
		ProcessesDir = origProcessesDir
		OutputDir = origOutputDir
	}
}

type processor struct {
	tracklistImporter tracklist.Importer
	setDownloader     downloader.Downloader
	audioProcessor    audio.Processor
	storage           storage.Storage
}

// New creates a new processor instance
func New(tracklistImporter tracklist.Importer, setDownloader downloader.Downloader, audioProcessor audio.Processor, storage storage.Storage) *processor {
	return &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
		storage:           storage,
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
	localInputPath   string
}

// newProcessingContext creates a new processing context with the given options
func newProcessingContext(opts *ProcessingOptions, progressCallback func(int, string)) *processingContext {
	// Generate a unique process ID
	processID := uuid.New().String()

	return &processingContext{
		processID:        processID,
		opts:             opts,
		progressCallback: progressCallback,
		setLength:        0, // Will be set after importing tracklist
	}
}

func (p *processor) ProcessTracks(opts *ProcessingOptions, progressCallback func(int, string)) ([]string, error) {
	// Setup a processing context
	ctx := newProcessingContext(opts, progressCallback)

	// Create a process directory
	processDir, err := p.storage.CreateProcessDir(ctx.processID)
	if err != nil {
		return nil, fmt.Errorf("failed to create process directory: %w", err)
	}

	// Set the directories in the context
	ctx.processDir = processDir
	ctx.downloadDir = p.storage.GetDownloadDir(ctx.processID)
	ctx.tempDir = p.storage.GetTempDir(ctx.processID)

	// Step 1: Import tracklist
	if err := p.importTracklist(ctx); err != nil {
		return nil, err
	}

	// Step 2: Download set directly to local file system
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

	// Cleanup process directory in the background
	go func() {
		if err := p.storage.CleanupProcessDir(ctx.processID); err != nil {
			slog.Error("Failed to cleanup process directory", "processID", ctx.processID, "error", err)
		}
	}()

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

	// Create the output directory and store it in the context
	ctx.processOutputDir = p.storage.GetOutputDir(sanitizeTitle(set.Name))
	if err := p.storage.CreateDir(ctx.processOutputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
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

	// Prepare file name and paths
	sanitizedSetName := sanitizeTitle(ctx.set.Name)

	// Get local download directory path for external tools to use
	localDownloadDir := p.storage.GetLocalDownloadDir(ctx.processID)

	// Create local directories if they don't exist
	if err := os.MkdirAll(localDownloadDir, 0755); err != nil {
		return fmt.Errorf("failed to create local download directory: %w", err)
	}

	// Download the set directly to local storage
	err = p.setDownloader.Download(url, sanitizedSetName, localDownloadDir, func(progress int, message string) {
		// Adjust the progress calculation to ensure it never goes negative
		// Scale the original 0-100% to 10-50% range
		adjustedProgress := 10 + (progress / 2) // This ensures progress starts at 10% and goes up to 50%
		ctx.progressCallback(adjustedProgress, message)
	})
	if err != nil {
		return err
	}

	// Use local file paths for processing
	fileExt := ctx.opts.FileExtension
	if fileExt == "" {
		fileExt = "mp3" // Default to mp3 if no extension provided
	}

	// Determine the actual path of the downloaded file
	localFilePath := filepath.Join(localDownloadDir, fmt.Sprintf("%s.%s", sanitizedSetName, fileExt))

	// Check if the file exists with the expected extension
	if _, err := os.Stat(localFilePath); err != nil {
		// File might have been downloaded with a different extension
		// Try to find the actual file in the directory
		entries, err := os.ReadDir(localDownloadDir)
		if err != nil {
			return fmt.Errorf("failed to read download directory: %w", err)
		}

		fileFound := false
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), sanitizedSetName+".") {
				// Found a file with the sanitized name but different extension
				localFilePath = filepath.Join(localDownloadDir, entry.Name())
				fileExt = filepath.Ext(entry.Name())[1:] // Get extension without the dot
				fileFound = true
				break
			}
		}

		if !fileFound {
			return fmt.Errorf("downloaded file not found in directory: %s", localDownloadDir)
		}
	}

	// Set file info in the context
	ctx.fileName = localFilePath       // Use local path directly
	ctx.localInputPath = localFilePath // Set the local input path for processing
	ctx.downloadedFile = localFilePath
	ctx.actualExtension = fileExt

	// Set output directory for processed tracks (these will be stored in the storage provider)
	// Using the structure: processID/setName/
	outputBaseDir := p.storage.GetOutputDir("")
	ctx.processOutputDir = filepath.Join(
		outputBaseDir,
		ctx.processID,
		sanitizedSetName,
	)

	slog.Debug("set downloaded successfully",
		"local_path", ctx.localInputPath,
		"output_dir", ctx.processOutputDir,
		"extension", ctx.actualExtension)

	return nil
}

func (p *processor) prepareForProcessing(ctx *processingContext) error {
	slog.Info("Preparing for processing", "set", ctx.set.Name, "file", ctx.fileName)

	// Set cover art path
	ctx.coverArtPath = filepath.Join(ctx.tempDir, "cover.jpg")

	// Ensure storage directories exist
	if err := p.storage.CreateDir(ctx.tempDir); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Since we're now downloading directly to a local file, we can use it directly
	// Make sure we have a valid local input path
	if ctx.localInputPath == "" {
		return fmt.Errorf("local input path not set")
	}

	// Verify the file exists
	if _, err := os.Stat(ctx.localInputPath); err != nil {
		return fmt.Errorf("local input file not found at %s: %w", ctx.localInputPath, err)
	}

	slog.Debug("Extracting cover art", "from", ctx.localInputPath, "to", ctx.coverArtPath)

	// Extract cover art directly from the local file
	if err := p.audioProcessor.ExtractCoverArt(ctx.localInputPath, ctx.coverArtPath); err != nil {
		return fmt.Errorf("failed to extract cover art: %w", err)
	}

	// Upload the cover art to storage for potential future use
	coverArtData, err := os.ReadFile(ctx.coverArtPath)
	if err != nil {
		slog.Warn("Failed to read cover art for upload", "error", err)
		// Continue without cover art in storage
	} else {
		// Store the cover art in the storage backend
		coverArtStoragePath := filepath.Join(ctx.tempDir, "cover.jpg")
		if err := p.storage.WriteFile(coverArtStoragePath, coverArtData); err != nil {
			slog.Warn("Failed to upload cover art to storage", "error", err)
			// Continue without cover art in storage
		}
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

	// Define paths for storage (remote) and local processing
	remoteOutputPath := filepath.Join(processingCtx.processOutputDir, trackName)
	remoteFilePath := fmt.Sprintf("%s.%s", remoteOutputPath, opts.FileExtension)

	// Create local temporary output path for FFMPEG to write to
	localTempDir := filepath.Join(p.storage.GetLocalDownloadDir(processingCtx.processID), "..", "temp_output")
	if err := os.MkdirAll(localTempDir, 0755); err != nil {
		errMsg := fmt.Sprintf("failed to create local temp output directory for track %d: %v", trackNumber, err)
		slog.Error(errMsg)
		select {
		case errCh <- fmt.Errorf("%s", errMsg):
			cancel()
		default:
		}
		return
	}

	localOutputPath := filepath.Join(localTempDir, trackName)
	localFilePath := fmt.Sprintf("%s.%s", localOutputPath, opts.FileExtension)

	// Input file path is now provided by the context - it was downloaded once before track processing
	inputFilePath := processingCtx.localInputPath

	// Quick sanity check - make sure the input file exists
	if _, err := os.Stat(inputFilePath); err != nil {
		errMsg := fmt.Sprintf("input file doesn't exist at %s for track %d: %v", inputFilePath, trackNumber, err)
		slog.Error(errMsg)
		select {
		case errCh <- fmt.Errorf("%s", errMsg):
			cancel()
		default:
		}
		return
	}

	// Create parameters for audio processor with the local paths
	splitParams := audio.SplitParams{
		InputPath:     inputFilePath,
		OutputPath:    localOutputPath, // Write to local temp file first
		FileExtension: opts.FileExtension,
		Track:         *track,
		TrackCount:    processingCtx.setLength,
		Artist:        processingCtx.set.Artist,
		Name:          processingCtx.set.Name,
		CoverArtPath:  processingCtx.coverArtPath,
	}

	// Process the track to the local temporary location
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

	// Read the processed file from local filesystem
	slog.Debug("uploading split track to storage",
		"track", trackNumber,
		"local", localFilePath,
		"remote", remoteFilePath)

	outputData, err := os.ReadFile(localFilePath)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to read processed track %d (%s): %v", trackNumber, track.Title, err)
		slog.Error(errorMsg)
		select {
		case errCh <- fmt.Errorf("failed to read processed track %d (%s): %w", trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Upload the processed file to storage
	if err := p.storage.WriteFile(remoteFilePath, outputData); err != nil {
		errorMsg := fmt.Sprintf("Failed to upload track %d (%s) to storage: %v", trackNumber, track.Title, err)
		slog.Error(errorMsg)
		select {
		case errCh <- fmt.Errorf("failed to upload track %d (%s) to storage: %w", trackNumber, track.Title, err):
			cancel()
		default:
		}
		return
	}

	// Clean up temporary output file
	if err := os.Remove(localFilePath); err != nil {
		slog.Warn("failed to remove temporary output file",
			"track", trackNumber,
			"file", localFilePath,
			"error", err)
	}

	// Update progress
	p.updateTrackProgress(bar, completedTracks, processingCtx.setLength, processingCtx.progressCallback)

	// Add the final storage path to the result channel
	filePathCh <- remoteFilePath
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
