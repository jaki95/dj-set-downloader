package downloader

import (
	"context"
	"testing"
)

// MockDownloader for testing
type MockDownloader struct {
	supportedURLs []string
	name          string
}

func (m *MockDownloader) Download(ctx context.Context, url, outputDir string, progressCallback ProgressCallback) (string, error) {
	return "mock_file.mp3", nil
}

func (m *MockDownloader) SupportsURL(url string) bool {
	for _, supportedURL := range m.supportedURLs {
		if url == supportedURL {
			return true
		}
	}
	return false
}

func TestDownloaderRegistry(t *testing.T) {
	registry := NewDownloaderRegistry()

	// Test with YouTube URL
	youtubeURL := "https://www.youtube.com/watch?v=test123"
	downloader, err := registry.GetDownloader(youtubeURL)
	if err != nil {
		t.Fatalf("Expected no error for YouTube URL, got: %v", err)
	}
	if downloader == nil {
		t.Fatal("Expected downloader, got nil")
	}

	// Test with SoundCloud URL
	soundcloudURL := "https://soundcloud.com/artist/track"
	downloader, err = registry.GetDownloader(soundcloudURL)
	if err != nil {
		t.Fatalf("Expected no error for SoundCloud URL, got: %v", err)
	}
	if downloader == nil {
		t.Fatal("Expected downloader, got nil")
	}

	// Test with unsupported URL
	unsupportedURL := "https://example.com/audio"
	_, err = registry.GetDownloader(unsupportedURL)
	if err == nil {
		t.Fatal("Expected error for unsupported URL, got nil")
	}
}

func TestDownloaderRegistry_Register(t *testing.T) {
	registry := NewDownloaderRegistry()

	// Create a mock downloader
	mockDownloader := &MockDownloader{
		supportedURLs: []string{"https://example.com/audio"},
		name:          "MockDownloader",
	}

	// Register the mock downloader
	registry.Register(mockDownloader)

	// Test that it can handle the URL
	url := "https://example.com/audio"
	downloader, err := registry.GetDownloader(url)
	if err != nil {
		t.Fatalf("Expected no error for registered URL, got: %v", err)
	}
	if downloader == nil {
		t.Fatal("Expected downloader, got nil")
	}
}

func TestGetDownloader(t *testing.T) {
	// Test the global function
	youtubeURL := "https://www.youtube.com/watch?v=test123"
	downloader, err := GetDownloader(youtubeURL)
	if err != nil {
		t.Fatalf("Expected no error for YouTube URL, got: %v", err)
	}
	if downloader == nil {
		t.Fatal("Expected downloader, got nil")
	}
}
