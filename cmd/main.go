package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/djset"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/jaki95/dj-set-downloader/pkg"
)

func main() {
	tracklistURL := flag.String("tracklist-url", "", "URL to the tracklist source (required)")
	soundcloudURL := flag.String("soundcloud-url", "", "URL of the SoundCloud DJ set (required)")
	setName := flag.String("name", "", "Name of the DJ set (required)")
	setArtist := flag.String("artist", "", "Artist name for the DJ set (required)")
	maxWorkers := flag.Int("workers", 4, "Maximum concurrent processing tasks")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	// Validate required flags with explicit checks
	if *tracklistURL == "" {
		log.Fatal("Missing required flag: -tracklist-url")
	}
	if *soundcloudURL == "" {
		log.Fatal("Missing required flag: -soundcloud-url")
	}
	if *setName == "" {
		log.Fatal("Missing required flag: -set-name")
	}
	if *setArtist == "" {
		log.Fatal("Missing required flag: -set-artist")
	}

	cfg, err := pkg.LoadConfig("./config/config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	tracklistImporter, err := tracklist.NewImporter(cfg)
	if err != nil {
		log.Fatal(err)
	}

	audioProcessor := audio.NewFFMPEGEngine()
	setProcessor := djset.NewProcessor(tracklistImporter, audioProcessor)

	processingOptions := &djset.ProcessingOptions{
		TracklistPath:      *tracklistURL,
		DJSetURL:           *soundcloudURL,
		InputFile:          fmt.Sprintf("%s.mp3", fmt.Sprintf("data/%s", *setName)),
		SetArtist:          *setArtist,
		SetName:            *setName,
		CoverArtPath:       "./data/cover_temp.jpg",
		MaxConcurrentTasks: *maxWorkers,
	}

	if err := setProcessor.ProcessTracks(processingOptions); err != nil {
		log.Fatal(err)
	}
}
