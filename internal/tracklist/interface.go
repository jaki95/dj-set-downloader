package tracklist

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// Importer imports a tracklist from a given source.
type Importer interface {
	Import(source string) (*domain.Tracklist, error)
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

func NewCompositeImporter() *CompositeImporter {
	return &CompositeImporter{
		importers: []Importer{
			New1001TracklistsImporter(),
			NewTrackIDParser(),
			NewCSVParser(),
		},
	}
}

func (c *CompositeImporter) Import(source string) (*domain.Tracklist, error) {
	var lastErr error
	for _, importer := range c.importers {
		tracklist, err := importer.Import(source)
		if err == nil {
			return tracklist, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("all importers failed: %w", lastErr)
}

func NewImporter(config *config.Config) (Importer, error) {
	// Always use the composite importer to try multiple sources
	return NewCompositeImporter(), nil
}
