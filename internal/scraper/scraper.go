package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/jaki95/dj-set-downloader/pkg"
)

func GetTracklist(url string) (*pkg.Tracklist, error) {
	var tracklist pkg.Tracklist

	cachedFile, err := os.Open(fmt.Sprintf("/Users/jaki/Projects/dj-set-downloader/internal/scraper/cache/%s.json", strings.ReplaceAll(url, "/", "")))
	if err == nil {
		defer cachedFile.Close()
		byteValue, err := io.ReadAll(cachedFile)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(byteValue, &tracklist); err == nil {
			fmt.Println("using cached data")
			return &tracklist, nil
		}
	}

	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.Async(false),
	)

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Referer", "https://www.google.com/")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Cache-Control", "max-age=0")
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %v failed with response: %v\nError: %v", r.Request.URL, &r, err)
	})

	trackCounter := 1

	c.OnHTML("div.tlpTog", func(e *colly.HTMLElement) {
		// Extract timestamp
		startTime := strings.TrimSpace(e.ChildText("div.cue.noWrap.action.mt5"))
		if startTime == "" {
			startTime = "00:00"
		}

		// Extract artist and track title
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

		// Create the current track
		track := &pkg.Track{
			Artist:      artist,
			Title:       title,
			StartTime:   startTime,
			EndTime:     "",
			TrackNumber: trackCounter,
		}

		// Update previous track's end time
		if trackCounter > 1 && len(tracklist.Tracks) > 0 {
			prevTrack := tracklist.Tracks[trackCounter-2]
			if prevTrack != nil {
				prevTrack.EndTime = startTime
			}
		}

		// Append the current track
		tracklist.Tracks = append(tracklist.Tracks, track)
		trackCounter++
	})

	// Visit the URL and scrape data
	err = c.Visit(url)
	if err != nil {
		for range 10 {
			time.Sleep(2 * time.Second)
			err = c.Visit(url)

		}
		if err != nil {
			return nil, fmt.Errorf("failed after retry: %w", err)
		}

	}

	// Handle last track
	if len(tracklist.Tracks) > 0 {
		lastTrack := tracklist.Tracks[len(tracklist.Tracks)-1]
		lastTrack.EndTime = lastTrack.StartTime + "+5:00"
	}

	// Validate we got all tracks
	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in the tracklist")
	}

	jsonBytes, err := json.Marshal(tracklist)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(fmt.Sprintf("/Users/jaki/Projects/dj-set-downloader/internal/scraper/cache/%s.json", strings.ReplaceAll(url, "/", "")), jsonBytes, 0644)
	if err != nil {
		return nil, err
	}

	return &tracklist, nil
}
