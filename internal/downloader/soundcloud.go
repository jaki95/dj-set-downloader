package downloader

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// SoundCloudDownloader handles downloading from SoundCloud using scdl
type SoundCloudDownloader struct {
	clientID string
}

// NewSoundCloudDownloader creates a new SoundCloud downloader
func NewSoundCloudDownloader() *SoundCloudDownloader {
	return &SoundCloudDownloader{
		clientID: os.Getenv("SOUNDCLOUD_CLIENT_ID"),
	}
}

// SupportsURL checks if the URL is from SoundCloud
func (d *SoundCloudDownloader) SupportsURL(url string) bool {
	return strings.Contains(url, "soundcloud.com")
}

// Download downloads audio from SoundCloud using scdl
func (d *SoundCloudDownloader) Download(ctx context.Context, url, outputDir string) (string, error) {
	slog.Info("Downloading from SoundCloud", "url", url, "outputDir", outputDir)

	// Check if scdl is available
	if err := d.checkScdlAvailable(); err != nil {
		return "", fmt.Errorf("scdl not available: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build scdl command with supported options
	args := []string{
		"-l", url,
		"--path", outputDir,
		"--onlymp3",
		"--addtofile",
		"--no-playlist-folder",
		"--flac",          // Prefer FLAC if available
		"--original-art",  // Download original artwork
		"--original-name", // Use original filename
		"--overwrite",     // Overwrite existing files
	}

	// Add client ID if available
	if d.clientID != "" {
		args = append(args, "--client-id", d.clientID)
	}

	// Execute scdl command with timeout
	cmd := exec.CommandContext(ctx, "scdl", args...)
	cmd.Dir = outputDir

	slog.Debug("Executing scdl command", "args", args)

	// Create buffers to capture output
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start scdl: %w", err)
	}

	// Wait for completion with timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			// Log the full output for debugging
			slog.Error("scdl command failed",
				"error", err,
				"stdout", stdoutBuf.String(),
				"stderr", stderrBuf.String(),
			)
			return "", fmt.Errorf("scdl download failed: %w\nstdout: %s\nstderr: %s",
				err, stdoutBuf.String(), stderrBuf.String())
		}
	case <-ctx.Done():
		if err := cmd.Process.Kill(); err != nil {
			slog.Error("Failed to kill process after context cancellation", "error", err)
		}
		return "", ctx.Err()
	case <-time.After(30 * time.Minute): // 30-minute timeout
		if err := cmd.Process.Kill(); err != nil {
			slog.Error("Failed to kill process after timeout", "error", err)
		}
		return "", fmt.Errorf("download timed out after 30 minutes")
	}

	// Find and validate the downloaded file
	downloadedFile, err := d.findDownloadedFile(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to find downloaded file: %w", err)
	}

	// Validate the downloaded file
	if err := d.validateAudioFile(downloadedFile); err != nil {
		return "", fmt.Errorf("downloaded file validation failed: %w", err)
	}

	slog.Info("Successfully downloaded from SoundCloud", "file", downloadedFile)
	return downloadedFile, nil
}

// checkScdlAvailable verifies that scdl is installed and available
func (d *SoundCloudDownloader) checkScdlAvailable() error {
	cmd := exec.Command("scdl", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scdl command not found. Please install scdl: pip install scdl")
	}
	return nil
}

// findDownloadedFile finds the most recently downloaded audio file in the directory
func (d *SoundCloudDownloader) findDownloadedFile(outputDir string) (string, error) {
	// Look for common audio file extensions
	audioExtensions := []string{".mp3", ".m4a", ".wav", ".flac"}

	var mostRecentFile string
	var mostRecentTime time.Time

	err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		for _, audioExt := range audioExtensions {
			if ext == audioExt {
				if info.ModTime().After(mostRecentTime) {
					mostRecentTime = info.ModTime()
					mostRecentFile = path
				}
				break
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error scanning output directory: %w", err)
	}

	if mostRecentFile == "" {
		return "", fmt.Errorf("no audio files found in output directory")
	}

	return mostRecentFile, nil
}

// validateAudioFile checks if the downloaded file is a valid audio file
func (d *SoundCloudDownloader) validateAudioFile(filepath string) error {
	// Check if file exists and has size
	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("downloaded file is empty")
	}

	// Check if file is at least 1MB (to avoid partial downloads)
	if info.Size() < 1024*1024 {
		return fmt.Errorf("downloaded file is too small (less than 1MB)")
	}

	// TODO: Add more validation like checking file headers, etc.
	return nil
}
