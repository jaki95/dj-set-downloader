package djset

import (
	"context"
	"fmt"
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
		return err
	}

	tl, err := p.tracklistImporter.Import(opts.TracklistPath)
	if err != nil {
		return err
	}

	if err := p.audioProcessor.ExtractCoverArt(opts.InputFile, opts.CoverArtPath); err != nil {
		return err
	}

	defer os.Remove(opts.CoverArtPath)

	err = os.MkdirAll(fmt.Sprintf("output/%s", opts.SetName), os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(
		len(tl.Tracks),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Processing tracks..."),
	)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.MaxConcurrentTasks)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	for i, t := range tl.Tracks {
		wg.Add(1)
		go func(i int, t *pkg.Track) {
			defer func() {
				bar.Add(1)
				wg.Done()
			}()

			select {
			case <-ctx.Done():
				return
			default:
			}

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}

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

			if splitErr := p.audioProcessor.Split(splitParams); splitErr != nil {
				select {
				case errCh <- fmt.Errorf("track %d (%s): %w", i+1, t.Title, err):
					cancel()
				default:
				}
			}
		}(i, t)

	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	if err := <-errCh; err != nil {
		return err
	}

	return nil
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}
