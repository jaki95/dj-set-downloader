package tracklistsearch

import (
	"context"
	"fmt"
	"sync"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/search"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

type _1001TracklistsSearcher struct {
	searchClient search.GoogleClient
	cfg          *config.Config
	scraper      tracklist.Scraper

	// Cache for search results
	mu      sync.RWMutex
	results []search.SearchResult
}

func New1001TracklistsSearcher(cfg *config.Config) (*_1001TracklistsSearcher, error) {
	searchClient, err := search.NewGoogleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google search client: %v", err)
	}

	scraper, err := tracklist.New1001TracklistsScraper(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracklist scraper: %v", err)
	}

	return &_1001TracklistsSearcher{
		searchClient: searchClient,
		cfg:          cfg,
		scraper:      scraper,
	}, nil
}

func (s *_1001TracklistsSearcher) Search(ctx context.Context, query string) ([]SearchResult, error) {
	// Add site:1001tracklists.com to ensure we only get results from 1001Tracklists
	siteQuery := query + " site:1001tracklists.com"

	results, err := s.searchClient.Search(ctx, siteQuery, "1001tracklists")
	if err != nil {
		return nil, fmt.Errorf("failed to search: %v", err)
	}

	// Store the results for later use
	s.mu.Lock()
	s.results = results
	s.mu.Unlock()

	var searchResults []SearchResult
	for i, result := range results {
		searchResults = append(searchResults, SearchResult{
			ID:          fmt.Sprintf("1001_%d", i),
			Title:       result.Title,
			Description: "", // Google search results don't include descriptions
			URL:         result.Link,
			Source:      "1001tracklists",
		})
	}

	return searchResults, nil
}

func (s *_1001TracklistsSearcher) GetTracklist(ctx context.Context, resultID string) (*domain.Tracklist, error) {
	// Extract the index from the resultID
	var index int
	_, err := fmt.Sscanf(resultID, "1001_%d", &index)
	if err != nil {
		return nil, fmt.Errorf("invalid result ID: %v", err)
	}

	// Get the URL from the cached results
	s.mu.RLock()
	results := s.results
	s.mu.RUnlock()

	if index >= len(results) {
		return nil, fmt.Errorf("result not found")
	}

	// Use the existing scraper to get the tracklist
	return s.scraper.Scrape(ctx, results[index].Link)
}
