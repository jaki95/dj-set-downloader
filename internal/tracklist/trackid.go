package tracklist

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type TrackIDScraper struct {
}

type trackIDResponse struct {
	Result struct {
		Title     string `json:"title"`
		Duration  string `json:"duration"`
		Processes []struct {
			Tracks []domain.Track `json:"detectionProcessMusicTracks"`
		} `json:"detectionProcesses"`
	} `json:"result"`
}

func NewTrackIDScraper() *TrackIDScraper {
	return &TrackIDScraper{}
}

func (t *TrackIDScraper) Name() string {
	return "trackid"
}

func (t *TrackIDScraper) Scrape(ctx context.Context, tracklistUrl string) (*domain.Tracklist, error) {
	slog.Info("Importing tracklist", "url", tracklistUrl)

	// Check if the URL is valid
	u, err := url.Parse(tracklistUrl)
	if err != nil || u.Scheme == "" || u.Host == "" {
		// Not a URL, error
		return nil, fmt.Errorf("invalid url: %s", tracklistUrl)
	}

	resp, err := t.fetchTrackData(ctx, tracklistUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch track data: %w", err)
	}

	tracklist, err := t.parseTracklist(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracklist: %w", err)
	}

	if len(tracklist.Tracks) == 0 {
		return nil, fmt.Errorf("no tracks found in TrackID response")
	}

	return tracklist, nil
}

func (t *TrackIDScraper) fetchTrackData(ctx context.Context, tracklistUrl string) (*trackIDResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tracklistUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create track data request: %w", err)
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
		return nil, fmt.Errorf("failed to read track data response body: %w", err)
	}

	var trackResp trackIDResponse
	if err := json.Unmarshal(body, &trackResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal track data JSON: %w", err)
	}

	return &trackResp, nil
}

func (t *TrackIDScraper) setCommonHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dj-set-downloader/1.0")
}

func (t *TrackIDScraper) parseTracklist(resp *trackIDResponse) (*domain.Tracklist, error) {
	title := resp.Result.Title
	artist := inferArtistFromTitle(title)

	tracklist := &domain.Tracklist{
		Name:   title,
		Artist: artist,
	}
	totalDuration := resp.Result.Duration
	trackCounter := 1
	previousEndTime := ""

	var allTracks []domain.Track
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

func (t *TrackIDScraper) processFirstTrack(tracklist *domain.Tracklist, firstTrack domain.Track, trackCounter *int, previousEndTime *string) error {
	if firstTrack.StartTime != "00:00:00" {
		t.addIDTrack(tracklist, "00:00:00", firstTrack.StartTime, *trackCounter)
		*trackCounter++
	}

	t.addTrack(tracklist, firstTrack.Artist, firstTrack.Title, firstTrack.StartTime, firstTrack.EndTime, *trackCounter)
	*trackCounter++
	*previousEndTime = firstTrack.EndTime
	return nil
}

func (t *TrackIDScraper) handleTrackGap(tracklist *domain.Tracklist, previousEndTime, startTime string, trackCounter *int) {
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

func (t *TrackIDScraper) handleFinalGap(tracklist *domain.Tracklist, previousEndTime, totalDuration string, trackCounter *int) error {
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

func (t *TrackIDScraper) addTrack(tracklist *domain.Tracklist, artist, title, startTime, endTime string, trackNumber int) {
	tracklist.Tracks = append(tracklist.Tracks, &domain.Track{
		Artist:      artist,
		Title:       title,
		StartTime:   startTime,
		EndTime:     endTime,
		TrackNumber: trackNumber,
	})
}

func (t *TrackIDScraper) addIDTrack(tracklist *domain.Tracklist, startTime, endTime string, trackNumber int) {
	t.addTrack(tracklist, "ID", "ID", startTime, endTime, trackNumber)
	slog.Debug("ID track added",
		"start", startTime,
		"end", endTime)
}

func (t *TrackIDScraper) calculateDuration(startTime, endTime string) int {
	start := t.parseTime(startTime)
	end := t.parseTime(endTime)
	if start.IsZero() || end.IsZero() {
		return -1
	}
	return int(end.Sub(start).Seconds())
}

func (t *TrackIDScraper) parseTime(timeStr string) time.Time {
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

func (t *TrackIDScraper) calculateMidpoint(startTime, endTime string) string {
	start := t.parseTime(startTime)
	end := t.parseTime(endTime)
	duration := end.Sub(start)
	midpoint := start.Add(duration / 2)
	return fmt.Sprintf("%02d:%02d:%02d", midpoint.Hour(), midpoint.Minute(), midpoint.Second())
}

// inferArtistFromTitle attempts to extract the artist name from a DJ set title
// based on common patterns like "Artist - Title", "Artist @ Venue", etc.
func inferArtistFromTitle(title string) string {
	if title == "" {
		return ""
	}

	// Don't extract artist from titles starting with "The"
	if strings.HasPrefix(title, "The ") {
		return ""
	}

	words := strings.Fields(title)
	if len(words) <= 1 {
		return ""
	}

	// Common separators that indicate end of artist name
	separators := []string{
		" - ",
		" | ",
		" @ ",
		" live at ",
		" presents ",
		" b2b ",
	}

	// Check for radio show patterns first
	if strings.Contains(title, "Episode") || strings.Contains(title, "Live") {
		// Extract show name without episode number
		parts := strings.Split(title, "Episode")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[0])
		}
		parts = strings.Split(title, "Live")
		if len(parts) > 1 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Try to find the first separator
	for _, sep := range separators {
		if idx := strings.Index(strings.ToLower(title), strings.ToLower(sep)); idx > 0 {
			return strings.TrimSpace(title[:idx])
		}
	}

	return ""
}
