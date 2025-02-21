package djset

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/schollz/progressbar/v3"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/jaki95/dj-set-downloader/pkg"
)

type processor struct {
	tracklistImporter tracklist.Importer
	audioProcessor    audio.Processor
}

func NewProcessor(
	tracklistImporter tracklist.Importer,
	audioProcessor audio.Processor,
) *processor {
	return &processor{
		tracklistImporter: tracklistImporter,
		audioProcessor:    audioProcessor,
	}
}

type ProcessingOptions struct {
	TracklistPath      string
	DJSetURL           string
	InputFile          string
	SetArtist          string
	SetName            string
	CoverArtPath       string
	MaxConcurrentTasks int
}

func (p *processor) ProcessTracks(opts *ProcessingOptions) error {
	err := downloader.Download(opts.SetName, opts.DJSetURL)
	if err != nil {
		log.Fatal(err)
	}

	tl, err := p.tracklistImporter.Import(opts.TracklistPath)
	if err != nil {
		return err
	}

	if err := p.audioProcessor.ExtractCoverArt(opts.InputFile, opts.CoverArtPath); err != nil {
		return err
	}

	defer os.Remove(opts.CoverArtPath)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.MaxConcurrentTasks)

	err = os.MkdirAll(fmt.Sprintf("output/%s", opts.SetName), os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(
		len(tl.Tracks),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Processing tracks..."),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
	)

	for i, t := range tl.Tracks {
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
				TrackCount:   len(tl.Tracks),
				Artist:       opts.SetArtist,
				Name:         opts.SetName,
				CoverArtPath: opts.CoverArtPath,
			}

			p.audioProcessor.Split(splitParams)
		}(i, t)

	}

	wg.Wait()

	return nil

}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}
