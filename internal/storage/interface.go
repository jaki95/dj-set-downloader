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
	SaveDownloadedSet(setName string, originalExt string) (string, error)

	SaveTrack(setName, trackName string, ext string) (string, error)

	GetSetPath(setName string, ext string) string

	GetCoverArtPath() string

	CreateSetOutputDir(setName string) error

	Cleanup() error

	GetReader(path string) (io.ReadCloser, error)

	GetWriter(path string) (io.WriteCloser, error)

	FileExists(path string) bool

	ListFiles(dir string, pattern string) ([]string, error)
}

func NewStorage(cfg *config.Config) (Storage, error) {
	switch cfg.Storage.Type {
	case "local", "":
		return NewLocalFileStorage(
			cfg.Storage.DataDir,
			cfg.Storage.OutputDir,
			cfg.Storage.TempDir,
		)
	case "gcs":
		// Check required GCS configuration
		if cfg.Storage.BucketName == "" {
			return nil, fmt.Errorf("GCS storage requires bucket_name configuration")
		}

		return NewGCSStorage(
			context.Background(),
			cfg.Storage.BucketName,
			cfg.Storage.ObjectPrefix,
			cfg.Storage.TempDir,
			cfg.Storage.CredentialsFile,
		)
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", cfg.Storage.Type)
	}
}
