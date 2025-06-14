package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/server"
)

func main() {
	port := flag.String("port", "8000", "Port to run the HTTP server on")
	flag.Parse()

	cfg, err := config.Load("./config/config.yaml")
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	// Override port from command line if provided
	portSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portSet = true
		}
	})
	if portSet {
		cfg.Server.Port = *port
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.Level(cfg.LogLevel)}))
	slog.SetDefault(logger)

	server := server.New(cfg)

	slog.Info("Starting DJ Set Processor HTTP server", "port", cfg.Server.Port)
	if err := server.Start(); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
