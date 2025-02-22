package main

import (
	"flag"
	"fmt"
	"os"

	"log/slog"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/djset"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/jaki95/dj-set-downloader/pkg"
)

func main() {
	tracklistURL := flag.String("tracklist-url", "", "URL to the tracklist source (required)")
	soundcloudURL := flag.String("soundcloud-url", "", "URL of the SoundCloud DJ set (optional)")
	maxWorkers := flag.Int("workers", 4, "Maximum concurrent processing tasks")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Validate required flags with explicit checks
	if *tracklistURL == "" {
		slog.Error("Missing required flag: -tracklist-url")
		return
	}

	cfg, err := pkg.LoadConfig("./config/config.yaml")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(cfg.LogLevel)}))
	slog.SetDefault(logger)

	tracklistImporter, err := tracklist.NewImporter(cfg)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	audioProcessor := audio.NewFFMPEGEngine()
	setProcessor := djset.NewProcessor(tracklistImporter, audioProcessor)

	processingOptions := &djset.ProcessingOptions{
		TracklistPath:      *tracklistURL,
		DJSetURL:           *soundcloudURL,
		CoverArtPath:       "./data/cover_temp.jpg",
		MaxConcurrentTasks: *maxWorkers,
	}

	if err := setProcessor.ProcessTracks(processingOptions); err != nil {
		slog.Error(err.Error())
		return
	}
}
