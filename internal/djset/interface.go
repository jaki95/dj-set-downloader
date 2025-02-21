package djset

import "github.com/jaki95/dj-set-downloader/pkg"

type DJSet interface {
	ProcessTracks([]*pkg.Track) error
}
