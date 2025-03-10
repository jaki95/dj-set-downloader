package audio

import (
	"context"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// Processor processes an audio track.
type Processor interface {
	ExtractCoverArt(ctx context.Context, inputPath, coverPath string) error
	Split(ctx context.Context, sp SplitParams) error
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
