// Package downloader provides functionality for downloading audio files from various sources.
package downloader

import (
	"context"
	"fmt"
)

// ProgressCallback is a function type for progress updates during download
// Parameters: progressPercent (0-100), message, optional data
type ProgressCallback func(int, string, []byte)

// Downloader represents a generic audio downloader interface
type Downloader interface {
	// Download downloads audio from the given URL to the output directory
	// Returns the path to the downloaded file
	// progressCallback can be nil if progress updates are not needed
	Download(ctx context.Context, url, outputDir string, progressCallback ProgressCallback) (string, error)

	// SupportsURL checks if this downloader can handle the given URL
	SupportsURL(url string) bool
}

// DownloaderRegistry manages available downloaders
type DownloaderRegistry struct {
	downloaders []Downloader
}

// NewDownloaderRegistry creates a new registry with default downloaders
func NewDownloaderRegistry() *DownloaderRegistry {
	return &DownloaderRegistry{
		downloaders: []Downloader{
			NewYouTubeDownloader(),
			NewSoundCloudDownloader(),
		},
	}
}

// Register adds a new downloader to the registry
func (r *DownloaderRegistry) Register(downloader Downloader) {
	r.downloaders = append(r.downloaders, downloader)
}

// GetDownloader returns the appropriate downloader for the given URL
func (r *DownloaderRegistry) GetDownloader(url string) (Downloader, error) {
	for _, downloader := range r.downloaders {
		if downloader.SupportsURL(url) {
			return downloader, nil
		}
	}
	return nil, fmt.Errorf("no downloader available for URL: %s", url)
}

// Global registry instance
var defaultRegistry = NewDownloaderRegistry()

// GetDownloader returns the appropriate downloader for the given URL using the default registry
func GetDownloader(url string) (Downloader, error) {
	return defaultRegistry.GetDownloader(url)
}
