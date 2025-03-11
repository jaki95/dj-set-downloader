package djset

import (
	"context"
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

type Processor interface {
	ProcessTracks(ctx context.Context, opts *ProcessingOptions, progressCallback func(int, string, []byte)) ([]string, error)
}

func NewProcessor(cfg *config.Config) (Processor, error) {
	// Create the track importer
	trackImporter, err := tracklist.NewImporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracklist importer: %w", err)
	}

	// Create the downloader
	setDownloader, err := downloader.NewDownloader(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create downloader: %w", err)
	}

	// Create the audio processor
	var audioProcessor audio.Processor
	processorType := cfg.AudioProcessor
	if processorType == "" {
		processorType = "ffmpeg" // Default to ffmpeg if not specified
	}

	switch processorType {
	case "ffmpeg":
		audioProcessor = audio.NewFFMPEGEngine()
	default:
		return nil, fmt.Errorf("unsupported audio processor: %s", processorType)
	}

	return &processor{
		tracklistImporter: trackImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
	}, nil
}
