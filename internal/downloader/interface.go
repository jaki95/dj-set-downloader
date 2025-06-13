package downloader

import "context"

// Downloader represents a generic audio downloader interface
type Downloader interface {
	// Download downloads audio from the given URL to the output directory
	// Returns the path to the downloaded file
	Download(ctx context.Context, url, outputDir string) (string, error)

	// SupportsURL checks if this downloader can handle the given URL
	SupportsURL(url string) bool
}
