package server

import (
	"archive/zip"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/service/job"
)

// preValidateZipFiles validates that all files can be opened and ZIP entries can be created
// before starting the ZIP stream to avoid errors after headers are set
func (s *Server) preValidateZipFiles(jobStatus *job.Status) error {
	for i, trackPath := range jobStatus.Results {
		// Check file exists
		if _, err := os.Stat(trackPath); os.IsNotExist(err) {
			return fmt.Errorf("track file %d not found: %s", i+1, trackPath)
		}

		// Check metadata exists
		if i >= len(jobStatus.Tracklist.Tracks) {
			return fmt.Errorf("track %d metadata not found", i+1)
		}

		// Try to open the file to ensure it's accessible
		file, err := os.Open(trackPath)
		if err != nil {
			return fmt.Errorf("cannot open track file %d (%s): %w", i+1, trackPath, err)
		}
		file.Close()

		// Validate ZIP filename creation
		track := jobStatus.Tracklist.Tracks[i]
		ext := s.getFileExtension(trackPath)
		zipFileName := fmt.Sprintf("%02d-%s.%s", i+1, SanitizeFilename(track.Name), ext)

		// Check if the sanitized filename is valid (not empty after sanitization)
		if SanitizeFilename(track.Name) == "" {
			return fmt.Errorf("track %d has invalid name that cannot be sanitized: %s", i+1, track.Name)
		}

		// Additional check: ensure filename is reasonable length
		if len(zipFileName) > 255 {
			return fmt.Errorf("track %d filename too long after sanitization: %s", i+1, zipFileName)
		}
	}

	return nil
}

// downloadAllTracks handles downloading all tracks for a job as a ZIP file
//
//	@Summary		Download all tracks as ZIP
//	@Description	Downloads all processed tracks for a completed job as a ZIP archive
//	@Tags			Downloads
//	@Accept			json
//	@Produce		application/zip
//	@Param			id	path		string			true	"Job ID"
//	@Success		200	{file}		application/zip	"ZIP file containing all tracks"
//	@Failure		400	{object}	ErrorResponse	"Job is not completed yet"
//	@Failure		404	{object}	ErrorResponse	"Job not found or no tracks available"
//	@Failure		500	{object}	ErrorResponse	"Server error during ZIP creation"
//	@Router			/api/jobs/{id}/download [get]
func (s *Server) downloadAllTracks(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, ErrorResponse{Error: fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, ErrorResponse{Error: "Job is not completed yet"})
		return
	}

	if len(jobStatus.Results) == 0 {
		c.JSON(404, ErrorResponse{Error: "No tracks available for download"})
		return
	}

	// Validate that we have corresponding tracks for all results
	if len(jobStatus.Results) > len(jobStatus.Tracklist.Tracks) {
		c.JSON(500, ErrorResponse{Error: "Mismatch between downloaded tracks and tracklist metadata"})
		return
	}

	// Pre-validation before starting ZIP stream
	if err := s.preValidateZipFiles(jobStatus); err != nil {
		slog.Error("ZIP pre-validation failed", "jobID", jobID, "error", err)
		c.JSON(500, ErrorResponse{Error: fmt.Sprintf("Cannot create ZIP archive: %v", err)})
		return
	}

	// Create ZIP file name
	zipFileName := fmt.Sprintf("%s - %s.zip",
		SanitizeFilename(jobStatus.Tracklist.Artist),
		SanitizeFilename(jobStatus.Tracklist.Name))

	// Set headers for ZIP download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	// Create ZIP writer
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	// Add each track to the ZIP
	for i, trackPath := range jobStatus.Results {
		if err := s.addFileToZip(zipWriter, trackPath, i+1, jobStatus.Tracklist.Tracks[i]); err != nil {
			// Log the error with detailed context since we can't send JSON response
			slog.Error("Failed to add file to ZIP after headers were set",
				"jobID", jobID,
				"trackNumber", i+1,
				"trackPath", trackPath,
				"trackName", jobStatus.Tracklist.Tracks[i].Name,
				"error", err)

			// Close the ZIP writer and return - client will receive incomplete ZIP
			// This is unavoidable since headers are already sent
			return
		}
	}

	slog.Info("Successfully created ZIP archive", "jobID", jobID, "trackCount", len(jobStatus.Results))
}

// downloadTrack handles downloading a single track
//
//	@Summary		Download a single track
//	@Description	Downloads a specific processed track by job ID and track number
//	@Tags			Downloads
//	@Accept			json
//	@Produce		audio/mpeg,audio/flac,audio/wav
//	@Param			id			path		string			true	"Job ID"
//	@Param			trackNumber	path		int				true	"Track number (1-based)"
//	@Success		200			{file}		audio/*			"Audio file"
//	@Failure		400			{object}	ErrorResponse	"Invalid track number or job not completed"
//	@Failure		404			{object}	ErrorResponse	"Job or track not found"
//	@Router			/api/jobs/{id}/tracks/{trackNumber}/download [get]
func (s *Server) downloadTrack(c *gin.Context) {
	jobID := c.Param("id")
	trackNumberStr := c.Param("trackNumber")

	trackNumber, err := strconv.Atoi(trackNumberStr)
	if err != nil {
		c.JSON(400, ErrorResponse{Error: "Invalid track number"})
		return
	}

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, ErrorResponse{Error: fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, ErrorResponse{Error: "Job is not completed yet"})
		return
	}

	if trackNumber < 1 || trackNumber > len(jobStatus.Results) || trackNumber > len(jobStatus.Tracklist.Tracks) {
		c.JSON(404, ErrorResponse{Error: "Track not found"})
		return
	}

	trackPath := jobStatus.Results[trackNumber-1]
	track := jobStatus.Tracklist.Tracks[trackNumber-1]

	// Check if file exists
	if _, err := os.Stat(trackPath); os.IsNotExist(err) {
		c.JSON(404, ErrorResponse{Error: "Track file not found"})
		return
	}

	// Set headers for file download
	fileName := fmt.Sprintf("%02d-%s.%s",
		trackNumber,
		SanitizeFilename(track.Name),
		s.getFileExtension(trackPath))

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.File(trackPath)
}

// getTracksInfo handles getting metadata about all tracks in a job
//
//	@Summary		Get tracks information
//	@Description	Retrieves metadata and download information for all tracks in a completed job
//	@Tags			Downloads
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Job ID"
//	@Success		200	{object}	job.TracksInfoResponse	"Tracks information retrieved successfully"
//	@Failure		400	{object}	ErrorResponse			"Job is not completed yet"
//	@Failure		404	{object}	ErrorResponse			"Job not found"
//	@Router			/api/jobs/{id}/tracks [get]
func (s *Server) getTracksInfo(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, ErrorResponse{Error: fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, ErrorResponse{Error: "Job is not completed yet"})
		return
	}

	// Create enhanced tracks with download information
	// Use the minimum of both lengths to avoid nil elements
	maxTracks := len(jobStatus.Results)
	if len(jobStatus.Tracklist.Tracks) < maxTracks {
		maxTracks = len(jobStatus.Tracklist.Tracks)
	}

	enhancedTracks := make([]*domain.Track, maxTracks)
	for i := 0; i < maxTracks; i++ {
		trackPath := jobStatus.Results[i]
		// Create a copy to avoid modifying the original
		track := *jobStatus.Tracklist.Tracks[i]

		// Get file info
		fileInfo, err := os.Stat(trackPath)
		available := err == nil
		var sizeBytes int64 = 0
		if available {
			sizeBytes = fileInfo.Size()
		}

		// Populate download fields
		track.DownloadURL = fmt.Sprintf("/api/jobs/%s/tracks/%d/download", jobID, i+1)
		track.SizeBytes = sizeBytes
		track.Available = available

		enhancedTracks[i] = &track
	}

	response := job.TracksInfoResponse{
		JobID:          jobID,
		Tracks:         enhancedTracks,
		TotalTracks:    len(enhancedTracks),
		DownloadAllURL: fmt.Sprintf("/api/jobs/%s/download", jobID),
	}

	c.JSON(200, response)
}
