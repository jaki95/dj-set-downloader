package djset

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"

	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
	"github.com/jaki95/dj-set-downloader/pkg"
)

type processor struct {
	tracklistImporter tracklist.Importer
	setDownloader     downloader.Downloader
	audioProcessor    audio.Processor
}

func NewProcessor(
	tracklistImporter tracklist.Importer,
	setDownloader downloader.Downloader,
	audioProcessor audio.Processor,
) *processor {
	return &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
	}
}

type ProcessingOptions struct {
	TracklistPath      string
	DJSetURL           string
	CoverArtPath       string
	MaxConcurrentTasks int
}

func (p *processor) ProcessTracks(opts *ProcessingOptions) error {
	set, err := p.tracklistImporter.Import(opts.TracklistPath)
	if err != nil {
		return err
	}

	findQuery := fmt.Sprintf("%s %s", set.Name, set.Artist)

	url, err := p.setDownloader.FindURL(findQuery)
	if err != nil {
		return err
	}

	slog.Debug("found match", "url", url)

	err = p.setDownloader.Download(url, set.Name)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s.mp3", fmt.Sprintf("data/%s", set.Name))

	if err := p.audioProcessor.ExtractCoverArt(fileName, opts.CoverArtPath); err != nil {
		return err
	}

	defer os.Remove(opts.CoverArtPath)

	err = os.MkdirAll(fmt.Sprintf("output/%s", set.Name), os.ModePerm)
	if err != nil {
		return err
	}

	bar := progressbar.NewOptions(
		len(set.Tracks),
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Processing tracks..."),
	)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, opts.MaxConcurrentTasks)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)

	for i, t := range set.Tracks {
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
			outputFile := fmt.Sprintf("output/%s/%02d - %s", set.Name, i+1, safeTitle)

			splitParams := audio.SplitParams{
				InputPath:    fileName,
				OutputPath:   outputFile,
				Track:        *t,
				TrackCount:   len(set.Tracks),
				Artist:       set.Artist,
				Name:         set.Name,
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
