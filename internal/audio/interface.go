package audio

import "github.com/jaki95/dj-set-downloader/internal/domain"

// Processor processes an audio track.
type Processor interface {
	ExtractCoverArt(inputPath, coverPath string) error
	Split(sp SplitParams) error
}

type SplitParams struct {
	InputPath     string
	OutputPath    string
	FileExtension string
	Track         domain.Track
	TrackCount    int
	Artist        string
	Name          string
	CoverArtPath  string
}
