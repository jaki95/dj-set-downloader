package audio

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ffmpeg struct{}

func NewFFMPEGProcessor() *ffmpeg {
	return &ffmpeg{}
}

func (f *ffmpeg) Split(opts SplitParams) error {
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

	args := []string{
		"-y",
		// Step 1: Extract cover art from the original input
		"-i", opts.InputPath,
		"-map", "0:a", // Map audio
		"-map", "0:v:0", // Map cover art
		"-c:a", "aac", // Set audio codec explicitly
		"-b:a", "128k", // Audio bitrate
		"-c:v", "mjpeg", // Set video codec for cover art
		"-disposition:v:0", "attached_pic", // Mark it as album artwork
		"-ss", fmt.Sprintf("%.3f", startSeconds),
	}
	if duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", duration))
	}
	args = append(args, []string{
		"-af", "aresample=async=1",
		"-movflags", "+faststart",
		"-f", "mp4",
		"-metadata", fmt.Sprintf("album_artist=%s", opts.Artist),
		"-metadata", fmt.Sprintf("artist=%s", opts.Track.Artist),
		"-metadata", fmt.Sprintf("title=%s", opts.Track.Title),
		"-metadata", fmt.Sprintf("track=%d/%d", opts.Track.TrackNumber, opts.TrackCount),
		"-metadata", fmt.Sprintf("album=%s", opts.Name),
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
		"-metadata", "compilation=1",
		fmt.Sprintf("%s.m4a", opts.OutputPath),
	}...)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

// TimeToSeconds converts a timestamp like "1:23:45" or "45:23" to seconds
func timeToSeconds(timestamp string) (float64, error) {
	parts := strings.Split(timestamp, ":")
	var hours, minutes, seconds float64
	var err error

	switch len(parts) {
	case 3: // H:MM:SS
		if hours, err = strconv.ParseFloat(parts[0], 64); err != nil {
			return 0, fmt.Errorf("invalid hours: %w", err)
		}
		if minutes, err = strconv.ParseFloat(parts[1], 64); err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		if seconds, err = strconv.ParseFloat(parts[2], 64); err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	case 2: // MM:SS
		if minutes, err = strconv.ParseFloat(parts[0], 64); err != nil {
			return 0, fmt.Errorf("invalid minutes: %w", err)
		}
		if seconds, err = strconv.ParseFloat(parts[1], 64); err != nil {
			return 0, fmt.Errorf("invalid seconds: %w", err)
		}
	default:
		return 0, fmt.Errorf("invalid timestamp format: %s", timestamp)
	}

	return hours*3600 + minutes*60 + seconds, nil
}
