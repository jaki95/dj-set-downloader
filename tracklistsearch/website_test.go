package tracklistsearch

import (
	"context"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebsiteSearcher(t *testing.T) {
	// Create a test configuration
	configs := []config.Config{
		{
			TracklistSource:  "1001tracklists",
			TracklistWebsite: "1001tracklists.com",
		},
		{
			TracklistSource:  "trackid",
			TracklistWebsite: "trackid.net",
		},
	}

	for _, cfg := range configs {
		t.Run(cfg.TracklistSource, func(t *testing.T) {
			// Create the searcher
			searcher, err := NewSearcher(&cfg)
			require.NoError(t, err)

			t.Run("Search", func(t *testing.T) {
				ctx := context.Background()
				query := "charlotte de witte tomorrowland 2024"

				results, err := searcher.Search(ctx, query)
				require.NoError(t, err)
				assert.NotEmpty(t, results)

				// Verify the structure of the results
				for _, result := range results {
					assert.NotEmpty(t, result.ID)
					assert.NotEmpty(t, result.Title)
					assert.NotEmpty(t, result.URL)
					assert.Equal(t, cfg.TracklistSource, result.Source)
					assert.Contains(t, result.URL, cfg.TracklistWebsite)
					// Verify ID format
					assert.Regexp(t, cfg.TracklistSource+"_\\d+", result.ID)
				}
			})

			t.Run("GetTracklist", func(t *testing.T) {
				ctx := context.Background()
				query := "charlotte de witte tomorrowland 2024"

				// First get a valid result ID
				results, err := searcher.Search(ctx, query)
				require.NoError(t, err)
				require.NotEmpty(t, results)

				resultID := results[0].ID
				tracklist, err := searcher.GetTracklist(ctx, resultID)
				require.NoError(t, err)
				require.NotNil(t, tracklist)

				// Verify tracklist structure
				assert.NotEmpty(t, tracklist.Tracks)
				for _, track := range tracklist.Tracks {
					assert.NotEmpty(t, track.Title)
					assert.NotEmpty(t, track.Artist)
				}
			})

			t.Run("GetTracklist with invalid ID", func(t *testing.T) {
				ctx := context.Background()
				_, err := searcher.GetTracklist(ctx, "invalid_id")
				assert.Error(t, err)
			})

			t.Run("GetTracklist with non-existent index", func(t *testing.T) {
				ctx := context.Background()
				_, err := searcher.GetTracklist(ctx, cfg.TracklistSource+"_999999")
				assert.Error(t, err)
			})
		})
	}
}
