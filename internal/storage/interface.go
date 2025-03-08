package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/jaki95/dj-set-downloader/config"
)

// Storage defines the interface for handling file storage operations
// related to downloaded sets and processed tracks.
type Storage interface {
	// SaveDownloadedSet returns the path for storing a downloaded set
	SaveDownloadedSet(setName string, originalExt string) (string, error)

	// SaveTrack returns the path for storing a track
	SaveTrack(setName, trackName string, ext string) (string, error)

	// GetSetPath returns the path to a downloaded set
	GetSetPath(setName string, ext string) string

	// GetCoverArtPath returns the path for temporary cover art storage
	GetCoverArtPath() string

	// CreateSetOutputDir creates the output directory for a set's tracks
	CreateSetOutputDir(setName string) error

	// Cleanup removes temporary files
	Cleanup() error

	// GetReader returns a reader for a file
	GetReader(path string) (io.ReadCloser, error)

	// GetWriter returns a writer for a file
	GetWriter(path string) (io.WriteCloser, error)

	// FileExists checks if a file exists
	FileExists(path string) bool

	// ListFiles lists files in a directory matching a pattern
	ListFiles(dir string, pattern string) ([]string, error)
}

func NewStorage(cfg *config.Config) (Storage, error) {
	// Set defaults if not provided
	dataDir := cfg.Storage.DataDir
	if dataDir == "" {
		dataDir = "data"
	}

	outputDir := cfg.Storage.OutputDir
	if outputDir == "" {
		outputDir = "output"
	}

	tempDir := cfg.Storage.TempDir
	if tempDir == "" {
		tempDir = "temp"
	}

	switch cfg.Storage.Type {
	case "local", "":
		return NewLocalFileStorage(dataDir, outputDir, tempDir)
	case "gcs":
		// Check required GCS configuration
		if cfg.Storage.BucketName == "" {
			return nil, fmt.Errorf("GCS storage requires bucket_name configuration")
		}

		return NewGCSStorage(
			context.Background(),
			cfg.Storage.BucketName,
			cfg.Storage.ObjectPrefix,
			tempDir,
			cfg.Storage.CredentialsFile,
		)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
