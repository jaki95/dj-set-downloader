package downloader

import (
	"context"
	"fmt"
)

// Downloader represents a generic audio downloader interface
type Downloader interface {
	// Download downloads audio from the given URL to the output directory
	// Returns the path to the downloaded file
	Download(ctx context.Context, url, outputDir string) (string, error)

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
