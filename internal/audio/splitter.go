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

	// First pass: Extract audio segment (without cover art)
	pass1Args := []string{
		"-y",
		"-i", opts.InputPath,
		"-ss", fmt.Sprintf("%.3f", startSeconds),
	}
	if duration > 0 {
		pass1Args = append(pass1Args, "-t", fmt.Sprintf("%.3f", duration))
	}
	pass1Args = append(pass1Args, []string{
		"-map", "0:a",
		"-c:a", "aac",
		"-b:a", "128k",
		"-af", "aresample=async=1",
		"-movflags", "+faststart",
		"-id3v2_version", "3",
		tempAudio,
	}...)

	cmd := exec.Command("ffmpeg", pass1Args...)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("first pass (audio extraction) failed: %w", err)
	}

	// Second pass: Reattach cover art to the extracted audio segment
	pass2Args := []string{
		"-y",
		"-i", tempAudio,
		"-i", opts.CoverArtPath,
		"-map", "0:a", // Map audio from temp file
		"-map", "1:v", // Map cover art from opts.CoverArtPath
		"-c:a", "copy", // Copy audio stream from the temp file
		"-c:v", "mjpeg", // Encode cover art as MJPEG
		"-disposition:v:0", "attached_pic", // Mark cover art as attached picture
		"-movflags", "+faststart",
		"-id3v2_version", "3",
		"-metadata", fmt.Sprintf("album_artist=%s", opts.Artist),
		"-metadata", fmt.Sprintf("artist=%s", opts.Track.Artist),
		"-metadata", fmt.Sprintf("title=%s", opts.Track.Title),
		"-metadata", fmt.Sprintf("track=%d/%d", opts.Track.TrackNumber, opts.TrackCount),
		"-metadata", fmt.Sprintf("album=%s", opts.Name),
		"-metadata:s:v", "title=Album cover",
		"-metadata:s:v", "comment=Cover (front)",
		"-metadata", "compilation=1",
		fmt.Sprintf("%s.m4a", opts.OutputPath),
	}

	cmd = exec.Command("ffmpeg", pass2Args...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("second pass (cover art attachment) failed: %w", err)
	}

	// Remove the temporary file
	os.Remove(tempAudio)

	return nil
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
