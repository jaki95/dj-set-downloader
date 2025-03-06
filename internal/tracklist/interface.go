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

func NewImporter(config *config.Config) (Importer, error) {
	switch config.TracklistSource {
	case Source1001Tracklists:
		return New1001TracklistsImporter(), nil
	case CSVTracklist:
		return NewCSVParser(), nil
	case TrackIDsTracklist:
		return NewTrackIDParser(), nil
	default:
		return nil, fmt.Errorf("unknown tracklist source: %s", config.TracklistSource)
	}
}
