package tracklist

import (
	"context"
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// Importer imports a tracklist from a given source.
type Importer interface {
	Import(ctx context.Context, source string) (*domain.Tracklist, error)
}

const (
	Source1001Tracklists = "1001tracklists"
	CSVTracklist         = "csv"
	TrackIDsTracklist    = "trackids"
)

// CompositeImporter tries multiple importers in sequence until one succeeds
type CompositeImporter struct {
	importers []Importer
}

func NewCompositeImporter() (*CompositeImporter, error) {
	importer, err := New1001TracklistsImporter()
	if err != nil {
		return nil, fmt.Errorf("failed to create 1001tracklists importer: %w", err)
	}

	return &CompositeImporter{
		importers: []Importer{
			importer,
			NewTrackIDParser(),
			NewCSVParser(),
		},
	}, nil
}

func (c *CompositeImporter) Import(ctx context.Context, source string) (*domain.Tracklist, error) {
	var lastErr error
	for _, importer := range c.importers {
		tracklist, err := importer.Import(ctx, source)
		if err == nil {
			return tracklist, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all importers failed: %w", lastErr)
}

func NewImporter(config *config.Config) (Importer, error) {
	// Always use the composite importer to try multiple sources
	return NewCompositeImporter()
}
