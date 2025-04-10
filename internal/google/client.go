package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

type SearchResult struct {
	Title string
	Link  string
}

type GoogleClient struct {
	apiKey        string
	searchEngines map[string]string // map of site name to search engine ID
	httpClient    *http.Client
}

func NewGoogleClient() (*GoogleClient, error) {
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

	return &GoogleClient{
		apiKey:        apiKey,
		searchEngines: searchEngines,
		httpClient:    &http.Client{},
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

	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Add("key", c.apiKey)
	params.Add("cx", searchID)
	params.Add("q", query)

	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			Title string `json:"title"`
			Link  string `json:"link"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	results := make([]SearchResult, len(result.Items))
	for i, item := range result.Items {
		results[i] = SearchResult{
			Title: item.Title,
			Link:  item.Link,
		}
	}

	return results, nil
}
