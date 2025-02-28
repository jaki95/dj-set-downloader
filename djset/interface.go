package djset

import "github.com/jaki95/dj-set-downloader/internal/domain"

type DJSet interface {
	ProcessTracks([]*domain.Track) error
}
