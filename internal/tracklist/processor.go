package tracklist

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/pkg"
)

type processor struct {
	audioSplitter audio.Processor
}

func NewProcessor(splitter audio.Processor) *processor {
	return &processor{
		audioSplitter: splitter,
	}
}

type ProcessOptions struct {
	InputFile          string
	SetArtist          string
	SetName            string
	CoverArtPath       string
	MaxConcurrentTasks int
}

func (p *processor) ProcessTracks(tracks []*pkg.Track, opts *ProcessOptions) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.MaxConcurrentTasks)

	for i, t := range tracks {
		wg.Add(1)
		go func(i int, t *pkg.Track) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			safeTitle := sanitizeTitle(t.Title)
			outputFile := fmt.Sprintf("%02d - %s", i+1, safeTitle)

			splitParams := audio.SplitParams{
				InputPath:    opts.InputFile,
				OutputPath:   outputFile,
				Track:        *t,
				TrackCount:   len(tracks),
				Artist:       opts.SetArtist,
				Name:         opts.SetName,
				CoverArtPath: opts.CoverArtPath,
			}

			p.audioSplitter.Split(splitParams)
		}(i, t)

	}

	wg.Wait()

	return nil

}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}
