package tracklist

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type tracklists1001Importer struct {
	cacheDir string
}

func New1001TracklistsImporter() *tracklists1001Importer {
	return &tracklists1001Importer{cacheDir: "./internal/tracklist/cache"}
}

func (t *tracklists1001Importer) Import(url string) (*domain.Tracklist, error) {
	var tracklist domain.Tracklist

	if err := os.MkdirAll(t.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Attempt to load from cache
	cachedFile, err := os.Open(fmt.Sprintf("%s/%s.json", t.cacheDir, strings.ReplaceAll(url, "/", "")))
	if err == nil {
		defer cachedFile.Close()
		byteValue, err := io.ReadAll(cachedFile)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(byteValue, &tracklist); err == nil {
			slog.Debug("Using cached tracklist data...")
			return &tracklist, nil
		}
	}

	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.Async(false),
	)

	// Set headers
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
		slog.Error("request failed", "url", r.Request.URL, "response", string(r.Body), "error", err)
	})

	// Counter for track numbering
	trackCounter := 1

	// Extract track information
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
		track := &domain.Track{
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

	// Extract artist names and set name from the page title
	c.OnHTML("div#pageTitle h1", func(e *colly.HTMLElement) {
		// Get full text from the <h1>
		fullText := e.Text
		// Extract artist names by looking for <a> elements with href containing "/dj/"
		var artists []string
		e.DOM.Find("a").Each(func(i int, s *goquery.Selection) {
			href, exists := s.Attr("href")
			if exists && strings.HasPrefix(href, "/dj/") {
				artists = append(artists, s.Text())
			}
		})

		// Extract set name using regex - text after '@' or '-' including the year
		re := regexp.MustCompile(`[@-] (.+) \d{4}`)
		matches := re.FindStringSubmatch(fullText)
		setName := ""
		if len(matches) > 1 {
			setName = matches[1]
		}

		slog.Info("Extracted artists", "artists", artists)
		slog.Info("Extracted Set Name", "setName", setName)

		tracklist.Artist = strings.Join(artists, " & ")
		tracklist.Name = setName
	})

	slog.Info("Fetching tracklist data...")

	// Visit the URL and scrape data
	err = c.Visit(url)
	if err != nil {
		maxRetries := 4
		for retry := 0; retry < maxRetries; retry++ {
			time.Sleep(2 * time.Second)
			if err = c.Visit(url); err == nil {
				break
			}
		}
		if err != nil {
			return nil, fmt.Errorf("failed after retrying: %w", err)
		}
	}

	// Handle last track's end time
	if len(tracklist.Tracks) > 0 {
		lastTrack := tracklist.Tracks[len(tracklist.Tracks)-1]
		lastTrack.EndTime = ""
	}

	// Validate we got tracks
	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in the tracklist")
	}

	// Cache the result
	jsonBytes, err := json.Marshal(tracklist)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(fmt.Sprintf("%s/%s.json", t.cacheDir, strings.ReplaceAll(url, "/", "")), jsonBytes, 0644)
	if err != nil {
		return nil, err
	}

	return &tracklist, nil
}
