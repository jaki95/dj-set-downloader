package tracklist

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/stretchr/testify/assert"
)

// MockScraper implements the Scraper interface for testing
type MockScraper struct {
	shouldFail bool
	tracklist  *domain.Tracklist
	err        error
	name       string
}

func (m *MockScraper) Scrape(ctx context.Context, source string) (*domain.Tracklist, error) {
	if m.shouldFail {
		return nil, m.err
	}
	return m.tracklist, nil
}

func (m *MockScraper) Name() string {
	if m.name == "" {
		return "mock"
	}
	return m.name
}

func setupTestEnv() func() {
	// Save original environment variables
	originalAPIKey := os.Getenv("GOOGLE_API_KEY")
	original1001ID := os.Getenv("GOOGLE_SEARCH_ID_1001TRACKLISTS")
	originalSCID := os.Getenv("GOOGLE_SEARCH_ID_SOUNDCLOUD")

	// Set test environment variables
	os.Setenv("GOOGLE_API_KEY", "test-api-key")
	os.Setenv("GOOGLE_SEARCH_ID_1001TRACKLISTS", "test-1001tracklists-id")
	os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", "test-soundcloud-id")

	// Return cleanup function
	return func() {
		os.Setenv("GOOGLE_API_KEY", originalAPIKey)
		os.Setenv("GOOGLE_SEARCH_ID_1001TRACKLISTS", original1001ID)
		os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", originalSCID)
	}
}

func TestNewImporter(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	tests := []struct {
		name           string
		config         *config.Config
		expectedType   string
		expectedErrMsg string
	}{
		{
			name:           "any config",
			config:         &config.Config{},
			expectedType:   "*tracklist.CompositeScraper",
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scraper, err := NewScraper(tt.config)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, scraper)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, scraper)
				assert.Equal(t, tt.expectedType, getTypeName(scraper))
			}
		})
	}
}

func TestCompositeImporter(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	cfg := &config.Config{}

	scraper, err := NewCompositeScraper(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, scraper)
	assert.Equal(t, 2, len(scraper.scrapers))
	assert.Equal(t, "*tracklist._1001TracklistsScraper", getTypeName(scraper.scrapers[0]))
	assert.Equal(t, "*tracklist.TrackIDScraper", getTypeName(scraper.scrapers[1]))
}

func TestCompositeImporterFallback(t *testing.T) {
	// Create test tracklist
	testTracklist := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test DJ",
		Tracks: []*domain.Track{
			{
				Artist:      "Artist 1",
				Title:       "Track 1",
				StartTime:   "00:00:00",
				EndTime:     "00:05:00",
				TrackNumber: 1,
			},
		},
	}

	tests := []struct {
		name           string
		scrapers       []Scraper
		expectedResult *domain.Tracklist
		expectError    bool
	}{
		{
			name: "first importer succeeds",
			scrapers: []Scraper{
				&MockScraper{tracklist: testTracklist},
				&MockScraper{shouldFail: true, err: fmt.Errorf("should not be called")},
				&MockScraper{shouldFail: true, err: fmt.Errorf("should not be called")},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "second importer succeeds",
			scrapers: []Scraper{
				&MockScraper{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockScraper{tracklist: testTracklist},
				&MockScraper{shouldFail: true, err: fmt.Errorf("should not be called")},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "third importer succeeds",
			scrapers: []Scraper{
				&MockScraper{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockScraper{shouldFail: true, err: fmt.Errorf("second failed")},
				&MockScraper{tracklist: testTracklist},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "all importers fail",
			scrapers: []Scraper{
				&MockScraper{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockScraper{shouldFail: true, err: fmt.Errorf("second failed")},
				&MockScraper{shouldFail: true, err: fmt.Errorf("third failed")},
			},
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composite := &CompositeScraper{
				scrapers: tt.scrapers,
			}

			result, err := composite.Scrape(context.Background(), "test-source")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// Helper function to get the type name as a string
func getTypeName(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%T", v)
}
