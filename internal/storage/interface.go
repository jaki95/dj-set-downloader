package storage

import (
	"io"
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
