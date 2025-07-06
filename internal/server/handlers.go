package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
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

	// Check if context is cancelled before processing
	select {
	case <-ctx.Done():
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
		return
	}

	// Download the audio file from the URL to the temp directory
	slog.Info("Downloading audio file", "url", url)
	downloadedFile, err := dl.Download(ctx, url, tempDir, nil)
	if err != nil {
		slog.Error("Failed to download audio file", "url", url, "error", err)
		if updateErr := s.jobManager.UpdateJobStatus(jobID, job.StatusFailed, []string{}, err.Error()); updateErr != nil {
			slog.Warn("Failed to update job status to failed", "error", updateErr)
		}
		return
	}

	slog.Info("Successfully downloaded audio file", "downloadedFile", downloadedFile)

	// Split the audio file
	results, err := s.process(ctx, downloadedFile, tracklist, req.MaxConcurrentTasks, tempDir, req.FileExtension)
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
func (s *Server) process(ctx context.Context, inputPath string, tracklist domain.Tracklist, maxConcurrentTasks int, tempDir string, requestedExtension string) ([]string, error) {
	results := make([]string, len(tracklist.Tracks))
	errors := make([]error, len(tracklist.Tracks))

	// First, extract cover art from the original file
	coverArtPath := filepath.Join(tempDir, "cover.jpg")
	if err := s.audioProcessor.ExtractCoverArt(ctx, inputPath, coverArtPath); err != nil {
		slog.Warn("Failed to extract cover art", "error", err)
		// Continue without cover art
		coverArtPath = ""
	}

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
