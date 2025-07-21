package job

import (
	"context"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/pkg/progress"
)

// Status represents the current state of a processing job
type Status struct {
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

	// Additional fields from main branch
	DownloadAllURL string `json:"download_all_url,omitempty"`
	TotalTracks    int    `json:"total_tracks,omitempty"`

	cancelFunc context.CancelFunc `json:"-"`
}

// Request represents the request body for processing a URL
type Request struct {
	URL                string `json:"url" binding:"required"`
	Tracklist          string `json:"tracklist" binding:"required"`
	FileExtension      string `json:"file_extension"`
	MaxConcurrentTasks int    `json:"max_concurrent_tasks"`
}

// Response represents the response for job status
type Response struct {
	Jobs       []*Status `json:"jobs"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalJobs  int       `json:"total_jobs"`
	TotalPages int       `json:"total_pages"`
}

// TracksInfoResponse represents the response for tracks info endpoint
type TracksInfoResponse struct {
	JobID          string          `json:"job_id"`
	Tracks         []*domain.Track `json:"tracks"`
	TotalTracks    int             `json:"total_tracks"`
	DownloadAllURL string          `json:"download_all_url"`
}

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
	MaxAllowedConcurrentTasks = 100 // Safety limit to prevent excessive memory allocation
)

// ValidateMaxConcurrentTasks validates and sanitizes the maxConcurrentTasks value
// to prevent excessive memory allocation attacks
func ValidateMaxConcurrentTasks(maxConcurrentTasks int) int {
	if maxConcurrentTasks <= 0 {
		return DefaultMaxConcurrentTasks
	}
	if maxConcurrentTasks > MaxAllowedConcurrentTasks {
		return MaxAllowedConcurrentTasks
	}
	return maxConcurrentTasks
}
