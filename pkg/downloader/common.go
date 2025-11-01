package downloader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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
	ErrNoAudioFiles    = fmt.Errorf("no audio files found")
	ErrFileTooSmall    = fmt.Errorf("file too small")
	ErrDownloadTimeout = fmt.Errorf("download timeout")
)

// findDownloadedFile finds the most recently downloaded audio file in the directory
func findDownloadedFile(outputDir string) (string, error) {
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
func validateAudioFile(filepath string) error {
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
