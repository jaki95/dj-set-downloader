package tracklist

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type GoogleSearchClient struct {
	apiKey     string
	searchID   string
	httpClient *http.Client
}

func NewGoogleSearchClient() (*GoogleSearchClient, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY environment variable is required")
	}

	searchID := os.Getenv("GOOGLE_SEARCH_ID")
	if searchID == "" {
		return nil, fmt.Errorf("GOOGLE_SEARCH_ID environment variable is required")
	}

	return &GoogleSearchClient{
		apiKey:     apiKey,
		searchID:   searchID,
		httpClient: &http.Client{},
	}, nil
}

type GoogleSearchResponse struct {
	Items []struct {
		Link    string `json:"link"`
		Title   string `json:"title"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

func (g *GoogleSearchClient) SearchTracklist(ctx context.Context, query string) (string, error) {
	// Add site:1001tracklists.com to the query to only search on 1001tracklists
	searchQuery := fmt.Sprintf("site:1001tracklists.com %s", query)

	// Create the search URL
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s",
		g.apiKey,
		g.searchID,
		url.QueryEscape(searchQuery),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search request failed with status: %d", resp.StatusCode)
	}

	var searchResponse GoogleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return "", fmt.Errorf("failed to decode search response: %w", err)
	}

	if len(searchResponse.Items) == 0 {
		return "", fmt.Errorf("no tracklist found for query: %s", query)
	}

	// Find the most relevant result
	// We look for results that contain "tracklist" in the title or snippet
	for _, item := range searchResponse.Items {
		lowerTitle := strings.ToLower(item.Title)
		lowerSnippet := strings.ToLower(item.Snippet)

		if strings.Contains(lowerTitle, "tracklist") || strings.Contains(lowerSnippet, "tracklist") {
			slog.Debug("Found tracklist URL", "url", item.Link, "title", item.Title)
			return item.Link, nil
		}
	}

	// If no result contains "tracklist", return the first result
	slog.Debug("Using first search result as tracklist URL", "url", searchResponse.Items[0].Link)
	return searchResponse.Items[0].Link, nil
}
