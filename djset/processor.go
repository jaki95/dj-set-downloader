package djset

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/jaki95/dj-set-downloader/internal/downloader"
	"github.com/jaki95/dj-set-downloader/internal/tracklist"
)

type processor struct {
	tracklistImporter tracklist.Importer
	setDownloader     downloader.Downloader
	audioProcessor    audio.Processor
}

func New(cfg *config.Config) (*processor, error) {
	tracklistImporter, err := tracklist.NewImporter(cfg)
	if err != nil {
		return nil, err
	}
	setDownloader, err := downloader.NewDownloader(cfg)
	if err != nil {
		return nil, err
	}
	audioProcessor := audio.NewFFMPEGEngine()
	return &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     setDownloader,
		audioProcessor:    audioProcessor,
	}, nil
}

type ProcessingOptions struct {
	TracklistPath      string
	DJSetURL           string
	CoverArtPath       string
	FileExtension      string
	MaxConcurrentTasks int
}

func (p *processor) ProcessTracks(
	opts *ProcessingOptions,
	progressCallback func(int, string),
) ([]string, error) {
	set, err := p.tracklistImporter.Import(opts.TracklistPath)
	if err != nil {
		return nil, err
	}

	setLength := len(set.Tracks)

	url := opts.DJSetURL
	if url == "" {
		findQuery := fmt.Sprintf("%s %s", set.Name, set.Artist)
		url, err = p.setDownloader.FindURL(findQuery)
		if err != nil {
			fmt.Println("could not find match, input name of set:")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return nil, err
			}
			input = strings.TrimSpace(input)
			url, err = p.setDownloader.FindURL(input)
			if err != nil {
				return nil, err
			}
		}
		slog.Debug("found match", "url", url)
	}

	err = p.setDownloader.Download(url, set.Name, func(progress int, message string) {
		totalProgress := progress / 2 // Scale to 0-50%
		progressCallback(totalProgress, message)
	})
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%s.mp3", fmt.Sprintf("data/%s", set.Name))

	if err := p.audioProcessor.ExtractCoverArt(fileName, opts.CoverArtPath); err != nil {
		return nil, err
	}
	defer os.Remove(opts.CoverArtPath)

	err = os.MkdirAll(fmt.Sprintf("output/%s", set.Name), os.ModePerm)
	if err != nil {
		return nil, err
	}

	bar := progressbar.NewOptions(
		setLength,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][2/2][reset] Processing tracks..."),
	)

	return p.splitTracks(*set, *opts, setLength, fileName, bar, progressCallback)
}

func sanitizeTitle(title string) string {
	replacer := strings.NewReplacer("/", "-", ":", "-", "\"", "'", "?", "", "\\", "-", "|", "-")
	return replacer.Replace(title)
}

func (p *processor) splitTracks(
	set domain.Tracklist,
	opts ProcessingOptions,
	setLength int,
	fileName string,
	bar *progressbar.ProgressBar,
	progressCallback func(int, string),
) ([]string, error) {
	var wg sync.WaitGroup
	var completedTracks int32 // Atomic counter for completed tracks
	maxWorkers := opts.MaxConcurrentTasks
	if maxWorkers < 1 || maxWorkers > 10 {
		slog.Warn("invalid max workers, defaulting to 1", "maxWorkers", opts.MaxConcurrentTasks)
		maxWorkers = 1
	}
	semaphore := make(chan struct{}, maxWorkers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	filePathCh := make(chan string, len(set.Tracks))

	slog.Info("Starting track splitting", "trackCount", setLength)
	progressCallback(50, "Processing tracks...")

	for i, t := range set.Tracks {
		wg.Add(1)
		go func(i int, t *domain.Track) {
			defer wg.Done()

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
				InputPath:     fileName,
				OutputPath:    outputFile,
				FileExtension: opts.FileExtension,
				Track:         *t,
				TrackCount:    setLength,
				Artist:        set.Artist,
				Name:          set.Name,
				CoverArtPath:  opts.CoverArtPath,
			}

			if err := p.audioProcessor.Split(splitParams); err != nil {
				select {
				case errCh <- fmt.Errorf("track %d (%s): %w", i+1, t.Title, err):
					cancel()
				default:
				}
				return
			}

			// Increment completed tracks atomically
			newCount := atomic.AddInt32(&completedTracks, 1)
			trackProgress := int((float64(newCount) / float64(setLength)) * 100)
			totalProgress := 50 + (trackProgress / 2) // Scale to 50-100%
			slog.Info("Track processed", "count", newCount, "title", t.Title, "progress", totalProgress)
			bar.Add(1)
			progressCallback(totalProgress, fmt.Sprintf("Processed %d/%d tracks", newCount, setLength))
			filePathCh <- outputFile
		}(i, t)
	}

	go func() {
		wg.Wait()
		close(filePathCh)
		close(errCh)
	}()

	filePaths := make([]string, 0, len(set.Tracks))
	for path := range filePathCh {
		filePaths = append(filePaths, fmt.Sprintf("%s.%s", path, opts.FileExtension))
	}

	if err := <-errCh; err != nil {
		return nil, err
	}

	slog.Info("Splitting completed")
	progressCallback(100, "Processing completed")
	return filePaths, nil
}
