package server

import (
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/progress"
)

// JobStatus represents the current state of a processing job
type JobStatus struct {
	ID        string           `json:"id"`
	Status    string           `json:"status"`
	Progress  float64          `json:"progress"`
	Message   string           `json:"message"`
	Error     string           `json:"error,omitempty"`
	Results   []string         `json:"results,omitempty"`
	Events    []progress.Event `json:"events"`
	StartTime time.Time        `json:"start_time"`
	EndTime   *time.Time       `json:"end_time,omitempty"`
	Tracklist domain.Tracklist `json:"tracklist"`
}

// ProcessUrlRequest represents the request body for processing a URL
type ProcessUrlRequest struct {
	URL                string `json:"url" binding:"required"`
	Tracklist          string `json:"tracklist" binding:"required"`
	FileExtension      string `json:"file_extension"`
	MaxConcurrentTasks int    `json:"max_concurrent_tasks"`
}

// JobStatusResponse represents the response for job status
type JobStatusResponse struct {
	Jobs       []*JobStatus `json:"jobs"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
	TotalJobs  int          `json:"total_jobs"`
	TotalPages int          `json:"total_pages"`
}
