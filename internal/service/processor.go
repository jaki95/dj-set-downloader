package service

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/job"
	"github.com/jaki95/dj-set-downloader/pkg/audio"
	"github.com/jaki95/dj-set-downloader/pkg/downloader"
)

// Processor performs the downloading the DJ set and splitting it into tracks.
//
// Progress updates are pushed to the caller via the provided callback.
// For percentage-to-stage mapping, see the constants in the job package.
//
// Future improvements:
// 1. make the downloader and audio engine pluggable via interfaces.
// 2. make processor goroutine safe
type Processor struct {
	cfg *config.Config
}

// NewProcessor builds a Processor with the given configuration.
func NewProcessor(cfg *config.Config) *Processor {
	return &Processor{cfg: cfg}
}

// Process orchestrates the full download + split + metadata pipeline.
//
// * tracklist        – validated track metadata
// * downloadURL      – direct link (currently only SoundCloud supported)
// * fileExtension    – mp3, flac, …  (passed to ffmpeg)
// * maxConcurrentTasks – reserved for future parallelism, currently unused
// * progressCallback – receives percentage [0-100], human message, optional binary data
func (p *Processor) Process(
	ctx context.Context,
	tracklist *domain.Tracklist,
	downloadURL string,
	fileExtension string,
	maxConcurrentTasks int,
	progressCallback func(int, string, []byte),
) ([]string, error) {
	slog.Info("Starting process", "url", downloadURL, "extension", fileExtension, "trackCount", len(tracklist.Tracks))

	tempDir := filepath.Join(os.TempDir(), "djset-server", fmt.Sprintf("direct_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		slog.Error("Failed to create temp directory", "tempDir", tempDir, "error", err)
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	slog.Info("Created temp directory", "tempDir", tempDir)

	// Map download progress (0-100%) to overall progress range.
	downloadProgressCallback := func(percent int, message string, data []byte) {
		downloadRange := float64(job.ProgressDownloadEnd - job.ProgressDownloadStart)
		mappedPercent := job.ProgressDownloadStart + int(float64(percent)*downloadRange/100.0)
		progressCallback(mappedPercent, message, data)
	}

	slog.Info("Getting downloader for URL", "url", downloadURL)
	dl, err := downloader.GetDownloader(downloadURL)
	if err != nil {
		slog.Error("Failed to get downloader", "url", downloadURL, "error", err)
		return nil, fmt.Errorf("failed to get downloader: %w", err)
	}

	slog.Info("Got downloader, starting download", "url", downloadURL)
	downloadedFile, err := dl.Download(ctx, downloadURL, tempDir, downloadProgressCallback)
	if err != nil {
		slog.Error("Download failed", "url", downloadURL, "error", err)
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}
	slog.Info("Download completed", "file", downloadedFile)

	// Basic sanity check on artist/name to avoid directory traversal etc.
	if strings.Contains(tracklist.Artist, "/") || strings.Contains(tracklist.Artist, "\\") || strings.Contains(tracklist.Artist, "..") ||
		strings.Contains(tracklist.Name, "/") || strings.Contains(tracklist.Name, "\\") || strings.Contains(tracklist.Name, "..") {
		slog.Error("Unsafe characters in tracklist", "artist", tracklist.Artist, "name", tracklist.Name)
		return nil, fmt.Errorf("artist or name contains unsafe characters")
	}

	outputDir := filepath.Join(p.cfg.Storage.OutputDir, fmt.Sprintf("%s - %s", tracklist.Artist, tracklist.Name))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("Failed to create output directory", "outputDir", outputDir, "error", err)
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	slog.Info("Created output directory", "outputDir", outputDir)

	slog.Info("Starting track splitting", "inputFile", downloadedFile, "outputDir", outputDir)
	return p.splitTracks(ctx, downloadedFile, tracklist, outputDir, "", fileExtension, progressCallback)
}

// splitTracks performs the ffmpeg splitting and tagging for each track.
func (p *Processor) splitTracks(
	ctx context.Context,
	inputFilePath string,
	tracklist *domain.Tracklist,
	outputDir string,
	coverArtPath string,
	fileExtension string,
	progressCallback func(int, string, []byte),
) ([]string, error) {
	audioProcessor := audio.NewFFMPEGEngine()

	if coverArtPath == "" {
		coverArtPath = filepath.Join(filepath.Dir(inputFilePath), "cover.jpg")
	}

	if err := audioProcessor.ExtractCoverArt(ctx, inputFilePath, coverArtPath); err != nil {
		coverArtPath = "" // ignore errors – cover art is optional
	}

	var results []string
	totalTracks := len(tracklist.Tracks)

	progressCallback(job.ProgressProcessingStart, "Starting track processing...", nil)

	for i, track := range tracklist.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		trackNumber := i + 1
		trackName := fmt.Sprintf("%02d-%s", trackNumber, sanitizeFilename(track.Name))
		outputPath := filepath.Join(outputDir, trackName)

		splitParams := audio.SplitParams{
			InputPath:     inputFilePath,
			OutputPath:    outputPath,
			FileExtension: fileExtension,
			Track:         *track,
			TrackCount:    totalTracks,
			Artist:        tracklist.Artist,
			Name:          tracklist.Name,
			CoverArtPath:  coverArtPath,
		}

		if err := audioProcessor.Split(ctx, splitParams); err != nil {
			return nil, fmt.Errorf("failed to process track %d (%s): %w", trackNumber, track.Name, err)
		}

		progress := job.ProgressProcessingStart + int(float64(trackNumber)/float64(totalTracks)*float64(job.ProgressProcessingEnd-job.ProgressProcessingStart))
		progressCallback(progress, fmt.Sprintf("Processed track %d/%d: %s", trackNumber, totalTracks, track.Name), nil)

		finalPath := fmt.Sprintf("%s.%s", outputPath, fileExtension)
		results = append(results, finalPath)
	}

	progressCallback(job.ProgressComplete, "Processing completed", nil)
	return results, nil
}

// sanitizeFilename replaces characters that are problematic on common filesystems.
func sanitizeFilename(name string) string {
	unsafe := []string{"/", "\\", "..", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return strings.TrimSpace(result)
}
