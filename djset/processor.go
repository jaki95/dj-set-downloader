package djset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

// Define base directories for local storage - these can be configured globally
var (
	OutputDir = "output"                                         // Directory for final output
	TempDir   = filepath.Join(os.TempDir(), "dj-set-downloader") // Base temp directory
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
	opts             *ProcessingOptions
	progressCallback func(int, string, []byte)
	set              *domain.Tracklist
	setLength        int

	// File information
	inputFile string // The downloaded input file
	extension string // The actual file extension
	outputDir string // The final output directory for this set
}

// newProcessingContext creates a new processing context with the given options
func newProcessingContext(opts *ProcessingOptions, progressCallback func(int, string, []byte)) *processingContext {
	return &processingContext{
		opts:             opts,
		progressCallback: progressCallback,
		setLength:        0, // Will be set after importing tracklist
	}
}

// getDownloadDir returns the directory for downloading files
func (ctx *processingContext) getDownloadDir() string {
	return filepath.Join(TempDir, "downloads")
}

// getTempDir returns the directory for temporary processing
func (ctx *processingContext) getTempDir() string {
	return filepath.Join(TempDir, "processing")
}

// getCoverArtPath returns the path for the cover art
func (ctx *processingContext) getCoverArtPath() string {
	return filepath.Join(ctx.getTempDir(), "cover.jpg")
}

func init() {
	// Clean up any leftover temp files from previous runs on startup
	os.RemoveAll(TempDir)
}

func (p *processor) ProcessTracks(ctx context.Context, opts *ProcessingOptions, progressCallback func(int, string, []byte)) ([]string, error) {
	// Setup a processing context
	procCtx := newProcessingContext(opts, progressCallback)

	// Create base directories
	if err := ensureBaseDirectories(); err != nil {
		return nil, fmt.Errorf("failed to create base directories: %w", err)
	}

	// Create temp directories
	if err := ensureTempDirectories(procCtx); err != nil {
		return nil, fmt.Errorf("failed to create temp directories: %w", err)
	}

	// Ensure temp files are cleaned up
	defer cleanupTempFiles(procCtx)

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 1: Import tracklist
	if err := p.importTracklist(procCtx); err != nil {
		return nil, err
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 2: Download set
	if err := p.downloadSet(ctx, procCtx); err != nil {
		return nil, err
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 3: Prepare for processing (extract cover art)
	if err := p.prepareForProcessing(ctx, procCtx); err != nil {
		return nil, err
	}

	// Check for cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Step 4: Split tracks
	results, err := p.processTracks(ctx, procCtx)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// ensureBaseDirectories creates the necessary base directories
func ensureBaseDirectories() error {
	if err := os.MkdirAll(OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	return nil
}

// ensureTempDirectories creates temporary directories
func ensureTempDirectories(ctx *processingContext) error {
	dirs := []string{ctx.getDownloadDir(), ctx.getTempDir()}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// cleanupTempFiles removes temporary files
func cleanupTempFiles(ctx *processingContext) {
	// Remove track-specific temp directories
	if err := os.RemoveAll(ctx.getDownloadDir()); err != nil {
		slog.Error("Failed to cleanup download directory", "error", err)
	}
	if err := os.RemoveAll(ctx.getTempDir()); err != nil {
		slog.Error("Failed to cleanup temp directory", "error", err)
	}

	// Try to remove the base temp directory if it's empty
	if empty, _ := isDirEmpty(TempDir); empty {
		if err := os.Remove(TempDir); err != nil {
			slog.Error("Failed to cleanup base temp directory", "error", err)
		}
	}
}

// isDirEmpty checks if a directory is empty
func isDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil // Directory not empty
	}
	if err == io.EOF {
		return true, nil // Directory empty
	}
	return false, err
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
	ctx.outputDir = filepath.Join(OutputDir, sanitizedSetName)
	if err := os.MkdirAll(ctx.outputDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create output directory for set: %w", err)
	}

	jsonData, err := json.Marshal(set)
	if err != nil {
		return fmt.Errorf("failed to marshal tracklist: %w", err)
	}

	// Report progress
	ctx.progressCallback(10, "Got tracklist", jsonData)
	return nil
}

func (p *processor) downloadSet(ctx context.Context, procCtx *processingContext) error {
	url := procCtx.opts.DJSetURL
	var err error

	if url == "" {
		// Try to find the URL using the user's search query if available, otherwise use metadata
		var input string
		if procCtx.opts.TracklistPath != "" && strings.HasPrefix(procCtx.opts.TracklistPath, "http") {
			// If the tracklist path looks like a URL, use it directly
			input = procCtx.opts.TracklistPath
		} else {
			// Otherwise, use the set name and artist
			input = procCtx.set.Name
			if procCtx.set.Artist != "" {
				input = fmt.Sprintf("%s %s", procCtx.set.Artist, input)
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
	sanitizedSetName := sanitizeTitle(procCtx.set.Name)
	downloadDir := procCtx.getDownloadDir()

	// Create download directory
	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	// Download the set - we'll let the downloader determine the extension
	err = p.setDownloader.Download(ctx, url, sanitizedSetName, downloadDir, func(progress int, message string) {
		adjustedProgress := 10 + ((progress * 40) / 100)
		procCtx.progressCallback(adjustedProgress, message, nil)
	})
	if err != nil {
		return err
	}

	// Get all files in the download directory
	files, err := os.ReadDir(downloadDir)
	if err != nil {
		return fmt.Errorf("failed to read download directory: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in download directory %s", downloadDir)
	}

	// Use the first file (should be our downloaded set)
	procCtx.inputFile = filepath.Join(downloadDir, files[0].Name())

	// Get the extension from the filename
	procCtx.extension = strings.TrimPrefix(filepath.Ext(procCtx.inputFile), ".")
	if procCtx.extension == "" {
		// Default to mp3 if no extension
		procCtx.extension = "mp3"
	}

	slog.Info("Downloaded set", "file", procCtx.inputFile, "extension", procCtx.extension)
	return nil
}

func (p *processor) prepareForProcessing(ctx context.Context, procCtx *processingContext) error {
	slog.Info("Preparing for processing", "set", procCtx.set.Name, "file", procCtx.inputFile)

	// Create temp directory
	if err := os.MkdirAll(procCtx.getTempDir(), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Extract cover art
	if err := p.audioProcessor.ExtractCoverArt(ctx, procCtx.inputFile, procCtx.getCoverArtPath()); err != nil {
		return fmt.Errorf("failed to extract cover art: %w", err)
	}

	file, err := os.ReadFile(procCtx.getCoverArtPath())
	if err != nil {
		slog.Warn("Failed to read cover art", "error", err)
	}

	if len(file) > 0 {
		procCtx.progressCallback(50, "Cover art extracted", file)
	} else {
		procCtx.progressCallback(50, "Cover art not found", nil)
	}

	return nil
}

func (p *processor) processTracks(ctx context.Context, procCtx *processingContext) ([]string, error) {
	bar := progressbar.NewOptions(
		procCtx.setLength,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Processing tracks..."),
	)

	// Use the updated ProcessingOptions with the actual extension
	updatedOpts := *procCtx.opts
	updatedOpts.FileExtension = procCtx.extension

	return p.splitTracks(ctx, procCtx, updatedOpts, bar)
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-", " ", "_")
	return replacer.Replace(title)
}

func (p *processor) splitTracks(
	ctx context.Context,
	procCtx *processingContext,
	opts ProcessingOptions,
	bar *progressbar.ProgressBar,
) ([]string, error) {
	// Configure worker pool
	maxWorkers := p.getMaxWorkers(opts.MaxConcurrentTasks)

	// Setup channels for communication
	errCh := make(chan error, 1)
	filePathCh := make(chan string, procCtx.setLength)
	semaphore := make(chan struct{}, maxWorkers)

	// Start initial progress
	slog.Info("Splitting tracks", "count", procCtx.setLength, "extension", opts.FileExtension)
	procCtx.progressCallback(50, fmt.Sprintf("Processing %d tracks", procCtx.setLength), nil)

	// Start track processing with a pool of workers
	var wg sync.WaitGroup
	var completedTracks int32 // Atomic counter for completed tracks

	// Launch worker goroutines for each track
	for i, t := range procCtx.set.Tracks {
		wg.Add(1)
		t := t // Capture variable
		i := i

		go func(i int, t *domain.Track) {
			defer wg.Done()

			// Check if processing should be canceled
			select {
			case <-ctx.Done():
				select {
				case errCh <- ctx.Err():
				default:
				}
				return
			default:
			}

			// Acquire semaphore slot or abort if canceled
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				select {
				case errCh <- ctx.Err():
				default:
				}
				return
			}
			defer func() { <-semaphore }()

			// Process the track and handle errors
			p.processTrack(
				ctx,
				i, t,
				procCtx,
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
	filePaths := make([]string, 0, procCtx.setLength)
	for path := range filePathCh {
		// Check for cancellation while collecting results
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			filePaths = append(filePaths, path)
		}
	}

	// Check if any errors occurred
	if err := <-errCh; err != nil {
		return nil, err
	}

	slog.Info("Splitting completed")
	procCtx.progressCallback(100, "Processing completed", nil)
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
	trackIndex int,
	track *domain.Track,
	procCtx *processingContext,
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

	// Check for cancellation before starting
	select {
	case <-ctx.Done():
		select {
		case errCh <- ctx.Err():
		default:
		}
		return
	default:
	}

	// Create track name with proper numbering
	safeTitle := sanitizeTitle(track.Title)
	trackName := fmt.Sprintf("%02d-%s", trackNumber, safeTitle)

	// Create a track-specific temp directory for processing
	trackTempDir := filepath.Join(procCtx.getTempDir(), fmt.Sprintf("track_%02d", trackNumber))
	if err := os.MkdirAll(trackTempDir, os.ModePerm); err != nil {
		select {
		case errCh <- fmt.Errorf("failed to create track temp directory for track %d (%s): %w",
			trackNumber, track.Title, err):
		default:
		}
		return
	}

	// Create output paths
	tempOutputPath := filepath.Join(trackTempDir, trackName)
	finalOutputPath := filepath.Join(procCtx.outputDir, trackName)

	// Create parameters for audio processor
	splitParams := audio.SplitParams{
		InputPath:     procCtx.inputFile,
		OutputPath:    tempOutputPath,
		FileExtension: opts.FileExtension,
		Track:         *track,
		TrackCount:    procCtx.setLength,
		Artist:        procCtx.set.Artist,
		Name:          procCtx.set.Name,
		CoverArtPath:  procCtx.getCoverArtPath(),
	}

	// Process the track
	if err := p.audioProcessor.Split(ctx, splitParams); err != nil {
		errorMsg := fmt.Sprintf("Failed to split track %d (%s): %v", trackNumber, track.Title, err)
		slog.Error(errorMsg)
		select {
		case errCh <- fmt.Errorf("failed to process track %d (%s): %w", trackNumber, track.Title, err):
		default:
		}
		return
	}

	// Check for cancellation before moving the file
	select {
	case <-ctx.Done():
		select {
		case errCh <- ctx.Err():
		default:
		}
		return
	default:
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
		default:
		}
		return
	}

	// Write to the final destination
	if err := os.WriteFile(finalFilePath, fileData, 0644); err != nil {
		select {
		case errCh <- fmt.Errorf("failed to write final track file %d (%s): %w",
			trackNumber, track.Title, err):
		default:
		}
		return
	}

	// Update progress
	p.updateTrackProgress(bar, completedTracks, procCtx.setLength, procCtx.progressCallback)

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
	progressCallback func(int, string, []byte),
) {
	// Increment completed tracks atomically
	newCount := atomic.AddInt32(completedTracks, 1)
	trackProgress := int((float64(newCount) / float64(setLength)) * 100)
	totalProgress := 50 + (trackProgress / 2) // Scale to 50-100%
	_ = bar.Add(1)
	progressCallback(totalProgress, fmt.Sprintf("Processed %d/%d tracks", newCount, setLength), nil)
}
