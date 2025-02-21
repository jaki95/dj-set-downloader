package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/scraper"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

func main() {
	tracklistURLFlag := flag.String("url", "", "Tracklist URL")
	soundcloudURLFlag := flag.String("input", "", "Soundcloud URL")
	setNameFlag := flag.String("name", "", "Set name")
	setArtistFlag := flag.String("artist", "", "Set artist name")
	flag.Parse()

	// Validate required flags
	if *tracklistURLFlag == "" || *soundcloudURLFlag == "" || *setNameFlag == "" || *setArtistFlag == "" {
		flag.Usage()
		log.Fatal("url, name, artist and input flags are required")
	}

	tracklistURL := *tracklistURLFlag
	soundcloudURL := *soundcloudURLFlag
	setName := *setNameFlag
	setArtist := *setArtistFlag

	tl, err := scraper.GetTracklist(tracklistURL)
	if err != nil {
		log.Fatal(err)
	}

	cover := "/Users/jaki/Projects/dj-set-downloader/data/cover_temp.jpg"

	err = downloader.Download(setName, soundcloudURL)
	if err != nil {
		log.Fatal(err)
	}

	setPath := fmt.Sprintf("data/%s", setName)

	audioProcessor := audio.NewFFMPEGProcessor()
	if err := audioProcessor.ExtractCoverArt(fmt.Sprintf("%s.mp3", setPath), cover); err != nil {
		log.Fatal(err)
	}

	tracklistProcessor := tracklist.NewProcessor(audioProcessor)

	if err := tracklistProcessor.ProcessTracks(
		tl.Tracks,
		&tracklist.ProcessOptions{
			InputFile:          fmt.Sprintf("%s.mp3", setPath),
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
