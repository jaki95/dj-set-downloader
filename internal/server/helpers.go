package server

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

const (
	// Grace period before cleaning up completed job files
	// This allows users time to download their files after processing completes
	JobCleanupGracePeriod = 2 * time.Hour
)


// SanitizeFilename sanitizes a filename by removing invalid characters
func SanitizeFilename(name string) string {
	// Replace invalid characters with underscores
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Remove leading and trailing spaces and dots
	result = strings.Trim(result, " .")

	// Ensure the filename is not empty
	if result == "" {
		result = "untitled"
	}

	return result
}

// createJobTempDir creates a temporary directory for a job
func (s *Server) createJobTempDir(jobID string) string {
	tempDir := filepath.Join(os.TempDir(), "djset-server-jobs", jobID)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		slog.Error("Failed to create temp directory", "dir", tempDir, "error", err)
	}
	return tempDir
}

// addFileToZip adds a single file to the ZIP archive
func (s *Server) addFileToZip(zipWriter *zip.Writer, filePath string, trackNumber int, track *domain.Track) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Create ZIP entry with proper filename
	ext := s.getFileExtension(filePath)
	zipFileName := fmt.Sprintf("%02d-%s.%s", trackNumber, SanitizeFilename(track.Name), ext)

	zipEntry, err := zipWriter.Create(zipFileName)
	if err != nil {
		return fmt.Errorf("failed to create ZIP entry: %w", err)
	}

	// Copy file content to ZIP entry
	_, err = io.Copy(zipEntry, file)
	if err != nil {
		return fmt.Errorf("failed to write file to ZIP: %w", err)
	}

	return nil
}

// getFileExtension safely extracts the file extension without causing index out of bounds
func (s *Server) getFileExtension(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext == "" {
		return "mp3" // Default extension if none found
	}
	return strings.ToLower(ext[1:]) // Remove the dot and convert to lowercase
}

// scheduleJobCleanup schedules the cleanup of a job's temporary directory after a grace period
func (s *Server) scheduleJobCleanup(jobID string, tempDir string) {
	go func() {
		// Wait for the grace period to allow users to download files
		time.Sleep(JobCleanupGracePeriod)
		
		// Clean up the temporary directory
		if err := os.RemoveAll(tempDir); err != nil {
			slog.Error("Failed to clean up job temporary directory", 
				"jobID", jobID, "tempDir", tempDir, "error", err)
		} else {
			slog.Info("Successfully cleaned up job temporary directory", 
				"jobID", jobID, "tempDir", tempDir, "gracePeriod", JobCleanupGracePeriod)
		}
	}()
}

// cleanupJobTempDir immediately cleans up a job's temporary directory (used for failed jobs)
func (s *Server) cleanupJobTempDir(jobID string, tempDir string) {
	if err := os.RemoveAll(tempDir); err != nil {
		slog.Error("Failed to clean up failed job temporary directory", 
			"jobID", jobID, "tempDir", tempDir, "error", err)
	} else {
		slog.Debug("Cleaned up failed job temporary directory", 
			"jobID", jobID, "tempDir", tempDir)
	}
}
