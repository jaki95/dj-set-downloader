package downloader

import (
	"fmt"
)

// GetDownloader returns the appropriate downloader for the given URL
func GetDownloader(url string) (Downloader, error) {
	// Check SoundCloud first
	soundcloudDownloader := NewSoundCloudDownloader()
	if soundcloudDownloader.SupportsURL(url) {
		return soundcloudDownloader, nil
	}

	// Fall back to HTTP downloader
	httpDownloader := NewHTTPDownloader()
	if httpDownloader.SupportsURL(url) {
		return httpDownloader, nil
	}

	return nil, fmt.Errorf("no downloader available for URL: %s", url)
}
