package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/gocolly/colly"
)

type Track struct {
	Artist string
	Title  string
}

func main() {
	url := "https://www.1001tracklists.com/tracklist/1zgd4cst/adriatique-bbc-radio-1-essential-mix-2017-02-04.html"
	c := colly.NewCollector(colly.AllowURLRevisit())

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	})

	var timestamps []string
	var tracks []Track

	c.OnHTML("div.tlpTog", func(e *colly.HTMLElement) {
		timestamp := strings.TrimSpace(e.ChildText("div.cue.noWrap.action.mt5"))
		if timestamp == "" {
			timestamp = "00:00"
		}
		timestamps = append(timestamps, timestamp)

		trackValue := e.ChildText("span.trackValue")
		parts := strings.Split(trackValue, " - ")

		var artist, title string
		if len(parts) > 1 {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
		} else {
			artist = "Unknown Artist"
			title = strings.TrimSpace(parts[0])
		}

		tracks = append(tracks, Track{Artist: artist, Title: title})
		fmt.Printf("Timestamp: %s | Artist: %s | Track: %s\n", timestamp, artist, title)
	})

	err := c.Visit(url)
	if err != nil {
		log.Fatal(err)
	}

	albumName := "Adriatique - BBC Radio 1 Essential Mix - 2017-02-04"
	inputFile := "Adriatique - BBC Radio 1 Essential Mix - 2017-02-04[medium].ogg"

	for i, t := range timestamps {
		var end string
		if i == len(timestamps)-1 {
			end = "01:59:14"
		} else {
			end = timestamps[i+1]
		}

		track := tracks[i]
		safeTitle := sanitizeTitle(track.Title)
		outputFile := fmt.Sprintf("%02d - %s.mp3", i+1, safeTitle)

		splitAudio(inputFile, t, end, outputFile, track.Artist, track.Title, i+1, len(tracks), albumName)
	}
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}

func splitAudio(inputFile, start, end, outputFile, artist, title string, trackNumber, totalTracks int, album string) {
	args := []string{
		"-i", inputFile,
		"-ss", start,
		"-to", end,
		"-c", "copy",
		"-metadata", fmt.Sprintf("album_artist=%s", "Adriatique"),
		"-metadata", fmt.Sprintf("artist=%s", artist),
		"-metadata", fmt.Sprintf("title=%s", title),
		"-metadata", fmt.Sprintf("track=%d/%d", trackNumber, totalTracks),
		"-metadata", fmt.Sprintf("album=%s", album),
		outputFile,
	}
	cmd := exec.Command("ffmpeg", args...)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error splitting audio for %s: %v\n", outputFile, err)
	}
}
