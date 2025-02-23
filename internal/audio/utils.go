package audio

import (
	"fmt"
	"strconv"
	"strings"
)

// timeToSeconds converts a timestamp like "1:23:45" or "45:23" to seconds.
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
