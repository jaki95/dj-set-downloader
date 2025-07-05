package server

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
)

// downloadAllTracks handles downloading all tracks for a job as a ZIP file
func (s *Server) downloadAllTracks(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, gin.H{"error": "Job is not completed yet"})
		return
	}

	if len(jobStatus.Results) == 0 {
		c.JSON(404, gin.H{"error": "No tracks available for download"})
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
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to add track %d to ZIP: %v", i+1, err)})
			return
		}
	}
}

// downloadTrack handles downloading a single track
func (s *Server) downloadTrack(c *gin.Context) {
	jobID := c.Param("id")
	trackNumberStr := c.Param("trackNumber")

	trackNumber, err := strconv.Atoi(trackNumberStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid track number"})
		return
	}

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, gin.H{"error": "Job is not completed yet"})
		return
	}

	if trackNumber < 1 || trackNumber > len(jobStatus.Results) {
		c.JSON(404, gin.H{"error": "Track not found"})
		return
	}

	trackPath := jobStatus.Results[trackNumber-1]
	track := jobStatus.Tracklist.Tracks[trackNumber-1]

	// Check if file exists
	if _, err := os.Stat(trackPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Track file not found"})
		return
	}

	// Set headers for file download
	fileName := fmt.Sprintf("%02d-%s.%s",
		trackNumber,
		SanitizeFilename(track.Name),
		strings.ToLower(filepath.Ext(trackPath)[1:]))

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	c.File(trackPath)
}

// getTracksInfo handles getting metadata about all tracks in a job
func (s *Server) getTracksInfo(c *gin.Context) {
	jobID := c.Param("id")

	jobStatus, err := s.jobManager.GetJob(jobID)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("%v: %s", job.ErrNotFound, jobID)})
		return
	}

	if jobStatus.Status != job.StatusCompleted {
		c.JSON(400, gin.H{"error": "Job is not completed yet"})
		return
	}

	// Create enhanced tracks with download information
	enhancedTracks := make([]*domain.Track, len(jobStatus.Results))
	for i, trackPath := range jobStatus.Results {
		if i < len(jobStatus.Tracklist.Tracks) {
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
	}

	response := job.TracksInfoResponse{
		JobID:          jobID,
		Tracks:         enhancedTracks,
		TotalTracks:    len(enhancedTracks),
		DownloadAllURL: fmt.Sprintf("/api/jobs/%s/download", jobID),
	}

	c.JSON(200, response)
}
