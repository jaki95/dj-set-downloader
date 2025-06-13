package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/progress"
)

// Server handles HTTP requests for the DJ set processor
type Server struct {
	cfg    *config.Config
	router *gin.Engine

	// Active processing jobs (in a real implementation, this would be a proper job queue)
	activeJobs map[string]*JobStatus
}

// JobStatus tracks the status of a processing job
type JobStatus struct {
	ID         string             `json:"id"`
	Status     string             `json:"status"` // pending, processing, completed, failed
	Progress   float64            `json:"progress"`
	Message    string             `json:"message"`
	StartTime  time.Time          `json:"startTime"`
	EndTime    *time.Time         `json:"endTime,omitempty"`
	Error      string             `json:"error,omitempty"`
	Results    []string           `json:"results,omitempty"`
	Events     []progress.Event   `json:"events,omitempty"`
	cancelFunc context.CancelFunc `json:"-"`
}

// ProcessUrlRequest represents processing with download URL and tracklist
type ProcessUrlRequest struct {
	DownloadURL        string `json:"downloadUrl" binding:"required"`
	Tracklist          string `json:"tracklist" binding:"required"` // JSON string
	FileExtension      string `json:"fileExtension,omitempty"`
	MaxConcurrentTasks int    `json:"maxConcurrentTasks,omitempty"`
}

// ProcessRequest represents a request to process a DJ set
type ProcessRequest struct {
	Query              string `json:"query" binding:"required"`
	FileExtension      string `json:"fileExtension,omitempty"`
	MaxConcurrentTasks int    `json:"maxConcurrentTasks,omitempty"`
}

// New creates a new HTTP server instance
func New(cfg *config.Config) (*Server, error) {
	// Set gin mode (default to debug mode)
	gin.SetMode(gin.DebugMode)

	router := gin.Default()

	server := &Server{
		cfg:        cfg,
		router:     router,
		activeJobs: make(map[string]*JobStatus),
	}

	server.setupRoutes()
	return server, nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Add CORS middleware
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// API endpoints
	api := s.router.Group("/api/v1")
	{
		api.POST("/process", s.processWithUrl) // Main endpoint (URL + tracklist)
		api.GET("/jobs/:id", s.getJobStatus)
		api.DELETE("/jobs/:id", s.cancelJob)
		api.GET("/jobs", s.listJobs)
	}
}

// Start starts the HTTP server
func (s *Server) Start(port string) error {
	return s.router.Run(":" + port)
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"service":   "dj-set-processor",
	})
}

// processWithUrl handles track processing with download URL and tracklist
func (s *Server) processWithUrl(c *gin.Context) {
	var req ProcessUrlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Parse the tracklist
	var tracklist domain.Tracklist
	if err := json.Unmarshal([]byte(req.Tracklist), &tracklist); err != nil {
		c.JSON(400, gin.H{"error": "Invalid tracklist JSON: " + err.Error()})
		return
	}

	// Set defaults
	if req.FileExtension == "" {
		req.FileExtension = s.cfg.FileExtension
	}
	if req.MaxConcurrentTasks == 0 {
		req.MaxConcurrentTasks = 4
	}

	// Generate job ID
	jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())

	// Create job status
	ctx, cancel := context.WithCancel(context.Background())
	job := &JobStatus{
		ID:         jobID,
		Status:     "pending",
		Progress:   0,
		Message:    "Job queued",
		StartTime:  time.Now(),
		cancelFunc: cancel,
		Events:     make([]progress.Event, 0),
	}

	s.activeJobs[jobID] = job

	// Start processing in background
	go s.processUrlInBackground(ctx, jobID, req.DownloadURL, tracklist, req)

	c.JSON(202, gin.H{
		"jobId":   jobID,
		"status":  "accepted",
		"message": "Processing started",
	})
}

// processUrlInBackground handles the actual track processing with download URL
func (s *Server) processUrlInBackground(ctx context.Context, jobID string, downloadURL string, tracklist domain.Tracklist, req ProcessUrlRequest) {
	job := s.activeJobs[jobID]
	job.Status = "processing"
	job.Message = "Starting download and processing"

	// Progress callback to update job status
	progressCallback := func(progressPercent int, message string, data []byte) {
		job.Progress = float64(progressPercent)
		job.Message = message

		// Add event to history
		event := progress.Event{
			Stage:     progress.StageProcessing,
			Progress:  float64(progressPercent),
			Message:   message,
			Timestamp: time.Now(),
		}
		if data != nil {
			event.Data = data
		}
		job.Events = append(job.Events, event)

		slog.Debug("Job progress update", "jobId", jobID, "progress", progressPercent, "message", message)
	}

	// Add track numbers to the tracklist
	for i, track := range tracklist.Tracks {
		track.TrackNumber = i + 1
	}

	// Process directly
	results, err := s.processDirectly(ctx, &tracklist, downloadURL, req.FileExtension, req.MaxConcurrentTasks, progressCallback)

	endTime := time.Now()
	job.EndTime = &endTime

	if err != nil {
		if ctx.Err() == context.Canceled {
			job.Status = "cancelled"
			job.Message = "Processing was cancelled"
		} else {
			job.Status = "failed"
			job.Error = err.Error()
			job.Message = "Processing failed"
		}
		slog.Error("Job failed", "jobId", jobID, "error", err)
	} else {
		job.Status = "completed"
		job.Progress = 100
		job.Message = "Processing completed successfully"
		job.Results = results
		slog.Info("Job completed", "jobId", jobID, "results", len(results))
	}
}

// processDirectly handles download and processing without any legacy dependencies
func (s *Server) processDirectly(ctx context.Context, tracklist *domain.Tracklist, downloadURL, fileExtension string, maxConcurrentTasks int, progressCallback func(int, string, []byte)) ([]string, error) {
	// Create temp directory
	tempDir := filepath.Join(os.TempDir(), "djset-server", fmt.Sprintf("direct_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download the audio file
	progressCallback(10, "Downloading audio file...", nil)
	dl, err := downloader.GetDownloader(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get downloader: %w", err)
	}
	downloadedFile, err := dl.Download(ctx, downloadURL, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	// Create output directory
	outputDir := filepath.Join(s.cfg.Storage.OutputDir, fmt.Sprintf("%s - %s", tracklist.Artist, tracklist.Name))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Process tracks directly
	return s.splitTracksDirectly(ctx, downloadedFile, tracklist, outputDir, "", fileExtension, progressCallback)
}

// splitTracksDirectly handles the track splitting
func (s *Server) splitTracksDirectly(ctx context.Context, inputFilePath string, tracklist *domain.Tracklist, outputDir, coverArtPath, fileExtension string, progressCallback func(int, string, []byte)) ([]string, error) {
	// Create audio processor instance
	audioProcessor := audio.NewFFMPEGEngine()

	// Try to extract cover art (optional)
	if coverArtPath == "" {
		coverArtPath = filepath.Join(filepath.Dir(inputFilePath), "cover.jpg")
	}

	// Extract cover art and only use it if successful
	if err := audioProcessor.ExtractCoverArt(ctx, inputFilePath, coverArtPath); err != nil {
		// Cover art extraction failed, don't use cover art
		coverArtPath = ""
	}

	var results []string
	totalTracks := len(tracklist.Tracks)

	progressCallback(20, "Starting track processing...", nil)

	for i, track := range tracklist.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		trackNumber := i + 1
		safeTitle := sanitizeTitle(track.Name)
		trackName := fmt.Sprintf("%02d-%s", trackNumber, safeTitle)
		outputPath := filepath.Join(outputDir, trackName)

		// Create split parameters
		splitParams := audio.SplitParams{
			InputPath:     inputFilePath,
			OutputPath:    outputPath,
			FileExtension: fileExtension,
			Track:         *track,
			TrackCount:    totalTracks,
			Artist:        tracklist.Artist,
			Name:          tracklist.Name,
			CoverArtPath:  coverArtPath,
		}

		// Process the track
		if err := audioProcessor.Split(ctx, splitParams); err != nil {
			return nil, fmt.Errorf("failed to process track %d (%s): %w", trackNumber, track.Name, err)
		}

		// Calculate progress
		progress := 20 + int(float64(trackNumber)/float64(totalTracks)*70) // 20-90%
		progressCallback(progress, fmt.Sprintf("Processed track %d/%d: %s", trackNumber, totalTracks, track.Name), nil)

		// Add result
		finalPath := fmt.Sprintf("%s.%s", outputPath, fileExtension)
		results = append(results, finalPath)
	}

	progressCallback(100, "Processing completed", nil)
	return results, nil
}

// sanitizeTitle removes invalid characters from track titles
func sanitizeTitle(title string) string {
	// Simple sanitization - remove/replace invalid filename characters
	result := ""
	for _, char := range title {
		switch char {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			result += "_"
		default:
			result += string(char)
		}
	}
	return result
}

// getJobStatus handles job status requests
func (s *Server) getJobStatus(c *gin.Context) {
	jobID := c.Param("id")

	job, exists := s.activeJobs[jobID]
	if !exists {
		c.JSON(404, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(200, job)
}

// cancelJob handles job cancellation requests
func (s *Server) cancelJob(c *gin.Context) {
	jobID := c.Param("id")

	job, exists := s.activeJobs[jobID]
	if !exists {
		c.JSON(404, gin.H{"error": "Job not found"})
		return
	}

	if job.Status == "processing" || job.Status == "pending" {
		job.cancelFunc()
		job.Status = "cancelled"
		job.Message = "Job cancelled by user"
		endTime := time.Now()
		job.EndTime = &endTime

		c.JSON(200, gin.H{"message": "Job cancelled"})
	} else {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Cannot cancel job in status: %s", job.Status)})
	}
}

// listJobs handles listing all jobs
func (s *Server) listJobs(c *gin.Context) {
	// Optional pagination
	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	// Convert map to slice for easier manipulation
	jobs := make([]*JobStatus, 0, len(s.activeJobs))
	for _, job := range s.activeJobs {
		jobs = append(jobs, job)
	}

	// Simple pagination
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= len(jobs) {
		c.JSON(200, gin.H{
			"jobs":       []*JobStatus{},
			"page":       page,
			"pageSize":   pageSize,
			"totalJobs":  len(jobs),
			"totalPages": (len(jobs) + pageSize - 1) / pageSize,
		})
		return
	}

	if end > len(jobs) {
		end = len(jobs)
	}

	paginatedJobs := jobs[start:end]

	c.JSON(200, gin.H{
		"jobs":       paginatedJobs,
		"page":       page,
		"pageSize":   pageSize,
		"totalJobs":  len(jobs),
		"totalPages": (len(jobs) + pageSize - 1) / pageSize,
	})
}
