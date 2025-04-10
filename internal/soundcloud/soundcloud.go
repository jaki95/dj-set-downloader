package soundcloud

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jaki95/dj-set-downloader/internal/search"
)

type SoundCloud struct {
	googleClient *search.GoogleClient
}

func NewSoundCloud() (*SoundCloud, error) {
	googleClient, err := search.NewGoogleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google search client: %w", err)
	}

	return &SoundCloud{
		googleClient: googleClient,
	}, nil
}

func (s *SoundCloud) FindURL(ctx context.Context, query string) (string, error) {
	slog.Info("Finding SoundCloud URL", "query", query)

	// Create a more specific search query for SoundCloud
	soundcloudQuery := fmt.Sprintf("site:soundcloud.com %s", query)

	// Search using Google with site-specific search engine
	results, err := s.googleClient.Search(ctx, soundcloudQuery, "soundcloud")
	if err != nil {
		return "", fmt.Errorf("failed to search for track: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no results found for query: %s", query)
	}

	// Find the first valid SoundCloud URL
	for _, result := range results {
		if strings.Contains(result.Link, "soundcloud.com") {
			slog.Debug("Found SoundCloud URL", "url", result.Link)
			return result.Link, nil
		}
	}

	return "", fmt.Errorf("no SoundCloud results found for query: %s", query)
}
