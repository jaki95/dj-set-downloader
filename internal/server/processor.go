package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/pkg/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/pkg/downloader"
	"github.com/jaki95/dj-set-downloader/internal/job"
	"github.com/jaki95/dj-set-downloader/pkg/progress"
)

// process handles download and processing
func (s *Server) process(ctx context.Context, tracklist *domain.Tracklist, downloadURL, fileExtension string, maxConcurrentTasks int, progressCallback func(int, string, []byte)) ([]string, error) {
	slog.Info("Starting process", "url", downloadURL, "extension", fileExtension, "trackCount", len(tracklist.Tracks))

	tempDir := filepath.Join(os.TempDir(), "djset-server", fmt.Sprintf("direct_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		slog.Error("Failed to create temp directory", "tempDir", tempDir, "error", err)
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	slog.Info("Created temp directory", "tempDir", tempDir)

	// Create a download progress callback that maps to the download range
	downloadProgressCallback := func(percent int, message string, data []byte) {
		// Map download progress (0-100%) to overall progress using constants
		downloadRange := float64(job.ProgressDownloadEnd - job.ProgressDownloadStart)
		mappedPercent := job.ProgressDownloadStart + int(float64(percent)*downloadRange/100.0)
		progressCallback(mappedPercent, message, data)
	}

	slog.Info("Getting downloader for URL", "url", downloadURL)
	dl, err := downloader.GetDownloader(downloadURL)
	if err != nil {
		slog.Error("Failed to get downloader", "url", downloadURL, "error", err)
		return nil, fmt.Errorf("failed to get downloader: %w", err)
	}
	slog.Info("Got downloader, starting download", "url", downloadURL)

	downloadedFile, err := dl.Download(ctx, downloadURL, tempDir, downloadProgressCallback)
	if err != nil {
		slog.Error("Download failed", "url", downloadURL, "error", err)
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}
	slog.Info("Download completed", "file", downloadedFile)

	if strings.Contains(tracklist.Artist, "/") || strings.Contains(tracklist.Artist, "\\") || strings.Contains(tracklist.Artist, "..") ||
		strings.Contains(tracklist.Name, "/") || strings.Contains(tracklist.Name, "\\") || strings.Contains(tracklist.Name, "..") {
		slog.Error("Unsafe characters in tracklist", "artist", tracklist.Artist, "name", tracklist.Name)
		return nil, fmt.Errorf("artist or name contains unsafe characters")
	}

	outputDir := filepath.Join(s.cfg.Storage.OutputDir, fmt.Sprintf("%s - %s", tracklist.Artist, tracklist.Name))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("Failed to create output directory", "outputDir", outputDir, "error", err)
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	slog.Info("Created output directory", "outputDir", outputDir)

	slog.Info("Starting track splitting", "inputFile", downloadedFile, "outputDir", outputDir)
	return s.splitTracks(ctx, downloadedFile, tracklist, outputDir, "", fileExtension, progressCallback)
}

// splitTracks handles the track splitting
func (s *Server) splitTracks(ctx context.Context, inputFilePath string, tracklist *domain.Tracklist, outputDir, coverArtPath, fileExtension string, progressCallback func(int, string, []byte)) ([]string, error) {
	audioProcessor := audio.NewFFMPEGEngine()

	if coverArtPath == "" {
		coverArtPath = filepath.Join(filepath.Dir(inputFilePath), "cover.jpg")
	}

	if err := audioProcessor.ExtractCoverArt(ctx, inputFilePath, coverArtPath); err != nil {
		coverArtPath = ""
	}

	var results []string
	totalTracks := len(tracklist.Tracks)

	progressCallback(job.ProgressProcessingStart, "Starting track processing...", nil)

	for i, track := range tracklist.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		trackNumber := i + 1
		trackName := fmt.Sprintf("%02d-%s", trackNumber, sanitizeFilename(track.Name))
		outputPath := filepath.Join(outputDir, trackName)

		splitParams := audio.SplitParams{
			InputPath:     inputFilePath,
			OutputPath:    outputPath,
			FileExtension: fileExtension,
			Track:         *track,
			TrackCount:    totalTracks,
			Artist:        tracklist.Artist,
			Name:          tracklist.Name,
			CoverArtPath:  coverArtPath,
		}

		if err := audioProcessor.Split(ctx, splitParams); err != nil {
			return nil, fmt.Errorf("failed to process track %d (%s): %w", trackNumber, track.Name, err)
		}

		progress := job.ProgressProcessingStart + int(float64(trackNumber)/float64(totalTracks)*float64(job.ProgressProcessingEnd-job.ProgressProcessingStart))
		progressCallback(progress, fmt.Sprintf("Processed track %d/%d: %s", trackNumber, totalTracks, track.Name), nil)

		finalPath := fmt.Sprintf("%s.%s", outputPath, fileExtension)
		results = append(results, finalPath)
	}

	progressCallback(job.ProgressComplete, "Processing completed", nil)
	return results, nil
}

// processUrlInBackground handles the background processing of a URL
func (s *Server) processUrlInBackground(ctx context.Context, jobID string, downloadURL string, tracklist domain.Tracklist, req job.Request) {
	slog.Info("Starting background processing", "jobId", jobID, "url", downloadURL)

	// Add timeout to prevent jobs from hanging indefinitely
	ctx, cancel := context.WithTimeout(ctx, 45*time.Minute)
	defer cancel()

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		slog.Error("Job failed to retrieve", "jobId", jobID, "error", err)
		return
	}

	// Initial status update
	jobStatus.Status = job.StatusProcessing
	jobStatus.Message = "Starting download and processing"
	slog.Info("Job status updated to processing", "jobId", jobID)

	progressCallback := func(progressPercent int, message string, data []byte) {
		jobStatus.Progress = float64(progressPercent)
		jobStatus.Message = message

		event := progress.Event{
			Stage:     progress.StageProcessing,
			Progress:  float64(progressPercent),
			Message:   message,
			Timestamp: time.Now(),
		}
		if data != nil {
			event.Data = data
		}
		jobStatus.Events = append(jobStatus.Events, event)

		slog.Info("Job progress update", "jobId", jobID, "progress", progressPercent, "message", message)
	}

	// Set track numbers
	for i, track := range tracklist.Tracks {
		track.TrackNumber = i + 1
	}

	slog.Info("Starting download and processing", "jobId", jobID, "trackCount", len(tracklist.Tracks))
	results, err := s.process(ctx, &tracklist, downloadURL, req.FileExtension, req.MaxConcurrentTasks, progressCallback)

	endTime := time.Now()
	jobStatus.EndTime = &endTime

	if err != nil {
		if ctx.Err() == context.Canceled {
			jobStatus.Status = job.StatusCancelled
			jobStatus.Message = "Processing was cancelled"
			slog.Warn("Job cancelled", "jobId", jobID)
		} else {
			jobStatus.Status = job.StatusFailed
			jobStatus.Error = err.Error()
			jobStatus.Message = "Processing failed"
			slog.Error("Job failed", "jobId", jobID, "error", err)
		}
	} else {
		jobStatus.Status = job.StatusCompleted
		jobStatus.Progress = float64(job.ProgressComplete)
		jobStatus.Message = "Processing completed successfully"
		jobStatus.Results = results
		slog.Info("Job completed successfully", "jobId", jobID, "results", len(results))
	}
}

func sanitizeFilename(name string) string {
	// Replace unsafe characters with underscores
	unsafe := []string{"/", "\\", "..", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Also remove any leading/trailing spaces
	result = strings.TrimSpace(result)
	return result
}
