package audio

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

type ffmpeg struct{}

func NewFFMPEGEngine() *ffmpeg {
	return &ffmpeg{}
}

func (f *ffmpeg) validateFile(path string) error {
	// Check if file exists and is readable
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("unable to access file: %s: %w", path, err)
	}

	// Check if it's a file (not directory)
	if fileInfo.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	// Check if file has content
	if fileInfo.Size() == 0 {
		return fmt.Errorf("file is empty (0 bytes): %s", path)
	}

	return nil
}

func (f *ffmpeg) ExtractCoverArt(inputPath, coverPath string) error {
	slog.Debug("Extracting cover art", "input", inputPath, "output", coverPath)

	// Validate input file before processing
	if err := f.validateFile(inputPath); err != nil {
		return fmt.Errorf("cover art extraction failed on input validation: %w", err)
	}

	// Make sure the output directory exists
	outputDir := filepath.Dir(coverPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory for cover art: %w", err)
	}

	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-map", "0:v:0", "-c:v", "mjpeg", "-vframes", "1", coverPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg cover extraction error: %s: %w", string(output), err)
	}

	return nil
}

func (f *ffmpeg) Split(opts SplitParams) error {
	// Validate input file before processing
	if err := f.validateFile(opts.InputPath); err != nil {
		return fmt.Errorf("track splitting failed on input validation: %w", err)
	}

	// Validate cover art if provided
	if opts.CoverArtPath != "" {
		if err := f.validateFile(opts.CoverArtPath); err != nil {
			return fmt.Errorf("track splitting failed on cover art validation: %w", err)
		}
	}

	// Calculate start time and duration
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

	// Define temporary file path for the audio segment
	tempAudio := fmt.Sprintf("%s_temp.%s", opts.OutputPath, opts.FileExtension)

	defer os.Remove(tempAudio)

	// First pass: Extract audio segment
	if err := f.extractAudio(opts.InputPath, startSeconds, duration, tempAudio); err != nil {
		return err
	}

	// Second pass: Attach metadata and cover art
	finalPath := fmt.Sprintf("%s.%s", opts.OutputPath, opts.FileExtension)

	return f.addMetadataAndCover(tempAudio, finalPath, opts)
}

func (f *ffmpeg) extractAudio(inputPath string, startSeconds, duration float64, outputPath string) error {
	slog.Debug("Extracting audio segment",
		"input", inputPath,
		"output", outputPath,
		"start", fmt.Sprintf("%.3f", startSeconds),
		"duration", fmt.Sprintf("%.3f", duration),
	)

	args := []string{
		"-y", "-i", inputPath,
		"-ss", fmt.Sprintf("%.3f", startSeconds),
	}

	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", duration))
	}

	args = append(args,
		"-map", "0:a",
		"-c:a", "aac",
		"-b:a", "128k",
		"-af", "aresample=async=1",
		"-movflags", "+faststart",
		"-id3v2_version", "3",
		outputPath,
	)

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmdStr := cmd.String()
		if len(cmdStr) > 200 {
			cmdStr = cmdStr[:200] + "..." // Truncate very long commands
		}
		return fmt.Errorf("ffmpeg extraction error: %s\nCommand: %s\nError: %w", string(output), cmdStr, err)
	}

	return nil
}

func (f *ffmpeg) addMetadataAndCover(inputPath, outputPath string, opts SplitParams) error {
	slog.Debug("Adding metadata and cover art",
		"input", inputPath,
		"output", outputPath,
		"track", opts.Track.Title,
	)

	args := []string{
		"-y",
		"-i", inputPath,
		"-i", opts.CoverArtPath,
		"-map", "0:a",
		"-map", "1:v",
		"-c:a", "copy",
		"-c:v", "mjpeg",
		"-disposition:v:0", "attached_pic",
		"-movflags", "+faststart",
		"-id3v2_version", "3",
	}

	// Add standard metadata
	metadata := map[string]string{
		"album_artist": opts.Artist,
		"artist":       opts.Track.Artist,
		"title":        opts.Track.Title,
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

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cmdStr := cmd.String()
		if len(cmdStr) > 200 {
			cmdStr = cmdStr[:200] + "..." // Truncate very long commands
		}
		return fmt.Errorf("ffmpeg metadata error: %s\nCommand: %s\nTrack: %s\nError: %w",
			string(output), cmdStr, opts.Track.Title, err)
	}

	return nil
}
