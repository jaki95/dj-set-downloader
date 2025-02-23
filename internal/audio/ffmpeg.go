package audio

import (
	"fmt"
	"os"
	"os/exec"
)

type ffmpeg struct{}

func NewFFMPEGEngine() *ffmpeg {
	return &ffmpeg{}
}

func (f *ffmpeg) ExtractCoverArt(inputPath, coverPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inputPath, "-map", "0:v:0", "-c:v", "mjpeg", "-vframes", "1", coverPath)
	return cmd.Run()
}

func (f *ffmpeg) Split(opts SplitParams) error {
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
	tempAudio := fmt.Sprintf("%s_temp.m4a", opts.OutputPath)

	defer os.Remove(tempAudio)

	// First pass: Extract audio segment
	if err := f.extractAudio(opts.InputPath, startSeconds, duration, tempAudio); err != nil {
		return err
	}

	// Second pass: Attach metadata and cover art
	finalPath := fmt.Sprintf("%s.m4a", opts.OutputPath)

	return f.addMetadataAndCover(tempAudio, finalPath, opts)
}

func (f *ffmpeg) extractAudio(inputPath string, startSeconds, duration float64, outputPath string) error {
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
		return fmt.Errorf("ffmpeg error: %s: %w", string(output), err)
	}

	return nil
}

func (f *ffmpeg) addMetadataAndCover(inputPath, outputPath string, opts SplitParams) error {
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
		return fmt.Errorf("ffmpeg error: %s: %w", string(output), err)
	}

	return nil
}
