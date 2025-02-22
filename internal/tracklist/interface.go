package tracklist

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/pkg"
)

type Importer interface {
	Import(source string) (*pkg.Tracklist, error)
}

const (
	Source1001Tracklists = "1001tracklists"
)

func NewImporter(config *pkg.Config) (Importer, error) {
	switch config.TracklistSource {
	case Source1001Tracklists:
		return New1001TracklistsImporter(), nil
	default:
		return nil, fmt.Errorf("unknown tracklist source: %s", config.TracklistSource)
	}
}
