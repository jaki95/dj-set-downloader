package tracklist

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type Importer interface {
	Import(source string) (*domain.Tracklist, error)
}

const (
	Source1001Tracklists = "1001tracklists"
)

func NewImporter(config *config.Config) (Importer, error) {
	switch config.TracklistSource {
	case Source1001Tracklists:
		return New1001TracklistsImporter(), nil
	default:
		return nil, fmt.Errorf("unknown tracklist source: %s", config.TracklistSource)
	}
}
