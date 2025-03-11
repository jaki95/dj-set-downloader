package tracklist

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type TrackIDParser struct {
	baseURL   string
	searchURL string
}

type TrackIDResponse struct {
	Result struct {
		Title     string `json:"title"`
		Duration  string `json:"duration"`
		Processes []struct {
			Tracks []Track `json:"detectionProcessMusicTracks"`
		} `json:"detectionProcesses"`
	} `json:"result"`
}

type Track struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Artist    string `json:"artist"`
	Title     string `json:"title"`
}

type TrackIDSearchResponse struct {
	Result struct {
		Audiostreams []struct {
			Slug  string `json:"slug"`
			Title string `json:"title"`
		} `json:"audiostreams"`
		RowCount int `json:"rowCount"`
	} `json:"result"`
}

func NewTrackIDParser() *TrackIDParser {
	return &TrackIDParser{
		baseURL:   "https://trackid.net:8001/api/public/audiostreams/",
		searchURL: "https://trackid.net:8001/api/public/audiostreams",
	}
}

func (t *TrackIDParser) Import(keywords string) (*domain.Tracklist, error) {
	slug, err := t.findSlug(keywords)
	if err != nil {
		return nil, err
	}

	resp, err := t.fetchTrackData(slug)
	if err != nil {
		return nil, err
	}

	tracklist, err := t.parseTracklist(resp)
	if err != nil {
		return nil, err
	}

	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in TrackID response")
	}

	return tracklist, nil
}

func (t *TrackIDParser) findSlug(keywords string) (string, error) {
	params := url.Values{}
	params.Add("keywords", keywords)
	params.Add("pageSize", "20")
	params.Add("currentPage", "0")
	params.Add("status", "3")

	url := t.searchURL + "?" + params.Encode()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}

	t.setCommonHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch search results: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read search response body: %w", err)
	}

	var searchResp TrackIDSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal search JSON: %w", err)
	}

	if searchResp.Result.RowCount == 0 {
		return "", fmt.Errorf("no matching audiostreams found for keywords: %s", keywords)
	}

	slug := searchResp.Result.Audiostreams[0].Slug
	slog.Debug("Found audiostream",
		"keywords", keywords,
		"slug", slug,
		"title", searchResp.Result.Audiostreams[0].Title)

	return slug, nil
}

func (t *TrackIDParser) fetchTrackData(slug string) (*TrackIDResponse, error) {
	url := t.baseURL + slug
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	t.setCommonHeaders(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch track data: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var trackResp TrackIDResponse
	if err := json.Unmarshal(body, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	slog.Debug("TrackID response received",
		"title", trackResp.Result.Title,
		"duration", trackResp.Result.Duration)

	return &trackResp, nil
}

func (t *TrackIDParser) setCommonHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Origin", "https://www.trackid.net")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.3 Safari/605.1.15")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Referer", "https://www.trackid.net/")
	req.Header.Set("Priority", "u=3, i")
}

func (t *TrackIDParser) parseTracklist(resp *TrackIDResponse) (*domain.Tracklist, error) {

	title := resp.Result.Title
	artist := inferArtistFromTitle(title)

	tracklist := &domain.Tracklist{
		Name:   title,
		Artist: artist,
	}
	totalDuration := resp.Result.Duration
	trackCounter := 1
	previousEndTime := ""

	var allTracks []Track
	for _, process := range resp.Result.Processes {
		allTracks = append(allTracks, process.Tracks...)
	}

	if len(allTracks) > 0 {
		if err := t.processFirstTrack(tracklist, allTracks[0], &trackCounter, &previousEndTime); err != nil {
			return nil, err
		}
	}

	for i := 1; i < len(allTracks); i++ {
		track := allTracks[i]
		t.handleTrackGap(tracklist, previousEndTime, track.StartTime, &trackCounter)
		t.addTrack(tracklist, track.Artist, track.Title, track.StartTime, track.EndTime, trackCounter)
		trackCounter++
		previousEndTime = track.EndTime
	}

	if err := t.handleFinalGap(tracklist, previousEndTime, totalDuration, &trackCounter); err != nil {
		return nil, err
	}

	return tracklist, nil
}

func (t *TrackIDParser) processFirstTrack(tracklist *domain.Tracklist, firstTrack Track, trackCounter *int, previousEndTime *string) error {
	if firstTrack.StartTime != "00:00:00" {
		t.addIDTrack(tracklist, "00:00:00", firstTrack.StartTime, *trackCounter)
		*trackCounter++
	}

	t.addTrack(tracklist, firstTrack.Artist, firstTrack.Title, firstTrack.StartTime, firstTrack.EndTime, *trackCounter)
	*trackCounter++
	*previousEndTime = firstTrack.EndTime
	return nil
}

func (t *TrackIDParser) handleTrackGap(tracklist *domain.Tracklist, previousEndTime, startTime string, trackCounter *int) {
	if previousEndTime == "" {
		return
	}

	gapDuration := t.calculateDuration(previousEndTime, startTime)
	slog.Debug("Track gap",
		"previousEnd", previousEndTime,
		"start", startTime,
		"duration", gapDuration)

	if gapDuration <= 0 {
		return
	}

	if gapDuration < 60 {
		midpointTime := t.calculateMidpoint(previousEndTime, startTime)
		slog.Debug("Gap < 60s, setting midpoint", "midpoint", midpointTime)
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = midpointTime
	} else {
		slog.Debug("Gap >= 60s, inserting ID track",
			"start", previousEndTime,
			"end", startTime)
		t.addIDTrack(tracklist, previousEndTime, startTime, *trackCounter)
		*trackCounter++
	}
}

func (t *TrackIDParser) handleFinalGap(tracklist *domain.Tracklist, previousEndTime, totalDuration string, trackCounter *int) error {
	if previousEndTime == "" || previousEndTime == totalDuration {
		return nil
	}

	finalGap := t.calculateDuration(previousEndTime, totalDuration)
	if finalGap <= 0 || finalGap >= 86400 {
		return fmt.Errorf("invalid final gap (%d seconds)", finalGap)
	}

	if finalGap < 60 {
		slog.Debug("Final gap < 60s, extending last track", "end", totalDuration)
		tracklist.Tracks[len(tracklist.Tracks)-1].EndTime = totalDuration
	} else {
		slog.Debug("Final gap >= 60s, inserting final ID track",
			"start", previousEndTime,
			"end", totalDuration)
		t.addIDTrack(tracklist, previousEndTime, totalDuration, *trackCounter)
	}
	return nil
}

func (t *TrackIDParser) addTrack(tracklist *domain.Tracklist, artist, title, startTime, endTime string, trackNumber int) {
	tracklist.Tracks = append(tracklist.Tracks, &domain.Track{
		Artist:      artist,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		TrackNumber: trackNumber,
	})
}

func (t *TrackIDParser) addIDTrack(tracklist *domain.Tracklist, startTime, endTime string, trackNumber int) {
	t.addTrack(tracklist, "ID", "ID", startTime, endTime, trackNumber)
	slog.Debug("ID track added",
		"start", startTime,
		"end", endTime)
}

func (t *TrackIDParser) calculateDuration(startTime, endTime string) int {
	start := t.parseTime(startTime)
	end := t.parseTime(endTime)
	if start.IsZero() || end.IsZero() {
		return -1
	}
	return int(end.Sub(start).Seconds())
}

func (t *TrackIDParser) parseTime(timeStr string) time.Time {
	parts := strings.Split(timeStr, ".")
	timeParts := strings.Split(parts[0], ":")
	if len(timeParts) != 3 {
		return time.Time{}
	}

	hours, _ := strconv.Atoi(timeParts[0])
	minutes, _ := strconv.Atoi(timeParts[1])
	seconds, _ := strconv.Atoi(timeParts[2])

	return time.Date(0, 1, 1, hours, minutes, seconds, 0, time.UTC)
}

func (t *TrackIDParser) calculateMidpoint(startTime, endTime string) string {
	start := t.parseTime(startTime)
	end := t.parseTime(endTime)
	duration := end.Sub(start)
	midpoint := start.Add(duration / 2)
	return fmt.Sprintf("%02d:%02d:%02d", midpoint.Hour(), midpoint.Minute(), midpoint.Second())
}

// SanitizeFilename removes or replaces unsafe characters from filenames
func SanitizeFilename(filename string) string {
	// Define a regex to match unsafe characters (anything except letters, numbers, underscore, and dot)
	re := regexp.MustCompile(`[^\w\.-]`)

	// Replace unsafe characters with an underscore
	safeName := re.ReplaceAllString(filename, "_")

	// Trim multiple underscores and ensure a clean format
	safeName = strings.Trim(safeName, "_")
	safeName = strings.ReplaceAll(safeName, "__", "_") // Avoid double underscores

	return safeName
}

// inferArtistFromTitle attempts to extract the artist name from a DJ set title
// based on common patterns like "Artist - Title", "Artist @ Venue", etc.
func inferArtistFromTitle(title string) string {
	// Common separators in DJ set titles
	separators := []struct {
		pattern string
		regex   *regexp.Regexp
	}{
		// Artist - Title
		{"-", regexp.MustCompile(`^([^-]+)\s*-\s*.+`)},
		// Artist @ Venue/Event
		{"@", regexp.MustCompile(`^([^@]+)\s*@\s*.+`)},
		// Artist | Event/Show
		{"|", regexp.MustCompile(`^([^|]+)\s*\|\s*.+`)},
		// Artist presents Title
		{"presents", regexp.MustCompile(`^([^\s]+(?:\s+[^\s]+)?)\s+presents\s+.+`)},
		// Artist live at Venue
		{"live at", regexp.MustCompile(`^([^(]+?)\s+live\s+at\s+.+`)},
	}

	for _, sep := range separators {
		matches := sep.regex.FindStringSubmatch(title)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}

	// Special format for series like "Drumcode Radio Live" -> "Drumcode Radio"
	// and "Anjunadeep Edition Episode 400" -> "Anjunadeep Edition"
	seriesPatterns := []struct {
		regex     *regexp.Regexp
		maxGroups int // How many words to consider as the artist name
	}{
		{regexp.MustCompile(`^((?:[A-Z][a-z]*\s*)+)(?:Live|Episode|Podcast|Radio Show|Mix|Set)\b`), 2},
	}

	for _, pattern := range seriesPatterns {
		matches := pattern.regex.FindStringSubmatch(title)
		if len(matches) > 1 {
			words := strings.Fields(matches[1])
			if len(words) > pattern.maxGroups {
				words = words[:pattern.maxGroups]
			}
			return strings.Join(words, " ")
		}
	}

	// If no patterns match, check if there's any whitespace - take first part
	// This is less reliable but can work for cases like "Artist Title"
	parts := strings.Fields(title)
	if len(parts) > 1 {
		// Skip articles and common non-artist words
		skipWords := map[string]bool{
			"The": true, "A": true, "An": true,
			"By": true, "With": true, "And": true,
			"From": true, "Of": true, "In": true,
			"On": true, "At": true, "To": true,
		}

		// If first word is a skip word, don't attempt this heuristic
		if len(parts) > 0 && skipWords[parts[0]] {
			return ""
		}

		// If first 2-3 words are capitalized, they likely represent an artist name
		// This is a heuristic and not always accurate
		nameCandidate := ""
		wordCount := min(3, len(parts))

		for i := 0; i < wordCount; i++ {
			// Check if word starts with uppercase (likely part of a name)
			word := parts[i]
			if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' && !skipWords[word] {
				if nameCandidate != "" {
					nameCandidate += " "
				}
				nameCandidate += word
			} else if nameCandidate != "" {
				// Stop if we've already found some name parts and hit a non-matching word
				break
			}
		}

		if nameCandidate != "" {
			return nameCandidate
		}
	}

	return "" // No pattern matched
}
