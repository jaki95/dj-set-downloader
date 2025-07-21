package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/service/job"
	"github.com/jaki95/dj-set-downloader/internal/pkg/progress"
)

// processWithUrl handles the processing of a URL with a tracklist
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

	if req.FileExtension == "" {
		req.FileExtension = "mp3"
	}

	if req.MaxConcurrentTasks <= 0 {
		req.MaxConcurrentTasks = job.DefaultMaxConcurrentTasks
	}

	jobStatus, ctx := s.jobManager.CreateJob(tracklist)
	go s.processUrlInBackground(ctx, jobStatus.ID, req.URL, tracklist, req)

	c.JSON(202, gin.H{
		"message": "Processing started",
		"jobId":   jobStatus.ID,
	})
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

// getJobStatus handles retrieving the status of a job
func (s *Server) getJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	c.JSON(200, jobStatus)
}

// cancelJob handles cancelling a job
func (s *Server) cancelJob(c *gin.Context) {
	jobID := c.Param("id")

	if err := s.jobManager.CancelJob(jobID); err != nil {
		switch err {
		case job.ErrNotFound:
			c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		case job.ErrInvalidState:
			c.JSON(400, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrInvalidState, jobID)})
		default:
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(200, gin.H{"message": "Job cancelled"})
}

// listJobs handles listing all jobs
func (s *Server) listJobs(c *gin.Context) {
	page := 1
	pageSize := job.DefaultPageSize

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= job.MaxPageSize {
			pageSize = parsed
		}
	}

	response := s.jobManager.ListJobs(page, pageSize)
	c.JSON(200, response)
}
