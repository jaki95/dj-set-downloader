package tracklist

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"

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

	err := os.MkdirAll(fmt.Sprintf("output/%s", opts.SetName), os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(
		len(tracks),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Processing tracks..."),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
	)

	for i, t := range tracks {
		wg.Add(1)
		go func(i int, t *pkg.Track) {
			defer bar.Add(1)
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			safeTitle := sanitizeTitle(t.Title)
			outputFile := fmt.Sprintf("output/%s/%02d - %s", opts.SetName, i+1, safeTitle)

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
