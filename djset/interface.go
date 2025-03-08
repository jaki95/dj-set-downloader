package djset

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/storage"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

type Processor interface {
	ProcessTracks(opts *ProcessingOptions, progressCallback func(int, string)) ([]string, error)
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

	// Create the storage
	fileStorage, err := storage.NewStorage(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}

	return &processor{
		tracklistImporter: trackImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
		storage:           fileStorage,
	}, nil
}
