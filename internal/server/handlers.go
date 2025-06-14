package server

import (
	"encoding/json"
	"fmt"
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
		if err == job.ErrNotFound {
			c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		} else if err == job.ErrInvalidState {
			c.JSON(400, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrInvalidState, jobID)})
		} else {
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
