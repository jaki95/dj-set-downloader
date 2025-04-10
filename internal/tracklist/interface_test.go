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

// MockImporter implements the Importer interface for testing
type MockImporter struct {
	shouldFail bool
	tracklist  *domain.Tracklist
	err        error
}

func (m *MockImporter) Import(ctx context.Context, source string) (*domain.Tracklist, error) {
	if m.shouldFail {
		return nil, m.err
	}
	return m.tracklist, nil
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
			name: "any config",
			config: &config.Config{
				TracklistSource: Source1001Tracklists,
			},
			expectedType:   "*tracklist.CompositeImporter",
			expectedErrMsg: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer, err := NewImporter(tt.config)

			if tt.expectedErrMsg != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, importer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, importer)
				assert.Equal(t, tt.expectedType, getTypeName(importer))
			}
		})
	}
}

func TestCompositeImporter(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	importer, err := NewCompositeImporter()
	assert.NoError(t, err)
	assert.NotNil(t, importer)
	assert.Equal(t, 3, len(importer.importers))
	assert.Equal(t, "*tracklist.tracklists1001Importer", getTypeName(importer.importers[0]))
	assert.Equal(t, "*tracklist.TrackIDParser", getTypeName(importer.importers[1]))
	assert.Equal(t, "*tracklist.CSVParser", getTypeName(importer.importers[2]))
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
		importers      []Importer
		expectedResult *domain.Tracklist
		expectError    bool
	}{
		{
			name: "first importer succeeds",
			importers: []Importer{
				&MockImporter{tracklist: testTracklist},
				&MockImporter{shouldFail: true, err: fmt.Errorf("should not be called")},
				&MockImporter{shouldFail: true, err: fmt.Errorf("should not be called")},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "second importer succeeds",
			importers: []Importer{
				&MockImporter{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockImporter{tracklist: testTracklist},
				&MockImporter{shouldFail: true, err: fmt.Errorf("should not be called")},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "third importer succeeds",
			importers: []Importer{
				&MockImporter{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockImporter{shouldFail: true, err: fmt.Errorf("second failed")},
				&MockImporter{tracklist: testTracklist},
			},
			expectedResult: testTracklist,
			expectError:    false,
		},
		{
			name: "all importers fail",
			importers: []Importer{
				&MockImporter{shouldFail: true, err: fmt.Errorf("first failed")},
				&MockImporter{shouldFail: true, err: fmt.Errorf("second failed")},
				&MockImporter{shouldFail: true, err: fmt.Errorf("third failed")},
			},
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			composite := &CompositeImporter{
				importers: tt.importers,
			}

			result, err := composite.Import(context.Background(), "test-source")

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
