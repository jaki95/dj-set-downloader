package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/scraper"
)

func main() {
	url := "https://www.1001tracklists.com/tracklist/1zgd4cst/adriatique-bbc-radio-1-essential-mix-2017-02-04.html"

	tracklist, err := scraper.GetTracklist(url)
	if err != nil {
		log.Fatal(err)
	}

	albumName := "Adriatique - BBC Radio 1 Essential Mix - 2017-02-04"
	inputFile := "/Users/jaki/Projects/dj-set-downloader/data/Adriatique - BBC Radio 1 Essential Mix - 2017-02-04[medium].ogg"

	for i, t := range tracklist.Tracks {
		safeTitle := sanitizeTitle(t.Title)
		outputFile := fmt.Sprintf("%02d - %s", i+1, safeTitle)

		audioProcessor := audio.NewFFMPEGProcessor()

		splitParams := audio.SplitParams{
			InputPath:  inputFile,
			OutputPath: outputFile,
			Track:      *t,
			TrackCount: len(tracklist.Tracks),
			Artist:     "Adriatique",
			Name:       albumName,
		}

		audioProcessor.Split(splitParams)

	}
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}
