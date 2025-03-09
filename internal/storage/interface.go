package storage

import (
	"context"
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Basic operations
	CreateDir(path string) error
	WriteFile(path string, data []byte) error
	ReadFile(path string) ([]byte, error)
	FileExists(path string) (bool, error)
	DeleteFile(path string) error

	// Directory operations
	ListDir(path string) ([]string, error)

	// Process operations - for handling files during processing
	CreateProcessDir(processID string) (string, error)
	GetDownloadDir(processID string) string
	GetTempDir(processID string) string
	GetOutputDir(setName string) string

	// External tool operations - for getting local paths to use with external tools
	GetLocalDownloadDir(processID string) string // Returns a local filesystem path for download operations

	// Cleanup
	CleanupProcessDir(processID string) error
}

// Factory function to create a storage implementation based on config
func NewStorage(ctx context.Context, cfg *config.Config) (Storage, error) {
	switch cfg.Storage.Type {
	case "local", "":
		dataDir := cfg.Storage.DataDir
		if dataDir == "" {
			dataDir = "storage"
		}

		outputDir := cfg.Storage.OutputDir
		if outputDir == "" {
			outputDir = "output"
		}

		return NewLocalStorage(dataDir, outputDir), nil

	case "gcs":
		// Check required GCS configuration
		if cfg.Storage.BucketName == "" {
			return nil, fmt.Errorf("GCS storage requires bucket_name configuration")
		}

		return NewGCSStorage(
			ctx,
			cfg.Storage.BucketName,
			cfg.Storage.ObjectPrefix,
			cfg.Storage.CredentialsFile,
		)

	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
