package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
)

// processUrlInBackground processes a URL in the background
func (s *Server) processUrlInBackground(ctx context.Context, jobID string, url string, tracklist domain.Tracklist, req job.Request) {
	defer func() {
		if r := recover(); r != nil {
			slog.Debug("Recovered from panic in background process", "panic", r)
			if err := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, "Internal server error"); err != nil {
				slog.Warn("Failed to update job status", "error", err)
			}
		}
	}()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return
	default:
	}

	// Create temporary directory for this job
	tempDir := s.createJobTempDir(jobID)
	defer func() {
		// Clean up temp directory on exit
		if err := os.RemoveAll(tempDir); err != nil {
			slog.Warn("Failed to clean up temp directory", "path", tempDir, "error", err)
		}
	}()

	// Update job status to processing
	if err := s.jobManager.UpdateJobStatus(jobID, job.StatusProcessing, []string{}, ""); err != nil {
		slog.Warn("Failed to update job status", "error", err)
	}

	// Process the URL
	outputPath := filepath.Join(tempDir, fmt.Sprintf("%s_%s.%s",
		SanitizeFilename(tracklist.Artist),
		SanitizeFilename(tracklist.Name),
		req.FileExtension))

	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
		return
	default:
	}

	slog.Info("Processing URL", "url", url)

	// Split the audio file
	results, err := s.process(ctx, outputPath, tracklist, req.MaxConcurrentTasks, tempDir)
	if err != nil {
		if updateErr := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, err.Error()); updateErr != nil {
			slog.Warn("Failed to update job status to failed", "error", updateErr)
		}
		return
	}

	// Update job status to completed
	if err := s.jobManager.UpdateJobStatus(jobID, job.StatusCompleted, results, ""); err != nil {
		slog.Warn("Failed to update job status to completed", "error", err)
	}
}

// process handles the splitting of audio files
func (s *Server) process(ctx context.Context, inputPath string, tracklist domain.Tracklist, maxConcurrentTasks int, tempDir string) ([]string, error) {
	results := make([]string, len(tracklist.Tracks))
	errors := make([]error, len(tracklist.Tracks))

	// Create a semaphore to limit concurrent tasks
	semaphore := make(chan struct{}, maxConcurrentTasks)
	var wg sync.WaitGroup

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

			// Create output path in temporary directory
			outputPath := filepath.Join(tempDir, fmt.Sprintf("%02d-%s.%s",
				i+1,
				SanitizeFilename(track.Name),
				strings.ToLower(filepath.Ext(inputPath)[1:])))

			// TODO: Implement actual splitting
			// For now, create placeholder files
			log.Printf("Processing track %d: %s", i+1, track.Name)

			results[i] = outputPath
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
