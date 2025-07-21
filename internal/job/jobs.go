package job

import (
	"context"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/pkg/progress"
)

// Constants for job status
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
	StatusCancelled  = "cancelled"
)

// Constants for progress percentages
const (
	ProgressDownloadStart   = 0
	ProgressDownloadEnd     = 25
	ProgressProcessingStart = 25
	ProgressProcessingEnd   = 99
	ProgressComplete        = 100
)

// Constants for pagination
const (
	DefaultPageSize = 10
	MaxPageSize     = 100
)

// Constants for configuration
const (
	DefaultMaxConcurrentTasks = 4
)

// Status represents the current state of a processing job.
type Status struct {
	ID         string           `json:"id"`
	Status     string           `json:"status"`
	Progress   float64          `json:"progress"`
	Message    string           `json:"message"`
	Error      string           `json:"error,omitempty"`
	Results    []string         `json:"results,omitempty"`
	Events     []progress.Event `json:"events"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    *time.Time       `json:"end_time,omitempty"`
	Tracklist  domain.Tracklist `json:"tracklist"`
	cancelFunc context.CancelFunc
}

// Request represents the request body for processing a URL.
type Request struct {
	URL                string `json:"url" binding:"required"`
	Tracklist          string `json:"tracklist" binding:"required"`
	FileExtension      string `json:"file_extension"`
	MaxConcurrentTasks int    `json:"max_concurrent_tasks"`
}

// Response represents the response for job status.
type Response struct {
	Jobs       []*Status `json:"jobs"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalJobs  int       `json:"total_jobs"`
	TotalPages int       `json:"total_pages"`
}
