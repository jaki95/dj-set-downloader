package downloader

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HTTPDownloader handles downloading from generic HTTP URLs
type HTTPDownloader struct{}

// NewHTTPDownloader creates a new HTTP downloader
func NewHTTPDownloader() *HTTPDownloader {
	return &HTTPDownloader{}
}

// SupportsURL checks if the URL is a generic HTTP/HTTPS URL (not SoundCloud)
func (d *HTTPDownloader) SupportsURL(url string) bool {
	return (strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) &&
		!strings.Contains(url, "soundcloud.com")
}

// Download downloads audio from a generic HTTP URL
func (d *HTTPDownloader) Download(ctx context.Context, downloadUrl, outputDir string) (string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Minute, // Long timeout for large audio files
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", downloadUrl, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	// Make request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Determine filename from URL or Content-Disposition
	filename := "audio_file"
	if contentDisp := resp.Header.Get("Content-Disposition"); contentDisp != "" {
		// Try to extract filename from Content-Disposition header
		if idx := strings.Index(contentDisp, "filename="); idx != -1 {
			filename = strings.Trim(contentDisp[idx+9:], "\"")
		}
	} else {
		// Extract from URL
		if u, err := url.Parse(downloadUrl); err == nil && u.Path != "" {
			if name := filepath.Base(u.Path); name != "" && name != "." {
				filename = name
			}
		}
	}

	// Ensure we have an extension
	if !strings.Contains(filename, ".") {
		filename += ".mp3" // Default extension
	}

	// Create output file
	outputPath := filepath.Join(outputDir, filename)
	outFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Copy data
	bytesWritten, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	// Basic validation - check if we actually downloaded something
	if bytesWritten == 0 {
		return "", fmt.Errorf("downloaded file is empty")
	}

	// Log download info
	slog.Info("Downloaded audio file", "path", outputPath, "size", bytesWritten, "filename", filename)

	// Basic file type validation - check the first few bytes
	if err := d.validateAudioFile(outputPath); err != nil {
		return "", fmt.Errorf("downloaded file validation failed: %w", err)
	}

	return outputPath, nil
}

// validateAudioFile performs basic validation to ensure the downloaded file is likely an audio file
func (d *HTTPDownloader) validateAudioFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for validation: %w", err)
	}
	defer file.Close()

	// Read first 512 bytes to check file signature
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read file header: %w", err)
	}

	if n < 4 {
		return fmt.Errorf("file too small to be a valid audio file")
	}

	// Check for common audio file signatures
	header := buffer[:n]

	// MP3 signatures
	if len(header) >= 3 && header[0] == 0xFF && (header[1]&0xE0) == 0xE0 {
		return nil // MP3 frame header
	}
	if len(header) >= 3 && string(header[:3]) == "ID3" {
		return nil // MP3 with ID3 tag
	}

	// Other audio formats
	if len(header) >= 4 && string(header[:4]) == "RIFF" {
		return nil // WAV
	}
	if len(header) >= 4 && string(header[:4]) == "fLaC" {
		return nil // FLAC
	}
	if len(header) >= 4 && string(header[:4]) == "OggS" {
		return nil // OGG
	}
	if len(header) >= 8 && string(header[4:8]) == "ftyp" {
		return nil // M4A/MP4
	}

	// Check if it looks like HTML/text (common when download fails)
	checkLen := len(header)
	if checkLen > 100 {
		checkLen = 100
	}
	headerStr := strings.ToLower(string(header[:checkLen]))
	if strings.Contains(headerStr, "<html") || strings.Contains(headerStr, "<!doctype") {
		return fmt.Errorf("downloaded file appears to be HTML, not an audio file - check the download URL")
	}
	if strings.Contains(headerStr, "error") || strings.Contains(headerStr, "not found") {
		return fmt.Errorf("downloaded file appears to contain an error message - check the download URL")
	}

	// Log warning but don't fail - let ffmpeg try to handle it
	headerLen := len(header)
	if headerLen > 16 {
		headerLen = 16
	}
	slog.Warn("Could not verify audio file format, proceeding anyway", "path", filePath, "header", fmt.Sprintf("%x", header[:headerLen]))
	return nil
}
