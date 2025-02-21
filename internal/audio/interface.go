package audio

import "github.com/jaki95/dj-set-downloader/pkg"

type Processor interface {
	ExtractCoverArt(inputPath, coverPath string) error
	Split(sp SplitParams) error
}

type SplitParams struct {
	InputPath    string
	OutputPath   string
	Track        pkg.Track
	TrackCount   int
	Artist       string
	Name         string
	CoverArtPath string
}
