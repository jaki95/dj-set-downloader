// Package server provides HTTP server functionality for the DJ set processor.
// It handles job management, track processing, and provides a REST API for clients.
package server

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/job"
)

// Server handles HTTP requests for the DJ set processor
type Server struct {
	cfg            *config.Config
	jobManager     *job.Manager
	router         *gin.Engine
	audioProcessor audio.Processor
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	return &Server{
		cfg:            cfg,
		jobManager:     job.NewManager(),
		audioProcessor: audio.NewFFMPEGEngine(),
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	router := gin.Default()
	s.router = router
	s.setupRoutes(router)

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(s.cfg.Storage.OutputDir, 0755); err != nil {
		return err
	}

	// Start the file cleanup worker
	s.StartCleanupWorker()

	slog.Info("Starting server", "port", s.cfg.Server.Port)
	return router.Run(":" + s.cfg.Server.Port)
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes(router *gin.Engine) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api")
	{
		api.POST("/process", s.processWithUrl)
		api.GET("/jobs", s.listJobs)
		api.GET("/jobs/:id", s.getJobStatus)
		api.POST("/jobs/:id/cancel", s.cancelJob)

		// Download endpoints
		api.GET("/jobs/:id/download", s.downloadAllTracks)
		api.GET("/jobs/:id/tracks", s.getTracksInfo)
		api.GET("/jobs/:id/tracks/:trackNumber/download", s.downloadTrack)
	}
}
