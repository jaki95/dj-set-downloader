package downloader

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

type Downloader interface {
	FindURL(trackQuery string) (string, error)
	Download(trackURL, name string) error
}

const (
	SoundCloud = "soundcloud"
)

func NewDownloader(config *domain.Config) (Downloader, error) {
	switch config.AudioSource {
	case SoundCloud:
		return NewSoundCloudDownloader()
	default:
		return nil, fmt.Errorf("unknown downloader source: %s", config.AudioSource)
	}
}
