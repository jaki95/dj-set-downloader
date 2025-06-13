package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/server"
)

func main() {
	port := flag.String("port", "8080", "Server port")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load("./config/config.yaml")
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Setup logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(cfg.LogLevel)}))
	slog.SetDefault(logger)

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	slog.Info("Starting DJ Set Processor API server", "port", *port)
	if err := srv.Start(*port); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
