package main

import (
	"flag"
	"log"
	"os"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/scraper"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

func main() {
	urlFlag := flag.String("url", "", "Tracklist URL")
	setNameFlag := flag.String("name", "", "Set name")
	setArtistFlag := flag.String("artist", "", "Set artist name")
	inputFlag := flag.String("input", "", "Path to input audio file")
	flag.Parse()

	// Validate required flags
	if *urlFlag == "" || *setNameFlag == "" || *setArtistFlag == "" || *inputFlag == "" {
		flag.Usage()
		log.Fatal("url, name, artist and input flags are required")
	}

	url := *urlFlag
	setName := *setNameFlag
	setArtist := *setArtistFlag
	inputFile := *inputFlag

	tl, err := scraper.GetTracklist(url)
	if err != nil {
		log.Fatal(err)
	}

	cover := "/Users/jaki/Projects/dj-set-downloader/cover_temp.jpg"

	audioProcessor := audio.NewFFMPEGProcessor()
	if err := audioProcessor.ExtractCoverArt(inputFile, cover); err != nil {
		log.Fatal(err)
	}

	tracklistProcessor := tracklist.NewProcessor(audioProcessor)

	if err := tracklistProcessor.ProcessTracks(
		tl.Tracks,
		&tracklist.ProcessOptions{
			InputFile:          inputFile,
			SetArtist:          setArtist,
			SetName:            setName,
			CoverArtPath:       cover,
			MaxConcurrentTasks: 4,
		},
	); err != nil {
		log.Fatal(err)
	}

	os.Remove(cover)
}
