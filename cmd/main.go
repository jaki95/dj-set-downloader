package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

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

	processor, err := djset.NewProcessor(cfg)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh
		slog.Info("Received interrupt signal, cancelling operations...")
		cancel()
	}()

	opts := &djset.ProcessingOptions{
		TracklistPath:      *tracklistURL,
		DJSetURL:           *soundcloudURL,
		FileExtension:      cfg.FileExtension,
		MaxConcurrentTasks: *maxWorkers,
	}

	if _, err := processor.ProcessTracks(ctx, opts, func(i int, s string, data []byte) {}); err != nil {
		if err == context.Canceled {
			slog.Info("Processing was cancelled")
		} else {
			slog.Error(err.Error())
		}
		return
	}
}
