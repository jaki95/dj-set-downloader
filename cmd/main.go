package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/scraper"
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
		log.Fatal("url, album, and input flags are required")
	}

	url := *urlFlag
	setName := *setNameFlag
	setArtist := *setArtistFlag
	inputFile := *inputFlag

	tracklist, err := scraper.GetTracklist(url)
	if err != nil {
		log.Fatal(err)
	}

	cover := "/Users/jaki/Projects/dj-set-downloader/cover_temp.jpg"

	audioProcessor := audio.NewFFMPEGProcessor()
	if err := audioProcessor.ExtractCoverArt(inputFile, cover); err != nil {
		log.Fatal(err)
	}

	for i, t := range tracklist.Tracks {
		safeTitle := sanitizeTitle(t.Title)
		outputFile := fmt.Sprintf("%02d - %s", i+1, safeTitle)

		splitParams := audio.SplitParams{
			InputPath:    inputFile,
			OutputPath:   outputFile,
			Track:        *t,
			TrackCount:   len(tracklist.Tracks),
			Artist:       setArtist,
			Name:         setName,
			CoverArtPath: cover,
		}

		audioProcessor.Split(splitParams)

	}

	os.Remove(cover)
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}
