package tracklist

import (
	"fmt"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/stretchr/testify/assert"
)

func TestNewImporter(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.Config
		expectedType   string
		expectedErrMsg string
	}{
		{
			name: "1001tracklists source",
			config: &config.Config{
				TracklistSource: Source1001Tracklists,
			},
			expectedType:   "*tracklist.tracklists1001Importer",
			expectedErrMsg: "",
		},
		{
			name: "csv source",
			config: &config.Config{
				TracklistSource: CSVTracklist,
			},
			expectedType:   "*tracklist.CSVParser",
			expectedErrMsg: "",
		},
		{
			name: "trackids source",
			config: &config.Config{
				TracklistSource: TrackIDsTracklist,
			},
			expectedType:   "*tracklist.TrackIDParser",
			expectedErrMsg: "",
		},
		{
			name: "unknown source",
			config: &config.Config{
				TracklistSource: "unknown",
			},
			expectedType:   "",
			expectedErrMsg: "unknown tracklist source: unknown",
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

// Helper function to get the type name as a string
func getTypeName(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%T", v)
}

// This function returns the type of the value
func typeOf(v interface{}) interface{} {
	return v
}
