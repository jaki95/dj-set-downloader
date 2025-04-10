package soundcloud

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/google"
)

type SoundCloudSearcher struct {
	googleClient *google.GoogleClient
	clientID     string
}

func NewSoundCloudSearcher(googleClient *google.GoogleClient, clientID string) *SoundCloudSearcher {
	return &SoundCloudSearcher{
		googleClient: googleClient,
		clientID:     clientID,
	}
}

func (s *SoundCloudSearcher) Search(ctx context.Context, query string) (*domain.SoundCloudTrack, error) {
	slog.Info("Searching for track", "query", query)

	// Create a more specific search query for SoundCloud
	soundcloudQuery := fmt.Sprintf("site:soundcloud.com %s", query)

	// Search using Google with site-specific search engine
	results, err := s.googleClient.Search(ctx, soundcloudQuery, "soundcloud")
	if err != nil {
		return nil, fmt.Errorf("failed to search for track: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for query: %s", query)
	}

	// Find the first valid SoundCloud URL
	var trackURL string
	for _, result := range results {
		if strings.Contains(result.Link, "soundcloud.com") {
			trackURL = result.Link
			slog.Debug("Found SoundCloud URL", "url", trackURL)
			break
		}
	}

	if trackURL == "" {
		return nil, fmt.Errorf("no SoundCloud results found for query: %s", query)
	}

	// Extract track ID from URL
	trackID, err := extractTrackID(trackURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract track ID: %w", err)
	}

	// Extract title and artist from the search result
	title, artist := extractTrackInfo(results[0].Title)

	// Create and return the track
	track := &domain.SoundCloudTrack{
		ID:     trackID,
		Title:  title,
		Artist: artist,
		URL:    trackURL,
	}

	slog.Info("Found track", "id", trackID, "title", title, "artist", artist)
	return track, nil
}

func extractTrackID(url string) (string, error) {
	// Match patterns like /tracks/123456789 or /sets/123456789
	re := regexp.MustCompile(`soundcloud\.com/(?:tracks|sets)/([^/]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) < 2 {
		return "", fmt.Errorf("invalid SoundCloud URL format: %s", url)
	}
	return matches[1], nil
}

func extractTrackInfo(title string) (string, string) {
	// Try to split on common separators
	separators := []string{" - ", " – ", " — ", " by ", " · "}
	for _, sep := range separators {
		parts := strings.SplitN(title, sep, 2)
		if len(parts) == 2 {
			// Clean up the parts
			artist := strings.TrimSpace(parts[0])
			trackTitle := strings.TrimSpace(parts[1])

			// Remove common suffixes
			trackTitle = strings.TrimSuffix(trackTitle, " | Free Download")
			trackTitle = strings.TrimSuffix(trackTitle, " | Free Stream")
			trackTitle = strings.TrimSuffix(trackTitle, " | Stream")
			trackTitle = strings.TrimSuffix(trackTitle, " | Download")

			return trackTitle, artist
		}
	}

	// If no separator found, return the whole title as the track title
	return title, ""
}

func (s *SoundCloudSearcher) GetTrack(ctx context.Context, trackID string) (*domain.SoundCloudTrack, error) {
	// TODO: Implement track details fetching using SoundCloud API
	// For now, return a basic track object
	return &domain.SoundCloudTrack{
		ID:     trackID,
		Title:  "Track Title",
		Artist: "Track Artist",
		URL:    fmt.Sprintf("https://soundcloud.com/tracks/%s", trackID),
	}, nil
}
