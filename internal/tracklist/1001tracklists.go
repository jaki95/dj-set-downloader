package tracklist

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type tracklists1001Importer struct {
	cacheDir        string
	cacheTTL        time.Duration
	maxRetries      int
	baseDelay       time.Duration
	userAgents      []string
	cookieFile      string
	proxyURLs       []string
	lastRequestTime time.Time
}

func New1001TracklistsImporter() *tracklists1001Importer {
	// Load 2Captcha API key from environment variable for security
	apiKey := os.Getenv("TWOCAPTCHA_API_KEY")
	if apiKey == "" {
		slog.Warn("No 2Captcha API key provided in environment variable TWOCAPTCHA_API_KEY")
	}

	return &tracklists1001Importer{
		cacheDir:   "./internal/tracklist/cache",
		cacheTTL:   24 * time.Hour,
		maxRetries: 4,
		baseDelay:  2 * time.Second,
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

	tracklistPtr, err := t.scrapeWithColly(url)
	if err != nil {
		return nil, fmt.Errorf("scraping failed: %w", err)
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
	slog.Debug("Starting to scrape tracklist", "url", url)
	var tracklist domain.Tracklist

	// First visit the homepage to get cookies and establish a session
	homepageCollector := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowedDomains("www.1001tracklists.com", "1001tracklists.com"),
	)

	// Set up proxy if available
	var transport *http.Transport
	if len(t.proxyURLs) > 0 {
		randProxy := t.proxyURLs[rand.Intn(len(t.proxyURLs))]
		transport = &http.Transport{
			Proxy: http.ProxyURL(mustParseURL(randProxy)),
		}
		homepageCollector.WithTransport(transport)
		slog.Debug("Using proxy", "proxy", randProxy)
	}

	// Configure homepage collector
	homepageCollector.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Sec-Ch-Ua", "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"")
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
	})

	// Visit homepage first
	slog.Debug("Visiting homepage to establish session")
	if err := homepageCollector.Visit("https://www.1001tracklists.com"); err != nil {
		slog.Warn("Failed to visit homepage", "error", err)
	}

	// Add random delay to simulate human behavior (3-7 seconds)
	time.Sleep(time.Duration(3000+rand.Intn(4000)) * time.Millisecond)

	// Create a new collector for the actual tracklist page
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		colly.AllowedDomains("www.1001tracklists.com", "1001tracklists.com"),
	)

	// Use the same transport settings if proxy was configured
	if transport != nil {
		c.WithTransport(transport)
	}

	// Set a reasonable timeout
	c.SetRequestTimeout(30 * time.Second)

	// Add CloudFlare bypass headers
	c.OnRequest(func(r *colly.Request) {
		slog.Debug("Making request", "url", r.URL.String(), "method", r.Method)

		// Calculate time since last request
		timeSinceLast := time.Since(t.lastRequestTime)
		// Add random delay between 3-7 seconds with microsecond precision
		delay := time.Duration(3000+rand.Intn(4000)) * time.Millisecond
		if timeSinceLast < delay {
			slog.Debug("Adding delay between requests", "delay", delay-timeSinceLast)
			time.Sleep(delay - timeSinceLast)
		}
		t.lastRequestTime = time.Now()

		// Set realistic browser headers
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.9")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "same-origin")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Cache-Control", "max-age=0")
		r.Headers.Set("DNT", "1")
		r.Headers.Set("Sec-Ch-Ua", "\"Not_A Brand\";v=\"8\", \"Chromium\";v=\"120\", \"Google Chrome\";v=\"120\"")
		r.Headers.Set("Sec-Ch-Ua-Mobile", "?0")
		r.Headers.Set("Sec-Ch-Ua-Platform", "\"macOS\"")
		r.Headers.Set("Referer", "https://www.1001tracklists.com/")

		// Add cookies from file
		if err := t.loadCookies(c); err != nil {
			slog.Warn("Failed to load cookies", "error", err)
		} else {
			slog.Debug("Successfully loaded cookies")
		}

		// Copy cookies from homepage collector
		for _, cookie := range homepageCollector.Cookies(r.URL.String()) {
			r.Headers.Set("Cookie", cookie.String())
		}
	})

	c.OnResponse(func(r *colly.Response) {
		slog.Debug("Received response", "url", r.Request.URL.String(), "status", r.StatusCode)

		// Save cookies after each response
		if err := t.saveCookies(c); err != nil {
			slog.Warn("Failed to save cookies", "error", err)
		} else {
			slog.Debug("Successfully saved cookies")
		}

		// Check for various bot detection indicators
		bodyText := string(r.Body)
		if strings.Contains(bodyText, "captcha") ||
			strings.Contains(bodyText, "security check") ||
			strings.Contains(bodyText, "verify you are human") ||
			strings.Contains(bodyText, "robot") ||
			strings.Contains(bodyText, "access denied") ||
			strings.Contains(bodyText, "blocked") {
			slog.Warn("Bot detection triggered", "url", r.Request.URL.String())

			// Add additional delay when bot detection is triggered (7-15 seconds)
			delay := time.Duration(7000+rand.Intn(8000)) * time.Millisecond
			slog.Debug("Adding delay after bot detection", "delay", delay)
			time.Sleep(delay)
		}

		// Check for CloudFlare specific headers in response
		cfRay := r.Headers.Get("CF-RAY")
		if cfRay != "" {
			slog.Debug("CloudFlare Ray ID", "ray", cfRay)
		}
	})

	// Improved CAPTCHA detection and handling
	c.OnHTML("form[action='']", func(e *colly.HTMLElement) {
		// Check if this is actually a CAPTCHA form
		if e.DOM.Find("input[name='captcha']").Length() > 0 {
			// Extract CAPTCHA image and salt
			imgSrc := e.DOM.Find("img").AttrOr("src", "")
			cSalt := e.DOM.Find("input[name='cSalt']").AttrOr("value", "")
			if strings.HasPrefix(imgSrc, "data:png;base64,") && cSalt != "" {
				slog.Info("CAPTCHA detected", "salt", cSalt)

			}
		}
	})

	trackCounter := 1
	c.OnHTML("div.tlpTog", func(e *colly.HTMLElement) {
		slog.Debug("Found track element", "trackNumber", trackCounter)
		startTime := strings.TrimSpace(e.ChildText("div.cue.noWrap.action.mt5"))
		if startTime == "" {
			startTime = "00:00"
		}

		trackValue := strings.TrimSpace(e.ChildText("span.trackValue"))
		artist, title := parseTrackValue(trackValue)
		slog.Debug("Parsed track", "artist", artist, "title", title, "startTime", startTime)

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

	// Check for JavaScript requirement or blocked content
	c.OnHTML("noscript", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, "Please enable JavaScript") {
			slog.Warn("Website requires JavaScript to be enabled")
		}
	})

	// Add more general CAPTCHA form detection
	c.OnHTML("form", func(e *colly.HTMLElement) {
		// Log form details to help debug
		slog.Debug("Found a form", "action", e.Attr("action"), "id", e.Attr("id"))

		// Check for any image elements that might be CAPTCHA images
		e.DOM.Find("img").Each(func(_ int, s *goquery.Selection) {
			src, exists := s.Attr("src")
			if exists {
				if strings.Contains(src, "captcha") || strings.HasPrefix(src, "data:image") {
					slog.Info("Potential CAPTCHA image found", "src", src)
				}
			}
		})

		// Check for CAPTCHA-related input fields with more general patterns
		if e.DOM.Find("input[name*='captcha']").Length() > 0 ||
			e.DOM.Find("input[id*='captcha']").Length() > 0 {
			slog.Info("CAPTCHA input field detected in form")
		}
	})

	slog.Info("Fetching tracklist data with Colly", "url", url)
	err := t.visitWithRetries(c, url)
	if err != nil {
		return nil, err
	}

	// Additional check for empty tracklist
	if len(tracklist.Tracks) == 0 && (tracklist.Artist == "" || tracklist.Name == "") {
		slog.Warn("Possibly blocked by anti-scraping measures", "trackCount", len(tracklist.Tracks))
	} else {
		if len(tracklist.Tracks) > 0 {
			tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = ""
		}
		slog.Debug("Tracklist", "tracklist", tracklist.Tracks)
		slog.Info("Successfully scraped tracklist", "trackCount", len(tracklist.Tracks))
	}

	return &tracklist, nil
}

func (t *tracklists1001Importer) loadCookies(c *colly.Collector) error {
	if _, err := os.Stat(t.cookieFile); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(t.cookieFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var cookies []*http.Cookie
	if err := json.NewDecoder(file).Decode(&cookies); err != nil {
		return err
	}

	// Filter out expired cookies
	var validCookies []*http.Cookie
	now := time.Now()
	for _, cookie := range cookies {
		if cookie.Expires.IsZero() || cookie.Expires.After(now) {
			validCookies = append(validCookies, cookie)
		}
	}

	if err := c.SetCookies("https://www.1001tracklists.com", validCookies); err != nil {
		slog.Warn("Warning: failed to set cookies", "error", err)
	}
	slog.Info("Loaded cookies", "count", len(validCookies), "file", t.cookieFile)
	return nil
}

func (t *tracklists1001Importer) saveCookies(c *colly.Collector) error {
	cookies := c.Cookies("https://www.1001tracklists.com")
	if len(cookies) == 0 {
		return nil
	}

	jsonBytes, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(t.cookieFile, jsonBytes, 0644); err != nil {
		return err
	}
	slog.Info("Saved cookies", "count", len(cookies), "file", t.cookieFile)
	return nil
}

func (t *tracklists1001Importer) visitWithRetries(c *colly.Collector, url string) error {
	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		if attempt > 0 {
			delay := t.baseDelay * time.Duration(1<<uint(attempt))
			jitter := time.Duration(rand.Int63n(3000)) * time.Millisecond
			totalDelay := delay + jitter
			slog.Info("Retrying request", "attempt", attempt+1, "delay", totalDelay.String(), "url", url)
			time.Sleep(totalDelay)
		}

		lastErr = c.Visit(url)
		if lastErr == nil {
			return nil
		}
		slog.Warn("Request failed", "attempt", attempt+1, "error", lastErr)
	}
	return fmt.Errorf("failed after %d attempts: %w", t.maxRetries, lastErr)
}

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
	// Try first with standard format
	re := regexp.MustCompile(`[@-] (.+?)( \d{4}|$)`)
	matches := re.FindStringSubmatch(fullText)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Fallback to simple split
	parts := strings.SplitN(fullText, " - ", 2)
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}

	// Last resort
	return strings.TrimSpace(fullText)
}

func mustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}
