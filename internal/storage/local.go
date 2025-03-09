package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements the Storage interface for local filesystem
type LocalStorage struct {
	baseDir   string
	outputDir string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(baseDir, outputDir string) *LocalStorage {
	// Ensure baseDir is absolute or relative to current working directory
	if !filepath.IsAbs(baseDir) {
		// Get current working directory to use as base
		cwd, err := os.Getwd()
		if err == nil {
			baseDir = filepath.Join(cwd, baseDir)
		}
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		// Log error but continue - will try to create directories on demand
		fmt.Printf("Warning: Failed to create base directory %s: %v\n", baseDir, err)
	}

	// Ensure the output directory exists
	fullOutputDir := outputDir
	if !filepath.IsAbs(outputDir) {
		fullOutputDir = filepath.Join(baseDir, outputDir)
	}
	if err := os.MkdirAll(fullOutputDir, 0755); err != nil {
		fmt.Printf("Warning: Failed to create output directory %s: %v\n", fullOutputDir, err)
	}

	return &LocalStorage{
		baseDir:   baseDir,
		outputDir: outputDir,
	}
}

// CreateDir creates a directory and any necessary parents
func (s *LocalStorage) CreateDir(path string) error {
	// If the path is relative and doesn't start with the baseDir, join it with baseDir
	fullPath := path
	if !filepath.IsAbs(path) && !strings.HasPrefix(path, s.baseDir) {
		fullPath = filepath.Join(s.baseDir, path)
	}

	err := os.MkdirAll(fullPath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", fullPath, err)
	}
	return nil
}

// WriteFile writes data to a file, creating parent directories if needed
func (s *LocalStorage) WriteFile(path string, data []byte) error {
	// Create parent directories if they don't exist
	if err := s.CreateDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ReadFile reads the entire file into memory
func (s *LocalStorage) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// FileExists checks if a file exists
func (s *LocalStorage) FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err // Some other error
}

// DeleteFile removes a file
func (s *LocalStorage) DeleteFile(path string) error {
	return os.Remove(path)
}

// ListDir returns a list of files in a directory
func (s *LocalStorage) ListDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, filepath.Join(path, entry.Name()))
	}
	return result, nil
}

// CreateProcessDir creates a directory for a specific process
func (s *LocalStorage) CreateProcessDir(processID string) (string, error) {
	// Use the same base path structure as local downloads
	localBaseDir := filepath.Dir(s.GetLocalDownloadDir(processID)) // Gets /<baseDir>/local/<processID>
	processDir := filepath.Dir(localBaseDir)                       // Gets /<baseDir>/local
	processDir = filepath.Join(processDir, processID)              // Gets /<baseDir>/local/<processID>

	// Create the process directory
	if err := s.CreateDir(processDir); err != nil {
		return "", fmt.Errorf("failed to create process directory: %w", err)
	}

	// Create subdirectories
	if err := s.CreateDir(s.GetDownloadDir(processID)); err != nil {
		return "", fmt.Errorf("failed to create download directory: %w", err)
	}
	if err := s.CreateDir(s.GetTempDir(processID)); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	return processDir, nil
}

// GetDownloadDir returns the download directory for a process
func (s *LocalStorage) GetDownloadDir(processID string) string {
	// Maintain consistency with the local download directory structure
	localBaseDir := filepath.Join(s.baseDir, "local", processID)
	return filepath.Join(localBaseDir, "download")
}

// GetTempDir returns the temporary directory for a process
func (s *LocalStorage) GetTempDir(processID string) string {
	// Use the same base path structure as local downloads to avoid permission issues
	localDir := filepath.Dir(s.GetLocalDownloadDir(processID)) // This gets /<baseDir>/local/<processID>
	return filepath.Join(localDir, "temp")
}

// GetOutputDir returns the output directory for a specific set
func (s *LocalStorage) GetOutputDir(setName string) string {
	if !filepath.IsAbs(s.outputDir) {
		return filepath.Join(s.baseDir, s.outputDir, setName)
	}
	return filepath.Join(s.outputDir, setName)
}

// CleanupProcessDir removes a process directory after a delay
func (s *LocalStorage) CleanupProcessDir(processID string) error {
	// Sleep for a bit to ensure all files are written and closed
	time.Sleep(5 * time.Minute)

	// Clean up the process directory in the local structure
	processDir := filepath.Join(s.baseDir, "local", processID)
	return os.RemoveAll(processDir)
}

// GetLocalDownloadDir returns a local filesystem path for the download directory
// For LocalStorage, this is the same as GetDownloadDir
func (s *LocalStorage) GetLocalDownloadDir(processID string) string {
	return s.GetDownloadDir(processID)
}
