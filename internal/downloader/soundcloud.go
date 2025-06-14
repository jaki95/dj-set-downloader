// Package downloader provides functionality for downloading audio files from various sources.
// It includes implementations for different platforms like SoundCloud.
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

// Constants for SoundCloud downloader
const (
	// Default timeout for downloads
	defaultDownloadTimeout = 30 * time.Minute

	// Minimum file size to consider a download valid (1MB)
	minValidFileSize = 1024 * 1024

	// Supported audio file extensions
	supportedAudioExtensions = ".mp3,.m4a,.wav,.flac"
)

// Error types for better error handling
var (
	ErrScdlNotAvailable = fmt.Errorf("scdl not available")
	ErrNoAudioFiles     = fmt.Errorf("no audio files found")
	ErrFileTooSmall     = fmt.Errorf("file too small")
	ErrDownloadTimeout  = fmt.Errorf("download timeout")
)

// SoundCloudDownloader handles downloading from SoundCloud using scdl
type SoundCloudDownloader struct {
	clientID string
	timeout  time.Duration
}

// NewSoundCloudDownloader creates a new SoundCloud downloader
func NewSoundCloudDownloader() *SoundCloudDownloader {
	return &SoundCloudDownloader{
		clientID: os.Getenv("SOUNDCLOUD_CLIENT_ID"),
		timeout:  defaultDownloadTimeout,
	}
}

// SupportsURL checks if the URL is from SoundCloud
func (d *SoundCloudDownloader) SupportsURL(url string) bool {
	return strings.Contains(url, "soundcloud.com")
}

// Download downloads audio from SoundCloud using scdl
func (d *SoundCloudDownloader) Download(ctx context.Context, url, outputDir string) (string, error) {
	slog.Info("Downloading from SoundCloud", "url", url, "outputDir", outputDir)

	if err := d.checkScdlAvailable(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrScdlNotAvailable, err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

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

	// Detect URL type and add appropriate download type
	urlType := d.detectURLType(url)
	slog.Info("Detected SoundCloud URL type", "url", url, "type", urlType)

	switch urlType {
	case "user_profile":
		return "", fmt.Errorf("user profiles are not supported")
	case "playlist", "set":
		return "", fmt.Errorf("playlists are not supported")
	case "track":
		fallthrough
	default:
	}

	cmd := exec.CommandContext(ctx, "scdl", args...)
	cmd.Dir = outputDir

	slog.Info("Executing scdl command", "args", args)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start scdl: %w", err)
	}

	slog.Info("scdl process started, waiting for completion", "pid", cmd.Process.Pid)

	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		slog.Info("scdl process finished", "pid", cmd.Process.Pid, "error", err)
		done <- err
	}()

	// Add periodic progress logging with output capture
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			if err != nil {
				slog.Error("scdl command failed",
					"error", err,
					"stdout", stdoutBuf.String(),
					"stderr", stderrBuf.String(),
				)
				return "", fmt.Errorf("scdl download failed: %w\nstdout: %s\nstderr: %s",
					err, stdoutBuf.String(), stderrBuf.String())
			}
			slog.Info("scdl download completed successfully")
			goto downloadComplete
		case <-ctx.Done():
			slog.Warn("Context cancelled, killing scdl process", "pid", cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				slog.Error("Failed to kill process after context cancellation", "error", err)
			}
			return "", ctx.Err()
		case <-time.After(d.timeout):
			slog.Error("Download timeout reached", "timeout", d.timeout, "pid", cmd.Process.Pid)
			if err := cmd.Process.Kill(); err != nil {
				slog.Error("Failed to kill process after timeout", "error", err)
			}
			return "", fmt.Errorf("%w: %v", ErrDownloadTimeout, d.timeout)
		case <-ticker.C:
			// Show current output buffers to see what scdl is doing
			stdout := stdoutBuf.String()
			stderr := stderrBuf.String()
			slog.Info("scdl download still in progress...",
				"pid", cmd.Process.Pid,
				"stdout_length", len(stdout),
				"stderr_length", len(stderr),
				"latest_stdout", func() string {
					if len(stdout) > 200 {
						return "..." + stdout[len(stdout)-200:]
					}
					return stdout
				}(),
				"latest_stderr", func() string {
					if len(stderr) > 200 {
						return "..." + stderr[len(stderr)-200:]
					}
					return stderr
				}())
		}
	}

downloadComplete:
	downloadedFile, err := d.findDownloadedFile(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to find downloaded file: %w", err)
	}

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
		return fmt.Errorf("%w: %v", ErrScdlNotAvailable, err)
	}
	return nil
}

// findDownloadedFile finds the most recently downloaded audio file in the directory
func (d *SoundCloudDownloader) findDownloadedFile(outputDir string) (string, error) {
	audioExtensions := strings.Split(supportedAudioExtensions, ",")
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
		return "", fmt.Errorf("%w: in directory %s", ErrNoAudioFiles, outputDir)
	}

	return mostRecentFile, nil
}

// validateAudioFile checks if the downloaded file is a valid audio file
func (d *SoundCloudDownloader) validateAudioFile(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("%w: file is empty", ErrFileTooSmall)
	}

	if info.Size() < minValidFileSize {
		return fmt.Errorf("%w: file size %d bytes is less than minimum %d bytes",
			ErrFileTooSmall, info.Size(), minValidFileSize)
	}

	// TODO: Add more validation like checking file headers, etc.
	return nil
}

// detectURLType determines the type of SoundCloud URL
func (d *SoundCloudDownloader) detectURLType(url string) string {
	// Remove trailing slashes and normalize
	url = strings.TrimSuffix(url, "/")

	// Split URL into parts
	parts := strings.Split(url, "/")
	if len(parts) < 4 {
		return "unknown"
	}

	// Check for specific patterns
	if strings.Contains(url, "/sets/") {
		return "playlist"
	}

	if strings.Contains(url, "/tracks") {
		return "user_tracks"
	}

	if strings.Contains(url, "/likes") {
		return "user_likes"
	}

	if strings.Contains(url, "/reposts") {
		return "user_reposts"
	}

	// If URL has exactly 4 parts (protocol, empty, domain, username), it's a user profile
	if len(parts) == 4 && strings.Contains(parts[2], "soundcloud.com") {
		return "user_profile"
	}

	// If URL has 5 parts (protocol, empty, domain, username, track), it's likely a track
	if len(parts) == 5 {
		return "track"
	}

	return "unknown"
}
