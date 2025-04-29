package tracklistsearch

import (
	"context"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// SearchResult represents a single result from a search operation
type SearchResult struct {
	ID          string
	Title       string
	Description string
	URL         string
	Source      string
}

// Searcher provides functionality to search for tracklists
type Searcher interface {
	// Search performs a search for tracklists matching the given query
	Search(ctx context.Context, query string) ([]SearchResult, error)

	// GetTracklist retrieves the full tracklist for a specific search result
	GetTracklist(ctx context.Context, resultID string) (*domain.Tracklist, error)
}

// NewSearcher creates a new Searcher instance based on the configuration
func NewSearcher(cfg *config.Config) (Searcher, error) {
	// For now, we'll use the 1001Tracklists implementation
	// TODO: Add support for other sources based on configuration
	return New1001TracklistsSearcher(cfg)
}
