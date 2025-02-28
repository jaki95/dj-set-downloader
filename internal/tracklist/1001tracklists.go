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
	"github.com/chromedp/chromedp"
	"github.com/gocolly/colly"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type tracklists1001Importer struct {
	cacheDir   string
	cacheTTL   time.Duration
	maxRetries int
	baseDelay  time.Duration
	userAgents []string
	cookieFile string
}

func New1001TracklistsImporter() *tracklists1001Importer {
	return &tracklists1001Importer{
		cacheDir:   "./internal/tracklist/cache",
		cacheTTL:   24 * time.Hour,
		maxRetries: 4,
		baseDelay:  2 * time.Second,
		userAgents: []string{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36",
		},
		cookieFile: "./cookies.json",
	}
}

func (t *tracklists1001Importer) Import(url string) (*domain.Tracklist, error) {
	var tracklist domain.Tracklist
	cacheFile := filepath.Join(t.cacheDir, strings.ReplaceAll(url, "/", "")+".json")

	if err := os.MkdirAll(t.cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	if cachedTracklist, err := t.loadFromCache(cacheFile); err == nil {
		slog.Debug("Using cached tracklist data", "url", url)
		return cachedTracklist, nil
	}

	// Try Colly first
	tracklistPtr, err := t.scrapeWithColly(url)
	if err != nil {
		slog.Warn("Colly failed, falling back to headless browser", "error", err)
		tracklistPtr, err = t.scrapeWithHeadless(url)
		if err != nil {
			return nil, fmt.Errorf("all scraping attempts failed: %w", err)
		}
	}
	tracklist = *tracklistPtr

	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in tracklist")
	}

	if err := t.saveToCache(cacheFile, &tracklist); err != nil {
		slog.Warn("Failed to cache tracklist", "error", err)
	}

	return &tracklist, nil
}

func (t *tracklists1001Importer) scrapeWithColly(url string) (*domain.Tracklist, error) {
	var tracklist domain.Tracklist
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
	)

	// Set request headers
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", t.userAgents[rand.Intn(len(t.userAgents))])
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Referer", "https://www.google.com/")
	})

	c.OnError(func(r *colly.Response, err error) {
		slog.Error("Request failed", "url", r.Request.URL, "status", r.StatusCode, "error", err)
		if r.StatusCode == 403 || strings.Contains(string(r.Body), "captcha") {
			slog.Info("Detected CAPTCHA or block. Please solve CAPTCHA manually and save cookies to", "file", t.cookieFile)
		}
	})

	trackCounter := 1
	c.OnHTML("div.tlpTog", func(e *colly.HTMLElement) {
		startTime := strings.TrimSpace(e.ChildText("div.cue.noWrap.action.mt5"))
		if startTime == "" {
			startTime = "00:00"
		}

		trackValue := strings.TrimSpace(e.ChildText("span.trackValue"))
		artist, title := parseTrackValue(trackValue)

		track := &domain.Track{
			Artist:      artist,
			Title:       title,
			StartTime:   startTime,
			EndTime:     "",
			TrackNumber: trackCounter,
		}

		if trackCounter > 1 && len(tracklist.Tracks) > 0 {
			tracklist.Tracks[trackCounter-2].EndTime = startTime
		}

		tracklist.Tracks = append(tracklist.Tracks, track)
		trackCounter++
	})

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

	slog.Info("Fetching tracklist data with Colly", "url", url)
	err := t.visitWithRetries(c, url)
	if err != nil {
		return nil, err
	}

	if len(tracklist.Tracks) > 0 {
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = ""
	}

	return &tracklist, nil
}

func (t *tracklists1001Importer) scrapeWithHeadless(url string) (*domain.Tracklist, error) {
	// Use a visible browser for manual CAPTCHA solving
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(slog.Info))
	defer cancel()

	// Make browser visible (not headless) for interaction
	ctx, cancel = chromedp.NewExecAllocator(ctx, chromedp.NoDefaultBrowserCheck, chromedp.Flag("headless", false))
	defer cancel()

	var htmlContent string
	slog.Info("Opening browser for manual CAPTCHA solving if needed. Close the browser when done.")
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible("div.tlpTog", chromedp.ByQuery), // Wait for tracklist or CAPTCHA
		chromedp.OuterHTML("html", &htmlContent),
	)
	if err != nil {
		return nil, fmt.Errorf("headless browser failed: %w", err)
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}

	var tracklist domain.Tracklist
	trackCounter := 1
	doc.Find("div.tlpTog").Each(func(i int, s *goquery.Selection) {
		startTime := strings.TrimSpace(s.Find("div.cue.noWrap.action.mt5").Text())
		if startTime == "" {
			startTime = "00:00"
		}

		trackValue := strings.TrimSpace(s.Find("span.trackValue").Text())
		artist, title := parseTrackValue(trackValue)

		track := &domain.Track{
			Artist:      artist,
			Title:       title,
			StartTime:   startTime,
			EndTime:     "",
			TrackNumber: trackCounter,
		}

		if trackCounter > 1 && len(tracklist.Tracks) > 0 {
			tracklist.Tracks[trackCounter-2].EndTime = startTime
		}

		tracklist.Tracks = append(tracklist.Tracks, track)
		trackCounter++
	})

	doc.Find("div#pageTitle h1").Each(func(i int, s *goquery.Selection) {
		fullText := strings.TrimSpace(s.Text())
		var artists []string
		s.Find("a[href*='/dj/']").Each(func(_ int, sel *goquery.Selection) {
			artists = append(artists, strings.TrimSpace(sel.Text()))
		})

		setName := extractSetName(fullText)
		tracklist.Artist = strings.Join(artists, " & ")
		tracklist.Name = setName
	})

	if len(tracklist.Tracks) > 0 {
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = ""
	}

	return &tracklist, nil
}

func (t *tracklists1001Importer) visitWithRetries(c *colly.Collector, url string) error {
	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<uint(attempt))
			jitter := time.Duration(rand.Intn(5000)) * time.Millisecond // Up to 5s jitter
			time.Sleep(delay + jitter)
			slog.Info("Retrying request", "attempt", attempt, "url", url)
		}

		lastErr = c.Visit(url)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// Helper functions (unchanged)
func (t *tracklists1001Importer) loadFromCache(filePath string) (*domain.Tracklist, error) {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) || time.Since(info.ModTime()) > t.cacheTTL {
		return nil, fmt.Errorf("cache miss or expired")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tracklist domain.Tracklist
	if err := json.NewDecoder(file).Decode(&tracklist); err != nil {
		return nil, err
	}
	return &tracklist, nil
}

func (t *tracklists1001Importer) saveToCache(filePath string, tracklist *domain.Tracklist) error {
	jsonBytes, err := json.MarshalIndent(tracklist, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, jsonBytes, 0644)
}

func parseTrackValue(trackValue string) (artist, title string) {
	parts := strings.SplitN(trackValue, " - ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "Unknown Artist", strings.TrimSpace(trackValue)
}

func extractSetName(fullText string) string {
	re := regexp.MustCompile(`[@-] (.+?)( \d{4}|$)`)
	matches := re.FindStringSubmatch(fullText)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return strings.SplitN(fullText, " - ", 2)[1]
}
