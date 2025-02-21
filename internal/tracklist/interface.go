package tracklist

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/pkg"
)

type Importer interface {
	Import(path string) (*pkg.Tracklist, error)
}

func NewImporter(config *pkg.Config) (Importer, error) {
	switch config.TracklistSource {
	case "1001tracklists":
		return NewTracklists1001Scraper(), nil
	default:
		return nil, fmt.Errorf("unknown tracklist source")
	}
}
