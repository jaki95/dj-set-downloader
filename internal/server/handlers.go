package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/service/job"
	"github.com/jaki95/dj-set-downloader/pkg/audio"
	"github.com/jaki95/dj-set-downloader/pkg/downloader"
)

type ProcessResponse struct {
	Message string `json:"message"`
	JobID   string `json:"jobId"`
}

type CancelResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// processWithUrl handles the processing of a URL with a tracklist
//
//	@Summary		Process a DJ set URL with tracklist
//	@Description	Starts processing a DJ set from a given URL using the provided tracklist. Returns a job ID for tracking progress.
//	@Tags			Process
//	@Accept			json
//	@Produce		json
//	@Param			request	body		job.Request		true	"Processing request with URL and tracklist"
//	@Success		202		{object}	ProcessResponse	"Processing started successfully"
//	@Failure		400		{object}	ErrorResponse	"Invalid request or tracklist"
//	@Router			/api/process [post]
func (s *Server) processWithUrl(c *gin.Context) {
	var req job.Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	var tracklist domain.Tracklist
	if err := json.Unmarshal([]byte(req.Tracklist), &tracklist); err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("%v: %v", job.ErrInvalidTracklist, err)})
		return
	}

	if tracklist.Artist == "" || tracklist.Name == "" {
		c.JSON(400, gin.H{"error": fmt.Sprintf("%v: artist and name are required", job.ErrInvalidTracklist)})
		return
	}

	if len(tracklist.Tracks) == 0 {
		c.JSON(400, gin.H{"error": fmt.Sprintf("%v: at least one track is required", job.ErrInvalidTracklist)})
		return
	}

	// Limit track count to prevent memory exhaustion attacks via large slice allocations
	const MaxAllowedTracks = 100
	if len(tracklist.Tracks) > MaxAllowedTracks {
		c.JSON(400, gin.H{"error": fmt.Sprintf("%v: maximum %d tracks allowed", job.ErrInvalidTracklist, MaxAllowedTracks)})
		return
	}

	if req.FileExtension == "" {
		req.FileExtension = "mp3"
	}

	// Validate and sanitize maxConcurrentTasks to prevent excessive memory allocation
	req.MaxConcurrentTasks = job.ValidateMaxConcurrentTasks(req.MaxConcurrentTasks)

	jobStatus, ctx := s.jobManager.CreateJob(tracklist)
	go s.processUrlInBackground(ctx, jobStatus.ID, req.URL, tracklist, req)

	c.JSON(202, ProcessResponse{
		Message: "Processing started",
		JobID:   jobStatus.ID,
	})
}

// processUrlInBackground processes a URL in the background
func (s *Server) processUrlInBackground(ctx context.Context, jobID string, url string, tracklist domain.Tracklist, req job.Request) {
	tempDir := s.createJobTempDir(jobID)

	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Recovered from panic in background process", "panic", r)
			if err := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, "Internal server error"); err != nil {
				slog.Warn("Failed to update job status", "error", err)
			}
			s.cleanupJobTempDir(jobID, tempDir)
		}
	}()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		s.cleanupJobTempDir(jobID, tempDir)
		return
	default:
	}

	// Update job status to processing
	if err := s.jobManager.UpdateJobStatus(jobID, job.StatusProcessing, []string{}, "Processing started"); err != nil {
		slog.Warn("Failed to update job status", "error", err)
	}

	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		s.cleanupJobTempDir(jobID, tempDir)
		return
	default:
	}

	slog.Info("Processing URL", "url", url)

	// Get the appropriate downloader for the URL
	dl, err := downloader.GetDownloader(url)
	if err != nil {
		slog.Error("Failed to get downloader for URL", "url", url, "error", err)
		if updateErr := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, err.Error()); updateErr != nil {
			slog.Warn("Failed to update job status to failed", "error", updateErr)
		}
		s.cleanupJobTempDir(jobID, tempDir)
		return
	}

	// Create progress callback for download
	downloadProgressCallback := func(percent int, message string, data []byte) {
		// Map download progress to overall job progress (0-25%)
		overallProgress := float64(percent) * 0.25
		if err := s.jobManager.UpdateJobProgress(jobID, overallProgress, message); err != nil {
			slog.Warn("Failed to update download progress", "error", err)
		}
	}

	// Download the audio file from the URL to the temp directory
	slog.Info("Downloading audio file", "url", url)
	downloadedFile, err := dl.Download(ctx, url, tempDir, downloadProgressCallback)
	if err != nil {
		slog.Error("Failed to download audio file", "url", url, "error", err)
		if updateErr := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, err.Error()); updateErr != nil {
			slog.Warn("Failed to update job status to failed", "error", updateErr)
		}
		s.cleanupJobTempDir(jobID, tempDir)
		return
	}

	slog.Info("Successfully downloaded audio file", "downloadedFile", downloadedFile)

	// Update progress to processing phase
	if err := s.jobManager.UpdateJobProgress(jobID, 25.0, "Starting audio processing..."); err != nil {
		slog.Warn("Failed to update processing progress", "error", err)
	}

	// Split the audio file
	results, err := s.process(ctx, downloadedFile, tracklist, req.MaxConcurrentTasks, tempDir, req.FileExtension, jobID)
	if err != nil {
		if updateErr := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, err.Error()); updateErr != nil {
			slog.Warn("Failed to update job status to failed", "error", updateErr)
		}
		s.cleanupJobTempDir(jobID, tempDir)
		return
	}

	// Update job status to completed
	if err := s.jobManager.UpdateJobStatus(jobID, job.StatusCompleted, results, "Processing completed"); err != nil {
		slog.Warn("Failed to update job status to completed", "error", err)
	}

	// Schedule cleanup of temporary directory after a grace period
	s.scheduleJobCleanup(jobID, tempDir)
}

// process handles the splitting of audio files
func (s *Server) process(ctx context.Context, inputPath string, tracklist domain.Tracklist, maxConcurrentTasks int, tempDir string, requestedExtension string, jobID string) ([]string, error) {
	results := make([]string, len(tracklist.Tracks))
	errors := make([]error, len(tracklist.Tracks))

	// First, extract cover art from the original file
	coverArtPath := filepath.Join(tempDir, "cover.jpg")
	if err := s.audioProcessor.ExtractCoverArt(ctx, inputPath, coverArtPath); err != nil {
		slog.Warn("Failed to extract cover art", "error", err)
		// Continue without cover art
		coverArtPath = ""
	}

	// Validate and sanitize maxConcurrentTasks to prevent excessive memory allocation
	validatedMaxConcurrentTasks := job.ValidateMaxConcurrentTasks(maxConcurrentTasks)
	if validatedMaxConcurrentTasks != maxConcurrentTasks {
		slog.Warn("MaxConcurrentTasks value was capped for security",
			"requested", maxConcurrentTasks, "capped_to", validatedMaxConcurrentTasks)
	}

	// Create a semaphore to limit concurrent tasks
	semaphore := make(chan struct{}, validatedMaxConcurrentTasks)
	var wg sync.WaitGroup

	// Track actual number of completed tracks (thread-safe)
	var completedTracks int64
	var mu sync.Mutex

	totalTracks := len(tracklist.Tracks)

	for i, track := range tracklist.Tracks {
		wg.Add(1)
		go func(i int, track *domain.Track) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				errors[i] = ctx.Err()
				return
			default:
			}

			// Create output path in temporary directory (without extension)
			outputPath := filepath.Join(tempDir, fmt.Sprintf("%02d-%s",
				i+1,
				SanitizeFilename(track.Name)))

			// Determine the file extension to use
			fileExtension := s.getFileExtension(inputPath)
			if requestedExtension != "" {
				fileExtension = strings.ToLower(requestedExtension)
			}

			// Set up split parameters
			splitParams := audio.SplitParams{
				InputPath:     inputPath,
				OutputPath:    outputPath,
				FileExtension: fileExtension,
				Track:         *track,
				TrackCount:    len(tracklist.Tracks),
				Artist:        tracklist.Artist,
				Name:          tracklist.Name,
				CoverArtPath:  coverArtPath,
			}

			// Actually split the audio file
			if err := s.audioProcessor.Split(ctx, splitParams); err != nil {
				slog.Error("Failed to split audio track", "track", track.Name, "error", err)
				errors[i] = err
				return
			}

			slog.Info("Successfully processed track", "track", track.Name)
			results[i] = outputPath + "." + splitParams.FileExtension

			// Update completed tracks counter (thread-safe)
			mu.Lock()
			completedTracks++
			currentCompleted := completedTracks
			mu.Unlock()

			// Calculate progress based on actual completed tracks
			completion := float64(currentCompleted) / float64(totalTracks)
			overallProgress := 25.0 + (completion * 75.0)
			progressMessage := fmt.Sprintf("Processed track %d/%d: %s", currentCompleted, totalTracks, track.Name)

			if err := s.jobManager.UpdateJobProgress(jobID, overallProgress, progressMessage); err != nil {
				slog.Warn("Failed to update processing progress", "error", err)
			}
		}(i, track)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("failed to process track %d: %w", i+1, err)
		}
	}

	return results, nil
}
