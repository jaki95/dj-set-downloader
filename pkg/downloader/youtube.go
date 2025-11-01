package downloader

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
		"--progress",
		"--newline",
		"--max-downloads", "1",
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	cmd.Dir = outputDir

	slog.Info("Executing yt-dlp command", "args", args)

	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf

	// Create a pipe for stderr to read progress in real-time
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	slog.Info("yt-dlp process started, waiting for completion", "pid", cmd.Process.Pid)

	// Channel for progress updates from stderr reader
	progressChan := make(chan int, 1)
	var stderrBuf bytes.Buffer

	// Read stderr line-by-line in a goroutine to capture progress in real-time
	go func() {
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.WriteString(line + "\n")

			// Parse progress from the line
			if progress := d.parseProgressFromLine(line); progress > 0 {
				select {
				case progressChan <- progress:
				default:
					// Channel full, skip this update
				}
			}
		}
		// Read any remaining data
		if remaining, err := io.ReadAll(stderrPipe); err == nil {
			stderrBuf.Write(remaining)
		}
	}()

	done := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		slog.Info("yt-dlp process finished", "pid", cmd.Process.Pid, "error", err)
		done <- err
	}()

	progressPercent := 10
	if progressCallback != nil {
		progressCallback(progressPercent, "Download initiated...", nil)
	}

	for {
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
			goto downloadComplete
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
		case newProgress := <-progressChan:
			if newProgress > progressPercent {
				progressPercent = newProgress
				if progressCallback != nil {
					progressCallback(progressPercent, fmt.Sprintf("Downloading... %d%%", progressPercent), nil)
				}
			}
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

	// Download and embed thumbnail as cover art
	if err := d.downloadAndEmbedThumbnail(ctx, url, downloadedFile, outputDir, cleanTitle); err != nil {
		slog.Warn("Failed to download or embed thumbnail, continuing without cover art", "error", err)
		// Continue without cover art - this is optional
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

// parseProgressFromLine extracts progress percentage from a single yt-dlp output line
// yt-dlp progress format: [download] 45.2% of 123.45MiB at 1.23MiB/s ETA 00:01:23
func (d *YouTubeDownloader) parseProgressFromLine(line string) int {
	// Look for the download progress pattern: [download] XX.X%
	progressRegex := regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)
	matches := progressRegex.FindStringSubmatch(line)

	if len(matches) > 1 {
		if percent, err := strconv.ParseFloat(matches[1], 64); err == nil {
			if percent >= 0 && percent <= 100 {
				return int(percent)
			}
		}
	}

	return 0
}

// findDownloadedFile finds the most recently downloaded audio file in the directory
func (d *YouTubeDownloader) findDownloadedFile(outputDir string) (string, error) {
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
func (d *YouTubeDownloader) validateAudioFile(filepath string) error {
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

	return nil
}

// downloadAndEmbedThumbnail downloads the video thumbnail and embeds it as cover art
func (d *YouTubeDownloader) downloadAndEmbedThumbnail(ctx context.Context, url, audioFile, outputDir, baseName string) error {
	// Download thumbnail
	thumbnailTemplate := filepath.Join(outputDir, baseName+".%(ext)s")
	thumbnailArgs := []string{
		url,
		"--write-thumbnail",
		"--skip-download",
		"--output", thumbnailTemplate,
		"--no-playlist",
		"--force-overwrites",
		"--no-warnings",
	}

	cmd := exec.CommandContext(ctx, "yt-dlp", thumbnailArgs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download thumbnail: %w", err)
	}

	// Find the downloaded thumbnail file
	thumbnailFile, err := d.findThumbnailFile(outputDir, baseName)
	if err != nil {
		return fmt.Errorf("failed to find thumbnail file: %w", err)
	}
	defer os.Remove(thumbnailFile) // Clean up thumbnail file after embedding

	// Embed thumbnail into audio file using ffmpeg
	return d.embedThumbnailWithFFmpeg(ctx, audioFile, thumbnailFile)
}

// findThumbnailFile finds the downloaded thumbnail file
func (d *YouTubeDownloader) findThumbnailFile(outputDir, baseName string) (string, error) {
	// yt-dlp can download thumbnails in various formats (jpg, png, webp, etc.)
	thumbnailExtensions := []string{".jpg", ".jpeg", ".png", ".webp"}

	for _, ext := range thumbnailExtensions {
		thumbnailPath := filepath.Join(outputDir, baseName+ext)
		if _, err := os.Stat(thumbnailPath); err == nil {
			return thumbnailPath, nil
		}
	}

	// Fallback: find any recently created image file in the directory
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
		for _, thumbnailExt := range thumbnailExtensions {
			if ext == thumbnailExt {
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
		return "", fmt.Errorf("error scanning for thumbnail: %w", err)
	}

	if mostRecentFile == "" {
		return "", fmt.Errorf("thumbnail file not found")
	}

	return mostRecentFile, nil
}

// embedThumbnailWithFFmpeg embeds the thumbnail into the audio file as cover art
func (d *YouTubeDownloader) embedThumbnailWithFFmpeg(ctx context.Context, audioFile, thumbnailFile string) error {
	ext := strings.ToLower(filepath.Ext(audioFile))
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}

	// Determine format
	var format string
	switch ext {
	case "mp3":
		format = "mp3"
	case "m4a":
		format = "mp4"
	case "wav":
		format = "wav"
	case "flac":
		format = "flac"
	default:
		return fmt.Errorf("unsupported audio format: %s", ext)
	}

	// Create temporary output file
	tempOutput := audioFile + ".tmp"
	defer os.Remove(tempOutput)

	args := []string{
		"-y",
		"-i", audioFile,
		"-i", thumbnailFile,
		"-map", "0:a",
		"-map", "1:v",
		"-c:a", "copy",
		"-c:v", "mjpeg",
		"-disposition:v:0", "attached_pic",
		"-f", format,
		"-movflags", "+faststart",
		"-id3v2_version", "3",
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
		tempOutput,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed to embed thumbnail: %w\noutput: %s", err, string(output))
	}

	// Replace original file with the one containing cover art
	if err := os.Rename(tempOutput, audioFile); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	slog.Info("Successfully embedded thumbnail as cover art", "file", audioFile)
	return nil
}
