package tracklist

import (
	"encoding/json"
	"fmt"
	"io"
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
	cacheDir         string
	cacheTTL         time.Duration
	maxRetries       int
	baseDelay        time.Duration
	userAgents       []string
	cookieFile       string
	twoCaptchaAPIKey string   // 2Captcha API key
	proxyURLs        []string // Proxy support retained
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
		},
		cookieFile:       "./cookies.json",
		twoCaptchaAPIKey: apiKey,
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
	var tracklist domain.Tracklist
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
		colly.MaxDepth(1),
		colly.UserAgent(t.userAgents[rand.Intn(len(t.userAgents))]),
	)

	// Set a reasonable timeout
	c.SetRequestTimeout(30 * time.Second)

	if err := t.loadCookies(c); err != nil {
		slog.Warn("Failed to load cookies", "error", err)
	}

	if len(t.proxyURLs) > 0 {
		randProxy := t.proxyURLs[rand.Intn(len(t.proxyURLs))]
		c.WithTransport(&http.Transport{
			Proxy: http.ProxyURL(mustParseURL(randProxy)),
		})
	}

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "en-US,en;q=0.5")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Referer", "https://www.1001tracklists.com/")
		r.Headers.Set("Cache-Control", "max-age=0")
	})

	c.OnResponse(func(r *colly.Response) {
		// Check if response contains "captcha" in the body
		if strings.Contains(string(r.Body), "captcha") {
			slog.Info("CAPTCHA detected in response")
		}
		// Save cookies for future use
		if err := t.saveCookies(c); err != nil {
			slog.Warn("Warning: failed to save cookies", "error", err)
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

				if t.twoCaptchaAPIKey == "" {
					slog.Error("Cannot solve CAPTCHA: API key not provided")
					return
				}

				base64Image := strings.TrimPrefix(imgSrc, "data:png;base64,")
				solution, err := t.solveImageCaptcha(base64Image)
				if err != nil {
					slog.Error("Failed to solve CAPTCHA", "error", err)
					return
				}

				// Submit CAPTCHA solution via POST
				slog.Info("Submitting CAPTCHA solution", "solution", solution)

				err = e.Request.Post(url, map[string]string{
					"captcha": solution,
					"cSalt":   cSalt,
				})

				if err != nil {
					slog.Error("Failed to submit CAPTCHA solution", "error", err)
					return
				}
				slog.Info("CAPTCHA solution submitted, waiting for response")
			}
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

	// Check for JavaScript requirement or blocked content
	c.OnHTML("noscript", func(e *colly.HTMLElement) {
		if strings.Contains(e.Text, "Please enable JavaScript") {
			slog.Warn("Website requires JavaScript to be enabled")
		}
	})

	// Add more extensive CAPTCHA detection logging
	c.OnResponse(func(r *colly.Response) {
		bodyText := string(r.Body)

		// Log more detailed information about potential CAPTCHA detection
		if strings.Contains(bodyText, "captcha") {
			slog.Info("CAPTCHA keyword detected in response body")

			// Save the HTML for inspection if needed
			if err := os.WriteFile("captcha_debug.html", r.Body, 0644); err != nil {
				slog.Warn("Warning: failed to write captcha debug file", "error", err)
			}
		}

		// Check for other potential CAPTCHA indicators
		if strings.Contains(bodyText, "security check") ||
			strings.Contains(bodyText, "verify you are human") ||
			strings.Contains(bodyText, "robot") {
			slog.Info("Possible CAPTCHA/security check detected through keywords")
		}

		// Save cookies for future use
		if err := t.saveCookies(c); err != nil {
			slog.Warn("Warning: failed to save cookies", "error", err)
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
		slog.Info("Successfully scraped tracklist", "trackCount", len(tracklist.Tracks))
	}

	return &tracklist, nil
}

// solveImageCaptcha submits an image CAPTCHA to 2Captcha and retrieves the solution
func (t *tracklists1001Importer) solveImageCaptcha(base64Image string) (string, error) {
	if t.twoCaptchaAPIKey == "" {
		return "", fmt.Errorf("2Captcha API key not provided")
	}

	client := &http.Client{Timeout: 60 * time.Second}

	// Step 1: Submit CAPTCHA image
	reqURL := "https://2captcha.com/in.php"
	data := url.Values{
		"key":    {t.twoCaptchaAPIKey},
		"method": {"base64"},
		"body":   {base64Image},
		"json":   {"1"},
	}
	resp, err := client.PostForm(reqURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to submit CAPTCHA: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse JSON response
	var submitResp struct {
		Status  int    `json:"status"`
		Request string `json:"request"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &submitResp); err != nil {
		return "", fmt.Errorf("failed to parse 2Captcha response: %w", err)
	}

	if submitResp.Status != 1 {
		return "", fmt.Errorf("2Captcha submission failed: %s", submitResp.Error)
	}

	captchaID := submitResp.Request

	// Step 2: Poll for solution with exponential backoff
	maxAttempts := 20
	baseWait := 5 * time.Second

	for attempt := 0; attempt < maxAttempts; attempt++ {
		waitTime := baseWait + time.Duration(attempt)*time.Second
		slog.Info("Waiting for CAPTCHA solution", "attempt", attempt+1, "wait", waitTime)
		time.Sleep(waitTime)

		pollURL := fmt.Sprintf("https://2captcha.com/res.php?key=%s&action=get&id=%s&json=1",
			t.twoCaptchaAPIKey, captchaID)

		resp, err = client.Get(pollURL)
		if err != nil {
			slog.Warn("Failed to poll CAPTCHA solution", "error", err)
			continue
		}

		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			slog.Warn("Failed to read response", "error", err)
			continue
		}

		var pollResp struct {
			Status  int    `json:"status"`
			Request string `json:"request"`
			Error   string `json:"error"`
		}

		if err := json.Unmarshal(body, &pollResp); err != nil {
			slog.Warn("Failed to parse response", "error", err)
			continue
		}

		if pollResp.Status == 1 {
			slog.Info("CAPTCHA solved successfully")
			return pollResp.Request, nil
		}

		if pollResp.Error != "CAPCHA_NOT_READY" {
			return "", fmt.Errorf("2Captcha error: %s", pollResp.Error)
		}
	}

	return "", fmt.Errorf("CAPTCHA solution timed out after %d attempts", maxAttempts)
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
