package job

import (
	"context"
	"fmt"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// Manager handles job management
type Manager struct {
	jobs map[string]*Status
}

// NewManager creates a new job manager
func NewManager() *Manager {
	return &Manager{
		jobs: make(map[string]*Status),
	}
}

// CreateJob creates a new job
func (m *Manager) CreateJob(tracklist domain.Tracklist) (*Status, context.Context) {
	jobID := fmt.Sprintf("%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())

	job := &Status{
		ID:         jobID,
		Status:     StatusPending,
		Progress:   0,
		Message:    "Job created",
		StartTime:  time.Now(),
		Tracklist:  tracklist,
		cancelFunc: cancel,
	}

	m.jobs[jobID] = job
	return job, ctx
}

// GetJob retrieves a job by ID
func (m *Manager) GetJob(jobID string) (*Status, error) {
	job, exists := m.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, jobID)
	}
	return job, nil
}

// CancelJob cancels a job
func (m *Manager) CancelJob(jobID string) error {
	job, err := m.GetJob(jobID)
	if err != nil {
		return err
	}

	if job.Status != StatusProcessing && job.Status != StatusPending {
		return fmt.Errorf("%w: %s", ErrInvalidState, job.Status)
	}

	job.cancelFunc()
	job.Status = StatusCancelled
	job.Message = "Job cancelled by user"
	endTime := time.Now()
	job.EndTime = &endTime

	return nil
}

// ListJobs lists all jobs with pagination
func (m *Manager) ListJobs(page, pageSize int) *Response {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > MaxPageSize {
		pageSize = DefaultPageSize
	}

	jobs := make([]*Status, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobs = append(jobs, job)
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(jobs) {
		return &Response{
			Jobs:       []*Status{},
			Page:       page,
			PageSize:   pageSize,
			TotalJobs:  len(jobs),
			TotalPages: (len(jobs) + pageSize - 1) / pageSize,
		}
	}

	if end > len(jobs) {
		end = len(jobs)
	}

	return &Response{
		Jobs:       jobs[start:end],
		Page:       page,
		PageSize:   pageSize,
		TotalJobs:  len(jobs),
		TotalPages: (len(jobs) + pageSize - 1) / pageSize,
	}
}
