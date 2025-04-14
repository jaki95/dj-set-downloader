package tracklist

import (
	"context"
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/search"
)

// Importer imports a tracklist from a given source.
type Importer interface {
	Import(ctx context.Context, source string) (*domain.Tracklist, error)
	Name() string
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

func (c *CompositeImporter) Name() string {
	return "composite"
}

func NewCompositeImporter() (*CompositeImporter, error) {
	// Create Google search client
	searchClient, err := search.NewGoogleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google search client: %v", err)
	}

	// Create 1001Tracklists importer with the search client
	importer, err := New1001TracklistsImporter(searchClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create 1001tracklists importer: %v", err)
	}

	return &CompositeImporter{
		importers: []Importer{
			importer,
			NewTrackIDImporter(),
			NewCSVImporter(),
		},
	}, nil
}

func (c *CompositeImporter) Import(ctx context.Context, source string) (*domain.Tracklist, error) {
	var errors []error
	for _, importer := range c.importers {
		tracklist, err := importer.Import(ctx, source)
		if err == nil {
			return tracklist, nil
		}
		errors = append(errors, fmt.Errorf("%s: %v", importer.Name(), err))
	}
	return nil, fmt.Errorf("all importers failed: %v", errors)
}

func NewImporter(config *config.Config) (Importer, error) {
	// Always use the composite importer to try multiple sources
	return NewCompositeImporter()
}
