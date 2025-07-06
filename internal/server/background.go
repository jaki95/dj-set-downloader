package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
	"github.com/jaki95/dj-set-downloader/pkg/progress"
)

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
	results, err := s.processor.Process(ctx, &tracklist, downloadURL, req.FileExtension, req.MaxConcurrentTasks, progressCallback)

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