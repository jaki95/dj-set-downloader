package tracklist

import "github.com/jaki95/dj-set-downloader/pkg"

type Tracklist interface {
	ProcessTracks([]*pkg.Track) error
}
