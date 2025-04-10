package downloader

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaki95/dj-set-downloader/internal/search"
	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type soundCloudClient struct {
	googleClient *search.GoogleClient
}

func NewSoundCloudDownloader() (*soundCloudClient, error) {
	googleClient, err := search.NewGoogleClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Google client: %w", err)
	}

	return &soundCloudClient{
		googleClient: googleClient,
	}, nil
}

func (s *soundCloudClient) FindURL(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("invalid query")
	}
	slog.Debug("searching soundcloud for set", "query", query)

	// Create a more specific search query for SoundCloud
	soundcloudQuery := fmt.Sprintf("site:soundcloud.com %s", query)

	// Search using Google with site-specific search engine
	results, err := s.googleClient.Search(ctx, soundcloudQuery, "soundcloud")
	if err != nil {
		return "", fmt.Errorf("failed to search for track: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no results found for query: %s", query)
	}

	// Find the first valid SoundCloud URL
	for _, result := range results {
		if strings.Contains(result.Link, "soundcloud.com") {
			slog.Debug("Found SoundCloud URL", "url", result.Link)
			return result.Link, nil
		}
	}

	return "", fmt.Errorf("no SoundCloud results found for query: %s", query)
}

func (s *soundCloudClient) Download(ctx context.Context, trackURL, name string, downloadPath string, progressCallback func(int, string)) error {
	slog.Debug("downloading set")
	cmd := exec.CommandContext(ctx, "scdl", "-l", trackURL, "--name-format", name, "--path", downloadPath)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	bar := progressbar.NewOptions(
		100,
		progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.ThemeASCII),
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("[cyan][1/2][reset] Downloading set..."),
	)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("command start error: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- readOutputAndReportProgress(ctx, stderr, bar, progressCallback)
	}()

	select {
	case <-ctx.Done():
		// Context was cancelled, kill the process
		if err := cmd.Cancel(); err != nil {
			slog.Error("failed to kill process", "error", err)
		}
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("command wait error: %w", err)
	}

	return nil
}

func readOutputAndReportProgress(ctx context.Context, stderr io.ReadCloser, bar *progressbar.ProgressBar, progressCallback func(int, string)) error {
	re := regexp.MustCompile(`(\d+)%`)
	progressRe := regexp.MustCompile(`\d+%`)

	var lineBuffer bytes.Buffer
	var lastProgress int

	// Create a channel to signal when reading is done
	done := make(chan struct{})
	var readErr error

	go func() {
		defer close(done)
		output := make([]byte, 1)
		for {
			_, err := stderr.Read(output)
			if err != nil {
				if err == io.EOF {
					break
				}
				readErr = fmt.Errorf("read error: %w", err)
				return
			}

			char := output[0]
			if char == '\r' || char == '\n' {
				line := lineBuffer.String()
				lineBuffer.Reset()

				if !progressRe.MatchString(line) {
					slog.Debug(line)
				}

				if matches := re.FindStringSubmatch(line); len(matches) > 1 {
					progress, _ := strconv.Atoi(matches[1])
					if progress > lastProgress {
						_ = bar.Set(progress) // Update terminal progress bar
						progressCallback(progress, "Downloading set...")
						lastProgress = progress
					}
				}
			} else {
				lineBuffer.WriteByte(char)
			}
		}
	}()

	// Wait for either context cancellation or reading completion
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		if readErr != nil {
			return readErr
		}
		if lastProgress < 100 {
			_ = bar.Set(100)
			progressCallback(100, "Download completed")
		}
		return nil
	}
}
