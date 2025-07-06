package server

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
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

	const MaxAllowedTracks = 100 // Security: Limit track count to prevent memory exhaustion attacks via large slice allocations
	if len(tracklist.Tracks) > MaxAllowedTracks {
		c.JSON(400, gin.H{"error": fmt.Sprintf("%v: maximum %d tracks allowed", job.ErrInvalidTracklist, MaxAllowedTracks)})
		return
	}

	if req.FileExtension == "" {
		req.FileExtension = "mp3"
	}

	const MaxAllowedConcurrentTasks = 10 // Define a reasonable upper limit to prevent memory exhaustion
	if req.MaxConcurrentTasks <= 0 {
		req.MaxConcurrentTasks = job.DefaultMaxConcurrentTasks
	} else if req.MaxConcurrentTasks > MaxAllowedConcurrentTasks {
		req.MaxConcurrentTasks = MaxAllowedConcurrentTasks
	}

	jobStatus, ctx := s.jobManager.CreateJob(tracklist)
	go s.processUrlInBackground(ctx, jobStatus.ID, req.URL, tracklist, req)

	c.JSON(202, gin.H{
		"message": "Processing started",
		"jobId":   jobStatus.ID,
	})
}

// getJobStatus handles retrieving the status of a job
func (s *Server) getJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	originalJobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	// Create a copy of the job status to avoid race conditions
	jobStatus := *originalJobStatus
	
	// Enhance the job status with download information if job is completed
	if jobStatus.Status == job.StatusCompleted && len(jobStatus.Results) > 0 {
		jobStatus.DownloadAllURL = fmt.Sprintf("/api/jobs/%s/download", jobID)
		jobStatus.TotalTracks = len(jobStatus.Results)

		// Create a copy of the tracklist to avoid modifying the original
		tracklistCopy := jobStatus.Tracklist
		tracksCopy := make([]*domain.Track, len(tracklistCopy.Tracks))
		copy(tracksCopy, tracklistCopy.Tracks)
		tracklistCopy.Tracks = tracksCopy
		jobStatus.Tracklist = tracklistCopy

		// Enhance tracks with download information
		for i, trackPath := range jobStatus.Results {
			if i < len(jobStatus.Tracklist.Tracks) {
				// Get file info
				fileInfo, err := os.Stat(trackPath)
				available := err == nil
				var sizeBytes int64 = 0
				if available {
					sizeBytes = fileInfo.Size()
				}

				// Populate download fields on the copied track
				jobStatus.Tracklist.Tracks[i].DownloadURL = fmt.Sprintf("/api/jobs/%s/tracks/%d/download", jobID, i+1)
				jobStatus.Tracklist.Tracks[i].SizeBytes = sizeBytes
				jobStatus.Tracklist.Tracks[i].Available = available
			}
		}
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
