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
	// Default TTL for processed files
	DefaultFileTTL = 24 * time.Hour

	// Cleanup interval for old files
	CleanupInterval = 2 * time.Hour
)

// StartCleanupWorker starts a background worker to clean up old files
func (s *Server) StartCleanupWorker() {
	ticker := time.NewTicker(CleanupInterval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			s.cleanupOldFiles()
		}
	}()
	slog.Info("File cleanup worker started", "interval", CleanupInterval)
}

// cleanupOldFiles removes files older than TTL
func (s *Server) cleanupOldFiles() {
	tempJobsDir := filepath.Join(os.TempDir(), "djset-server-jobs")
	if _, err := os.Stat(tempJobsDir); os.IsNotExist(err) {
		return
	}

	cutoffTime := time.Now().Add(-DefaultFileTTL)
	slog.Debug("Starting cleanup of old files", "cutoff", cutoffTime)

	entries, err := os.ReadDir(tempJobsDir)
	if err != nil {
		slog.Error("Failed to read temp jobs directory", "error", err)
		return
	}

	cleaned := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		jobDir := filepath.Join(tempJobsDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.RemoveAll(jobDir); err != nil {
				slog.Error("Failed to remove old job directory", "dir", jobDir, "error", err)
			} else {
				slog.Debug("Cleaned up old job directory", "dir", jobDir, "age", time.Since(info.ModTime()))
				cleaned++
			}
		}
	}

	if cleaned > 0 {
		slog.Info("Cleanup completed", "directories_cleaned", cleaned)
	}
}

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
	ext := strings.ToLower(filepath.Ext(filePath)[1:])
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
