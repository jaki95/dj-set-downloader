package audio

import (
	"fmt"
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
		"-i", opts.InputPath,
		"-ss", fmt.Sprintf("%.3f", startSeconds),
		"-t", fmt.Sprintf("%.3f", duration),
		"-c", "copy",
		"-metadata", fmt.Sprintf("album_artist=%s", opts.Artist),
		"-metadata", fmt.Sprintf("artist=%s", opts.Track.Artist),
		"-metadata", fmt.Sprintf("title=%s", opts.Track.Title),
		"-metadata", fmt.Sprintf("track=%d/%d", opts.Track.TrackNumber, opts.TrackCount),
		"-metadata", fmt.Sprintf("album=%s", opts.Name),
		opts.OutputPath,
	}
	cmd := exec.Command("ffmpeg", args...)

	return cmd.Run()
}

// TimeToSeconds converts a timestamp like "1:23:45" or "45:23" to seconds
func timeToSeconds(timestamp string) (float64, error) {
	parts := strings.Split(timestamp, ":")
	var hours, minutes, seconds float64

	switch len(parts) {
	case 3: // H:MM:SS
		h, _ := strconv.ParseFloat(parts[0], 64)
		m, _ := strconv.ParseFloat(parts[1], 64)
		s, _ := strconv.ParseFloat(parts[2], 64)
		hours = h
		minutes = m
		seconds = s
	case 2: // MM:SS
		m, _ := strconv.ParseFloat(parts[0], 64)
		s, _ := strconv.ParseFloat(parts[1], 64)
		minutes = m
		seconds = s
	default:
		return 0, fmt.Errorf("invalid timestamp format: %s", timestamp)
	}

	return hours*3600 + minutes*60 + seconds, nil
}
