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

// GetDownloader returns the appropriate downloader for the given URL
func GetDownloader(url string) (Downloader, error) {
	// Check SoundCloud first
	soundcloudDownloader := NewSoundCloudDownloader()
	if soundcloudDownloader.SupportsURL(url) {
		return soundcloudDownloader, nil
	}

	return nil, fmt.Errorf("no downloader available for URL: %s", url)
}
