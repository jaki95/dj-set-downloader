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

type WebsiteSearcher struct {
	searchClient search.GoogleClient
	cfg          *config.Config
	scraper      tracklist.Scraper
	website      string
	source       string

	// Cache for search results
	mu      sync.RWMutex
	results []search.SearchResult
}

func NewWebsiteSearcher(cfg *config.Config, website, source string) (*WebsiteSearcher, error) {
	searchClient, err := search.NewGoogleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google search client: %v", err)
	}

	scraper, err := tracklist.NewScraper(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create tracklist scraper: %v", err)
	}

	return &WebsiteSearcher{
		searchClient: searchClient,
		cfg:          cfg,
		scraper:      scraper,
		website:      website,
		source:       source,
	}, nil
}

func (s *WebsiteSearcher) Search(ctx context.Context, query string) ([]SearchResult, error) {
	// Add site:website.com to ensure we only get results from the specified website
	siteQuery := query + " site:" + s.website

	results, err := s.searchClient.Search(ctx, siteQuery, s.source)
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
			ID:          fmt.Sprintf("%s_%d", s.source, i),
			Title:       result.Title,
			Description: "", // Google search results don't include descriptions
			URL:         result.Link,
			Source:      s.source,
		})
	}

	return searchResults, nil
}

func (s *WebsiteSearcher) GetTracklist(ctx context.Context, resultID string) (*domain.Tracklist, error) {
	// The resultID is in the format "source_X" where X is a numeric index
	// We extract X to look up the corresponding search result
	var index int
	_, err := fmt.Sscanf(resultID, s.source+"_%d", &index)
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
