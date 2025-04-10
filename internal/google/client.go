package google

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type SearchResult struct {
	Title string
	Link  string
}

type GoogleClient struct {
	service       *customsearch.Service
	searchEngines map[string]string
}

func NewClient() (*GoogleClient, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	searchEngines := map[string]string{
		"1001tracklists": os.Getenv("GOOGLE_SEARCH_ID_1001TRACKLISTS"),
		"soundcloud":     os.Getenv("GOOGLE_SEARCH_ID_SOUNDCLOUD"),
	}

	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is required")
	}

	// Validate that we have at least one search engine configured
	hasValidEngine := false
	for _, id := range searchEngines {
		if id != "" {
			hasValidEngine = true
			break
		}
	}
	if !hasValidEngine {
		return nil, fmt.Errorf("at least one search engine ID must be configured (GOOGLE_SEARCH_ID_1001TRACKLISTS or GOOGLE_SEARCH_ID_SOUNDCLOUD)")
	}

	ctx := context.Background()
	service, err := customsearch.NewService(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create custom search service: %w", err)
	}

	return &GoogleClient{
		service:       service,
		searchEngines: searchEngines,
	}, nil
}

func (c *GoogleClient) Search(ctx context.Context, query string, site string) ([]SearchResult, error) {
	searchID, ok := c.searchEngines[site]
	if !ok {
		return nil, fmt.Errorf("no search engine configured for site: %s", site)
	}
	if searchID == "" {
		return nil, fmt.Errorf("search engine ID not configured for site: %s", site)
	}

	call := c.service.Cse.List().Cx(searchID).Q(query)
	resp, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}

	results := make([]SearchResult, len(resp.Items))
	for i, item := range resp.Items {
		results[i] = SearchResult{
			Title: item.Title,
			Link:  item.Link,
		}
	}

	return results, nil
}
