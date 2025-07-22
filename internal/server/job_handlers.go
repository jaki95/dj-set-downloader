package server

import (
	"fmt"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/service/job"
)

// getJobStatus handles retrieving the status of a job
//
//	@Summary		Get job status
//	@Description	Retrieves the current status and progress of a processing job by ID
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string			true	"Job ID"
//	@Success		200	{object}	job.Status		"Job status retrieved successfully"
//	@Failure		404	{object}	ErrorResponse	"Job not found"
//	@Router			/api/jobs/{id} [get]
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
		for i, track := range tracklistCopy.Tracks {
			// Create a copy of each track to avoid modifying the original
			trackCopy := *track
			tracksCopy[i] = &trackCopy
		}
		tracklistCopy.Tracks = tracksCopy

		// Replace the jobStatus tracklist with the defensive copy
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
//
//	@Summary		Cancel a job
//	@Description	Cancels a running or pending processing job by ID
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string			true	"Job ID"
//	@Success		200	{object}	CancelResponse	"Job cancelled successfully"
//	@Failure		404	{object}	ErrorResponse	"Job not found"
//	@Failure		400	{object}	ErrorResponse	"Job cannot be cancelled (invalid state)"
//	@Failure		500	{object}	ErrorResponse	"Internal server error"
//	@Router			/api/jobs/{id}/cancel [post]
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

	c.JSON(200, CancelResponse{Message: "Job cancelled"})
}

// listJobs handles listing all jobs
//
//	@Summary		List all jobs
//	@Description	Retrieves a paginated list of all processing jobs
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int				false	"Page number"						default(1)
//	@Param			pageSize	query		int				false	"Number of jobs per page (max 100)"	default(10)
//	@Success		200			{object}	job.Response	"Jobs retrieved successfully"
//	@Router			/api/jobs [get]
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
