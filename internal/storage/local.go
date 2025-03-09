package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LocalStorage implements the Storage interface for local filesystem
type LocalStorage struct {
	baseDir   string
	outputDir string
}

// NewLocalStorage creates a new LocalStorage instance
func NewLocalStorage(baseDir, outputDir string) *LocalStorage {
	return &LocalStorage{
		baseDir:   baseDir,
		outputDir: outputDir,
	}
}

// CreateDir creates a directory and any necessary parents
func (s *LocalStorage) CreateDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
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
	processesDir := filepath.Join(s.baseDir, "processes")
	if err := s.CreateDir(processesDir); err != nil {
		return "", err
	}

	processDir := filepath.Join(processesDir, processID)
	if err := s.CreateDir(processDir); err != nil {
		return "", err
	}

	// Create subdirectories
	if err := s.CreateDir(s.GetDownloadDir(processID)); err != nil {
		return "", err
	}
	if err := s.CreateDir(s.GetTempDir(processID)); err != nil {
		return "", err
	}

	return processDir, nil
}

// GetDownloadDir returns the download directory for a process
func (s *LocalStorage) GetDownloadDir(processID string) string {
	return filepath.Join(s.baseDir, "processes", processID, "download")
}

// GetTempDir returns the temporary directory for a process
func (s *LocalStorage) GetTempDir(processID string) string {
	return filepath.Join(s.baseDir, "processes", processID, "temp")
}

// GetOutputDir returns the output directory for a specific set
func (s *LocalStorage) GetOutputDir(setName string) string {
	return filepath.Join(s.outputDir, setName)
}

// CleanupProcessDir removes a process directory after a delay
func (s *LocalStorage) CleanupProcessDir(processID string) error {
	// Sleep for a bit to ensure all files are written and closed
	time.Sleep(5 * time.Minute)

	processDir := filepath.Join(s.baseDir, "processes", processID)
	return os.RemoveAll(processDir)
}

// GetLocalDownloadDir returns a local filesystem path for the download directory
// For LocalStorage, this is the same as GetDownloadDir
func (s *LocalStorage) GetLocalDownloadDir(processID string) string {
	return s.GetDownloadDir(processID)
}
