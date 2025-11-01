package downloader

import (
	"fmt"
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
