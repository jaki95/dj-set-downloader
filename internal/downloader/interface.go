package downloader

import (
	"fmt"

	"github.com/jaki95/dj-set-downloader/config"
)

// Downloader handles the downloading of DJ sets.
type Downloader interface {
	FindURL(trackQuery string) (string, error)
	Download(trackURL, name string, downloadPath string, progressCallback func(int, string)) error
}

const (
	SoundCloud = "soundcloud"
)

func NewDownloader(config *config.Config) (Downloader, error) {
	switch config.AudioSource {
	case SoundCloud:
		return NewSoundCloudDownloader()
	default:
		return nil, fmt.Errorf("unknown downloader source: %s", config.AudioSource)
	}
}
