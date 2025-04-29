package tracklistsearch

import (
	"context"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test1001TracklistsSearcher(t *testing.T) {
	// Create a test configuration
	cfg := &config.Config{}

	// Create the searcher
	searcher, err := New1001TracklistsSearcher(cfg)
	require.NoError(t, err)

	searchQuery := "charlotte de witte tomorrowland 2024"

	t.Run("Search", func(t *testing.T) {
		ctx := context.Background()

		results, err := searcher.Search(ctx, searchQuery)
		require.NoError(t, err)
		assert.NotEmpty(t, results)

		// Verify the structure of the results
		for _, result := range results {
			assert.NotEmpty(t, result.ID)
			assert.NotEmpty(t, result.Title)
			assert.NotEmpty(t, result.URL)
			assert.Equal(t, "1001tracklists", result.Source)
		}
	})

	t.Run("GetTracklist", func(t *testing.T) {
		ctx := context.Background()

		// First get a valid result ID
		results, err := searcher.Search(ctx, searchQuery)
		require.NoError(t, err)
		require.NotEmpty(t, results)

		resultID := results[0].ID
		tracklist, err := searcher.GetTracklist(ctx, resultID)
		require.NoError(t, err)
		require.NotNil(t, tracklist)
	})
}
