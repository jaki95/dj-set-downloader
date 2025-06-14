package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
)

// Processor handles track processing
type Processor struct {
	outputDir string
}

// New creates a new processor
func New(outputDir string) *Processor {
	return &Processor{
		outputDir: outputDir,
	}
}

// Process handles the processing of a tracklist
func (p *Processor) Process(ctx context.Context, tracklist *domain.Tracklist, downloadURL, fileExtension string, maxConcurrentTasks int, progressCallback func(int, string, []byte)) ([]string, error) {
	tempDir := filepath.Join(os.TempDir(), "djset-server", fmt.Sprintf("direct_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	progressCallback(10, "Downloading audio file...", nil)
	dl, err := downloader.GetDownloader(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get downloader: %w", err)
	}
	downloadedFile, err := dl.Download(ctx, downloadURL, tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download audio: %w", err)
	}

	if strings.Contains(tracklist.Artist, "/") || strings.Contains(tracklist.Artist, "\\") || strings.Contains(tracklist.Artist, "..") ||
		strings.Contains(tracklist.Name, "/") || strings.Contains(tracklist.Name, "\\") || strings.Contains(tracklist.Name, "..") {
		return nil, fmt.Errorf("invalid tracklist fields: artist or name contains unsafe characters")
	}

	outputDir := filepath.Join(p.outputDir, fmt.Sprintf("%s - %s", tracklist.Artist, tracklist.Name))
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return p.splitTracks(ctx, downloadedFile, tracklist, outputDir, "", fileExtension, progressCallback)
}

// splitTracks handles the track splitting
func (p *Processor) splitTracks(ctx context.Context, inputFilePath string, tracklist *domain.Tracklist, outputDir, coverArtPath, fileExtension string, progressCallback func(int, string, []byte)) ([]string, error) {
	audioProcessor := audio.NewFFMPEGEngine()

	if coverArtPath == "" {
		coverArtPath = filepath.Join(filepath.Dir(inputFilePath), "cover.jpg")
	}

	if err := audioProcessor.ExtractCoverArt(ctx, inputFilePath, coverArtPath); err != nil {
		coverArtPath = ""
	}

	var results []string
	totalTracks := len(tracklist.Tracks)

	progressCallback(20, "Starting track processing...", nil)

	for i, track := range tracklist.Tracks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		trackNumber := i + 1
		safeTitle := sanitizeTitle(track.Name)
		trackName := fmt.Sprintf("%02d-%s", trackNumber, safeTitle)
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

		progress := 20 + int(float64(trackNumber)/float64(totalTracks)*70) // 20-90%
		progressCallback(progress, fmt.Sprintf("Processed track %d/%d: %s", trackNumber, totalTracks, track.Name), nil)

		finalPath := fmt.Sprintf("%s.%s", outputPath, fileExtension)
		results = append(results, finalPath)
	}

	progressCallback(100, "Processing completed", nil)
	return results, nil
}

// sanitizeTitle removes unsafe characters from a title
func sanitizeTitle(title string) string {
	// Replace unsafe characters with underscores
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := title
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}
	return result
}
