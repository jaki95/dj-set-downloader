package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/djset"
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

	if *tracklistURL == "" {
		slog.Error("Missing required flag: -tracklist-url")
		return
	}

	cfg, err := config.Load("./config/config.yaml")
	if err != nil {
		slog.Error(err.Error())
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(cfg.LogLevel)}))
	slog.SetDefault(logger)

	processor, err := djset.New(cfg)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	opts := &djset.ProcessingOptions{
		TracklistPath:      *tracklistURL,
		DJSetURL:           *soundcloudURL,
		CoverArtPath:       "./data/cover_temp.jpg",
		MaxConcurrentTasks: *maxWorkers,
	}

	if err := processor.ProcessTracks(opts); err != nil {
		slog.Error(err.Error())
		return
	}
}
