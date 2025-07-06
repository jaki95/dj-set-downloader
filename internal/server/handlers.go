package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
)

// processWithUrl godoc
// @Summary Start processing a DJ set URL
// @Description Submits a job that downloads and processes the given DJ set URL using the supplied tracklist.
// @Tags Jobs
// @Accept json
// @Produce json
// @Param request body job.Request true "Processing parameters"
// @Success 202 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/process [post]
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

// getJobStatus godoc
// @Summary Get job status
// @Tags Jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} job.Status
// @Failure 404 {object} ErrorResponse
// @Router /api/jobs/{id} [get]
func (s *Server) getJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	c.JSON(200, jobStatus)
}

// cancelJob godoc
// @Summary Cancel a job
// @Tags Jobs
// @Produce json
// @Param id path string true "Job ID"
// @Success 200 {object} MessageResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/jobs/{id}/cancel [post]
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

// listJobs godoc
// @Summary List jobs
// @Tags Jobs
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} job.Response
// @Router /api/jobs [get]
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

// health godoc
// @Summary Health check
// @Tags Utility
// @Produce json
// @Success 200 {object} MessageResponse
// @Router /health [get]
func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
