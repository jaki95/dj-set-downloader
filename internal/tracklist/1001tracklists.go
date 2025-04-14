package tracklist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/search"
)

type tracklists1001Importer struct {
	searchClient    search.GoogleClient
	cacheDir        string
	cacheTTL        time.Duration
	maxRetries      int
	baseDelay       time.Duration
	userAgents      []string
	cookieFile      string
	lastRequestTime time.Time
}

func (t *tracklists1001Importer) Name() string {
	return "1001tracklists"
}

func New1001TracklistsImporter(searchClient search.GoogleClient) (*tracklists1001Importer, error) {
	cacheDir := filepath.Join(os.TempDir(), "dj-set-downloader", "1001tracklists")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &tracklists1001Importer{
		searchClient: searchClient,
		cacheDir:     cacheDir,
		cacheTTL:     24 * time.Hour,
		maxRetries:   4,
		baseDelay:    2 * time.Second,
		userAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/120.0",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
		},
		cookieFile:      "./cookies.json",
		lastRequestTime: time.Now(),
	}, nil
}

func (i *tracklists1001Importer) Import(ctx context.Context, query string) (*domain.Tracklist, error) {
	slog.Info("Importing tracklist", "query", query)

	// First try to find the tracklist URL using Google search
	searchQuery := fmt.Sprintf("site:1001tracklists.com %s", query)
	results, err := i.searchClient.Search(ctx, searchQuery, "1001tracklists")
	if err != nil {
		return nil, fmt.Errorf("failed to search for tracklist: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no tracklist found for query: %s", query)
	}

	// Check if we have a cached version of this tracklist
	cacheFile := filepath.Join(i.cacheDir, fmt.Sprintf("%x.json", results[0].Link))
	if data, err := os.ReadFile(cacheFile); err == nil {
		var cachedTracklist domain.Tracklist
		if err := json.Unmarshal(data, &cachedTracklist); err == nil {
			slog.Debug("Using cached tracklist", "url", results[0].Link)
			return &cachedTracklist, nil
		}
	}

	// If not cached, scrape the tracklist
	tracklist, err := i.scrapeWithColly(ctx, results[0].Link)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape tracklist: %w", err)
	}

	// Cache the tracklist
	if data, err := json.Marshal(tracklist); err == nil {
		if err := os.WriteFile(cacheFile, data, 0644); err != nil {
			slog.Warn("Failed to cache tracklist", "error", err)
		}
	}

	return tracklist, nil
}

func (t *tracklists1001Importer) scrapeWithColly(ctx context.Context, url string) (*domain.Tracklist, error) {
	slog.Debug("Starting to scrape tracklist", "url", url)
	tracklist := &domain.Tracklist{
		Tracks: make([]*domain.Track, 0),
	}

	c := colly.NewCollector(
		colly.UserAgent(t.userAgents[0]),
		colly.AllowURLRevisit(),
		colly.AllowedDomains("www.1001tracklists.com", "1001tracklists.com"),
	)

	// Set up request headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
	})

	// Extract tracklist metadata
	c.OnHTML("div#pageTitle h1", func(e *colly.HTMLElement) {
		fullText := strings.TrimSpace(e.Text)
		var artists []string
		e.DOM.Find("a[href*='/dj/']").Each(func(_ int, s *goquery.Selection) {
			artists = append(artists, strings.TrimSpace(s.Text()))
		})

		setName := extractSetName(fullText)
		tracklist.Artist = strings.Join(artists, " & ")
		tracklist.Name = setName
		slog.Info("Extracted metadata", "artists", tracklist.Artist, "setName", setName)
	})

	// Extract tracks with timing information
	trackCounter := 1
	var lastTrack *domain.Track

	c.OnHTML("div.tlpTog", func(e *colly.HTMLElement) {
		slog.Debug("Found track element", "trackNumber", trackCounter)

		// Get start time and handle empty values
		startTime := strings.TrimSpace(e.ChildText("div.cue.noWrap.action.mt5"))
		if startTime == "" {
			// If this is the first track, use 00:00
			if trackCounter == 1 {
				startTime = "00:00"
			} else if lastTrack != nil {
				// Otherwise, use the end time of the previous track
				startTime = lastTrack.EndTime
			}
		}

		// Get track value and handle non-breaking spaces
		trackValue := strings.TrimSpace(e.ChildText("span.trackValue"))
		trackValue = strings.ReplaceAll(trackValue, "\u00A0", " ") // Replace non-breaking spaces
		artist, title := parseTrackValue(trackValue)
		if artist == "" || title == "" {
			slog.Warn("Invalid track value", "trackValue", trackValue)
			return
		}

		slog.Debug("Parsed track", "artist", artist, "title", title, "startTime", startTime)

		track := &domain.Track{
			Artist:      artist,
			Title:       title,
			StartTime:   normaliseTime(startTime),
			TrackNumber: trackCounter,
		}

		// If we have a previous track, set its end time to this track's start time
		if lastTrack != nil {
			lastTrack.EndTime = track.StartTime
		}

		tracklist.Tracks = append(tracklist.Tracks, track)
		lastTrack = track
		trackCounter++
	})

	// Handle errors
	c.OnError(func(r *colly.Response, err error) {
		slog.Error("Scraping error", "url", r.Request.URL, "error", err)
	})

	// Visit the URL with retries
	if err := t.visitWithRetries(ctx, c, url); err != nil {
		return nil, fmt.Errorf("failed to visit URL: %w", err)
	}

	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in tracklist")
	}

	// Set end time for last track to empty string
	if lastTrack != nil {
		lastTrack.EndTime = ""
	}

	return tracklist, nil
}

func (t *tracklists1001Importer) visitWithRetries(ctx context.Context, c *colly.Collector, url string) error {
	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<uint(attempt))
			jitter := time.Duration(rand.Int63n(3000)) * time.Millisecond
			totalDelay := delay + jitter
			slog.Info("Retrying request", "attempt", attempt+1, "delay", totalDelay.String(), "url", url)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(totalDelay):
			}
		}

		lastErr = c.Visit(url)
		if lastErr == nil {
			return nil
		}
		slog.Warn("Request failed", "attempt", attempt+1, "error", lastErr)
	}
	return fmt.Errorf("failed after %d attempts: %w", t.maxRetries, lastErr)
}

func parseTrackValue(trackValue string) (string, string) {
	parts := strings.SplitN(trackValue, " - ", 2)
	if len(parts) != 2 {
		return "", trackValue
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func extractSetName(fullText string) string {
	// Remove artist names and common suffixes
	name := fullText
	name = strings.ReplaceAll(name, "Tracklist", "")
	name = strings.ReplaceAll(name, "Track List", "")
	name = strings.ReplaceAll(name, "Set", "")
	name = strings.TrimSpace(name)
	return name
}

// normaliseTime converts various time formats to a consistent format
func normaliseTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	// Remove any leading/trailing whitespace
	timeStr = strings.TrimSpace(timeStr)

	// If it's already in the correct format (HH:MM:SS or MM:SS), return as is
	if matched, _ := regexp.MatchString(`^(\d{1,2}:)?[0-5]?\d:[0-5]\d$`, timeStr); matched {
		return timeStr
	}

	// Try to parse the time
	var hours, minutes, seconds int
	if _, err := fmt.Sscanf(timeStr, "%d:%d:%d", &hours, &minutes, &seconds); err == nil {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	if _, err := fmt.Sscanf(timeStr, "%d:%d", &minutes, &seconds); err == nil {
		if minutes >= 60 {
			hours = minutes / 60
			minutes = minutes % 60
			return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
		}
		return fmt.Sprintf("%02d:%02d", minutes, seconds)
	}

	// If we can't parse it, return the original string
	return timeStr
}
