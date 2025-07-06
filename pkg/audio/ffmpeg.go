// Package audio provides functionality for processing audio files using FFmpeg.
// It includes features for extracting cover art, splitting audio files into tracks,
// and adding metadata to audio files.
package audio

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Supported audio file extensions and their corresponding FFmpeg codecs and formats
var (
	supportedExtensions = map[string]struct {
		codec  string
		format string
	}{
		"mp3":  {"libmp3lame", "mp3"},
		"m4a":  {"aac", "mp4"},
		"wav":  {"pcm_s16le", "wav"},
		"flac": {"flac", "flac"},
	}

	// Default audio settings
	defaultAudioBitrate = "128k"
	defaultID3Version   = "3"
)

var (
	ErrFileNotFound     = fmt.Errorf("file not found")
	ErrFileEmpty        = fmt.Errorf("file is empty")
	ErrInvalidPath      = fmt.Errorf("invalid path")
	ErrInvalidExtension = fmt.Errorf("invalid file extension")
	ErrInvalidPrefix    = fmt.Errorf("invalid file prefix")
)

// ffmpegError wraps FFmpeg command errors with additional context
type ffmpegError struct {
	cmd     string
	output  string
	wrapped error
}

func (e *ffmpegError) Error() string {
	return fmt.Sprintf("ffmpeg error: %s\nCommand: %s\nOutput: %s", e.wrapped, e.cmd, e.output)
}

func (e *ffmpegError) Unwrap() error {
	return e.wrapped
}

// newFFmpegError creates a new ffmpegError with truncated command output
func newFFmpegError(cmd *exec.Cmd, output []byte, err error) error {
	cmdStr := cmd.String()
	if len(cmdStr) > 200 {
		cmdStr = cmdStr[:200] + "..."
	}
	return &ffmpegError{
		cmd:     cmdStr,
		output:  string(output),
		wrapped: err,
	}
}

type ffmpeg struct{}

func NewFFMPEGEngine() *ffmpeg {
	return &ffmpeg{}
}

func (f *ffmpeg) validateFile(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("unable to access file: %s: %w", path, err)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%w: %s is a directory", ErrInvalidPath, path)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%w: %s", ErrFileEmpty, path)
	}

	return nil
}

// ExtractCoverArt extracts the cover art from the input file and saves it to the coverPath
func (f *ffmpeg) ExtractCoverArt(ctx context.Context, inputPath, coverPath string) error {
	slog.Debug("Extracting cover art", "input", inputPath, "output", coverPath)

	if err := f.validateFile(inputPath); err != nil {
		return fmt.Errorf("cover art extraction failed: %w", err)
	}

	outputDir := filepath.Dir(coverPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", inputPath,
		"-map", "0:v:0",
		"-c:v", "mjpeg",
		"-vframes", "1",
		coverPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return newFFmpegError(cmd, output, err)
	}

	return nil
}

// sanitizePath ensures the path is safe and returns an absolute path
func (f *ffmpeg) sanitizePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Allow temporary files (system temp directory)
	tempDir := os.TempDir()
	if tempDir != "" {
		absTempDir, err := filepath.Abs(tempDir)
		if err == nil && strings.HasPrefix(absPath, absTempDir) {
			return absPath, nil
		}
	}

	// Allow paths within the working directory
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	if strings.HasPrefix(absPath, baseDir) {
		return absPath, nil
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("%w: path contains '..' which is not allowed", ErrInvalidPath)
	}

	// For output directories, allow if they're absolute paths without traversal
	if filepath.IsAbs(path) && !strings.Contains(path, "..") {
		return absPath, nil
	}

	return "", fmt.Errorf("%w: path must be within the working directory or a safe absolute path", ErrInvalidPath)
}

// createTempFile creates a temporary file in the system's temp directory
func (f *ffmpeg) createTempFile(extension string) (string, error) {
	const prefix = "audio_segment"

	tempFile, err := os.CreateTemp("", prefix+"_*."+extension)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()
	return tempPath, nil
}

func (f *ffmpeg) Split(ctx context.Context, opts SplitParams) error {
	if err := f.validateFile(opts.InputPath); err != nil {
		return fmt.Errorf("track splitting failed: %w", err)
	}

	if _, ok := supportedExtensions[opts.FileExtension]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidExtension, opts.FileExtension)
	}

	if opts.CoverArtPath != "" {
		if err := f.validateFile(opts.CoverArtPath); err != nil {
			return fmt.Errorf("cover art validation failed: %w", err)
		}
	}

	sanitizedOutputPath, err := f.sanitizePath(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	startSeconds, err := timeToSeconds(opts.Track.StartTime)
	if err != nil {
		return fmt.Errorf("error parsing start time for track %d: %w", opts.Track.TrackNumber, err)
	}

	var duration float64
	if opts.Track.EndTime != "" {
		endSeconds, err := timeToSeconds(opts.Track.EndTime)
		if err != nil {
			return fmt.Errorf("error parsing end time for track %d: %w", opts.Track.TrackNumber, err)
		}
		duration = endSeconds - startSeconds
	}

	tempAudio, err := f.createTempFile(opts.FileExtension)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempAudio)

	if err := f.extractAudio(ctx, opts.InputPath, startSeconds, duration, tempAudio); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	finalPath := fmt.Sprintf("%s.%s", sanitizedOutputPath, opts.FileExtension)
	return f.addMetadataAndCover(ctx, tempAudio, finalPath, opts)
}

func (f *ffmpeg) extractAudio(ctx context.Context, inputPath string, startSeconds, duration float64, outputPath string) error {
	slog.Debug("Extracting audio segment",
		"input", inputPath,
		"output", outputPath,
		"start", fmt.Sprintf("%.3f", startSeconds),
		"duration", fmt.Sprintf("%.3f", duration),
	)

	sanitizedOutputPath, err := f.sanitizePath(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}

	outputDir := filepath.Dir(sanitizedOutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	ext := filepath.Ext(sanitizedOutputPath)
	if ext != "" {
		ext = ext[1:] // Remove the leading dot
	}

	codecInfo, ok := supportedExtensions[strings.ToLower(ext)]
	if !ok {
		return fmt.Errorf("%w: %s", ErrInvalidExtension, ext)
	}

	args := []string{
		"-y",
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.3f", startSeconds),
	}

	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", duration))
	}

	args = append(args,
		"-map", "0:a",
		"-c:a", codecInfo.codec,
		"-f", codecInfo.format,
		"-b:a", defaultAudioBitrate,
		"-af", "aresample=async=1",
		"-movflags", "+faststart",
		"-id3v2_version", defaultID3Version,
		sanitizedOutputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return newFFmpegError(cmd, output, err)
	}

	return nil
}

func (f *ffmpeg) addMetadataAndCover(ctx context.Context, inputPath, outputPath string, opts SplitParams) error {
	slog.Debug("Adding metadata and cover art",
		"input", inputPath,
		"output", outputPath,
		"track", opts.Track.Name,
	)

	ext := filepath.Ext(outputPath)
	if ext != "" {
		ext = ext[1:] // Remove the leading dot
	}

	codecInfo, ok := supportedExtensions[strings.ToLower(ext)]
	if !ok {
		return fmt.Errorf("%w: %s", ErrInvalidExtension, ext)
	}

	args := []string{
		"-y",
		"-i", inputPath,
		"-i", opts.CoverArtPath,
		"-map", "0:a",
		"-map", "1:v",
		"-c:a", "copy",
		"-c:v", "mjpeg",
		"-f", codecInfo.format,
		"-disposition:v:0", "attached_pic",
		"-movflags", "+faststart",
		"-id3v2_version", defaultID3Version,
	}

	// Add standard metadata
	metadata := map[string]string{
		"album_artist": opts.Artist,
		"artist":       opts.Track.Artist,
		"title":        opts.Track.Name,
		"track":        fmt.Sprintf("%d/%d", opts.Track.TrackNumber, opts.TrackCount),
		"album":        opts.Name,
		"compilation":  "1",
	}
	for k, v := range metadata {
		args = append(args, "-metadata", fmt.Sprintf("%s=%s", k, v))
	}

	// Add video stream metadata
	videoMetadata := map[string]string{
		"title":   "Album cover",
		"comment": "Cover (front)",
	}
	for k, v := range videoMetadata {
		args = append(args, "-metadata:s:v", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, outputPath)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return newFFmpegError(cmd, output, err)
	}

	return nil
}
