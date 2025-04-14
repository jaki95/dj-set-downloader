package search

import "context"

// MockGoogleClient is a mock implementation of GoogleClient for testing
type MockGoogleClient struct {
	SearchFunc func(ctx context.Context, query string, site string) ([]SearchResult, error)
}

// Search implements the GoogleClient interface
func (m *MockGoogleClient) Search(ctx context.Context, query string, site string) ([]SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, site)
	}
	return nil, nil
}
