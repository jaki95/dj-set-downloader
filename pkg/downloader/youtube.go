package downloader

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Error types specific to YouTube downloader
var (
	ErrYtDlpNotAvailable = fmt.Errorf("yt-dlp not available")
)

// YouTubeDownloader handles downloading from YouTube using yt-dlp
type YouTubeDownloader struct {
	timeout time.Duration
}

// NewYouTubeDownloader creates a new YouTube downloader
func NewYouTubeDownloader() *YouTubeDownloader {
	return &YouTubeDownloader{
		timeout: defaultDownloadTimeout,
	}
}

// SupportsURL checks if the URL is from YouTube
func (d *YouTubeDownloader) SupportsURL(url string) bool {
	return strings.Contains(url, "youtube.com") || strings.Contains(url, "youtu.be")
}

// Download downloads audio from YouTube using yt-dlp
func (d *YouTubeDownloader) Download(ctx context.Context, url, outputDir string, progressCallback ProgressCallback) (string, error) {
	slog.Info("Downloading from YouTube", "url", url, "outputDir", outputDir)

	if err := d.checkYtDlpAvailable(); err != nil {
		return "", fmt.Errorf("%w: %v", ErrYtDlpNotAvailable, err)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get video title for filename
	title, err := d.getVideoTitle(ctx, url)
	if err != nil {
		slog.Warn("Failed to get video title, using fallback", "error", err)
		title = "youtube_video"
	}

	// Clean title for filename
	cleanTitle := d.cleanFilename(title)
	outputTemplate := filepath.Join(outputDir, cleanTitle+".%(ext)s")

	args := []string{
		url,
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "0",
		"--output", outputTemplate,
		"--no-playlist",
		"--force-overwrites",
		"--no-warnings",
		"--max-downloads", "1",
		"--embed-thumbnail", // Embed thumbnail as cover art
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Dir = outputDir

	slog.Info("Executing yt-dlp command", "args", args)

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	slog.Info("yt-dlp process started, waiting for completion", "pid", cmd.Process.Pid)

	// Notify client that download has started
	if progressCallback != nil {
		progressCallback(10, "Download initiated...", nil)
	}

	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		slog.Info("yt-dlp process finished", "pid", cmd.Process.Pid, "error", err)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode := exitError.ExitCode()
				// Exit code 101 is used by yt-dlp when --max-downloads is reached (not an error)
				if exitCode == 101 {
					slog.Info("yt-dlp completed with max-downloads reached (normal)")
					goto downloadComplete
				}
			}

			slog.Error("yt-dlp command failed",
				"error", err,
				"stdout", stdoutBuf.String(),
				"stderr", stderrBuf.String(),
			)
			return "", fmt.Errorf("yt-dlp download failed: %w\nstdout: %s\nstderr: %s",
				err, stdoutBuf.String(), stderrBuf.String())
		}
		slog.Info("yt-dlp download completed successfully")
		if progressCallback != nil {
			progressCallback(100, "Download completed", nil)
		}
	case <-ctx.Done():
		slog.Warn("Context cancelled, killing yt-dlp process", "pid", cmd.Process.Pid)
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
	}

downloadComplete:
	downloadedFile, err := findDownloadedFile(outputDir)
	if err != nil {
		return "", fmt.Errorf("failed to find downloaded file: %w", err)
	}

	if err := validateAudioFile(downloadedFile); err != nil {
		return "", fmt.Errorf("downloaded file validation failed: %w", err)
	}

	slog.Info("Successfully downloaded from YouTube", "file", downloadedFile)
	return downloadedFile, nil
}

// checkYtDlpAvailable verifies that yt-dlp is installed and available
func (d *YouTubeDownloader) checkYtDlpAvailable() error {
	cmd := exec.Command("yt-dlp", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrYtDlpNotAvailable, err)
	}
	return nil
}

// getVideoTitle gets the video title for use as filename
func (d *YouTubeDownloader) getVideoTitle(ctx context.Context, url string) (string, error) {
	cmd := exec.CommandContext(ctx, "yt-dlp", "--get-title", url)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// cleanFilename removes invalid characters from filename
func (d *YouTubeDownloader) cleanFilename(filename string) string {
	// Remove or replace invalid filename characters
	invalidChars := regexp.MustCompile(`[<>:"/\\|?*]`)
	clean := invalidChars.ReplaceAllString(filename, "_")

	// Limit length
	if len(clean) > 100 {
		clean = clean[:100]
	}

	return strings.TrimSpace(clean)
}
