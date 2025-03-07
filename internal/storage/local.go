package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalFileStorage implements the Storage interface for local filesystem
type LocalFileStorage struct {
	dataDir   string
	outputDir string
	tempDir   string
}

// NewLocalFileStorage creates a new local file storage instance
func NewLocalFileStorage(dataDir, outputDir, tempDir string) (*LocalFileStorage, error) {
	// Ensure directories exist
	for _, dir := range []string{dataDir, outputDir, tempDir} {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return &LocalFileStorage{
		dataDir:   dataDir,
		outputDir: outputDir,
		tempDir:   tempDir,
	}, nil
}

// SaveDownloadedSet returns the path where the set should be stored
// Note: The actual saving is done by the downloader tool
func (s *LocalFileStorage) SaveDownloadedSet(setName string, ext string) (string, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(s.dataDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return s.GetSetPath(setName, ext), nil
}

// SaveTrack returns the path where the track should be stored
// Note: The actual saving is done by the audio processor
func (s *LocalFileStorage) SaveTrack(setName, trackName string, ext string) (string, error) {
	// Ensure the output directory exists
	outputDir := filepath.Join(s.outputDir, setName)
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Return the full path
	return filepath.Join(outputDir, fmt.Sprintf("%s.%s", trackName, ext)), nil
}

// GetSetPath returns the path to a downloaded set
func (s *LocalFileStorage) GetSetPath(setName string, ext string) string {
	return filepath.Join(s.dataDir, fmt.Sprintf("%s.%s", setName, ext))
}

// GetCoverArtPath returns the path for temporary cover art storage
func (s *LocalFileStorage) GetCoverArtPath() string {
	return filepath.Join(s.tempDir, "cover_temp.jpg")
}

// CreateSetOutputDir creates the output directory for a set's tracks
func (s *LocalFileStorage) CreateSetOutputDir(setName string) error {
	outputDir := filepath.Join(s.outputDir, setName)
	return os.MkdirAll(outputDir, os.ModePerm)
}

// Cleanup removes temporary files
func (s *LocalFileStorage) Cleanup() error {
	// Currently just removes the cover art
	if err := os.Remove(s.GetCoverArtPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cover art: %w", err)
	}
	return nil
}

// GetReader returns a reader for the specified file
func (s *LocalFileStorage) GetReader(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

// GetWriter returns a writer for the specified file
func (s *LocalFileStorage) GetWriter(path string) (io.WriteCloser, error) {
	return os.Create(path)
}

// FileExists checks if a file exists
func (s *LocalFileStorage) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ListFiles lists files in a directory matching a pattern
func (s *LocalFileStorage) ListFiles(dir string, pattern string) ([]string, error) {
	// If dir is empty, use the data directory
	if dir == "" {
		dir = s.dataDir
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var results []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Match pattern (simple prefix for now)
		if pattern != "" && !strings.HasPrefix(file.Name(), pattern) {
			continue
		}

		results = append(results, filepath.Join(dir, file.Name()))
	}

	return results, nil
}
