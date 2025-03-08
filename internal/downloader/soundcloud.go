package downloader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/k0kubun/go-ansi"
	"github.com/schollz/progressbar/v3"
)

type soundCloudClient struct {
	baseURL  string
	clientID string
}

func NewSoundCloudDownloader() (*soundCloudClient, error) {
	clientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("SOUNDCLOUD_CLIENT_ID not set")
	}
	return &soundCloudClient{
		baseURL:  "https://api-v2.soundcloud.com",
		clientID: clientID,
	}, nil
}

func (s *soundCloudClient) FindURL(query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("invalid query")
	}
	slog.Debug("searching soundcloud for set", "query", query)
	encodedQuery := url.QueryEscape(query)
	res, err := http.Get(fmt.Sprintf("%s/search?q=%s&client_id=%s", s.baseURL, encodedQuery, s.clientID))
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error: %d", res.StatusCode)
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	var response interface{}

	err = json.Unmarshal(resBody, &response)
	if err != nil {
		return "", err
	}

	responseList := response.(map[string]any)["collection"].([]interface{})

	if len(responseList) == 0 {
		return "", fmt.Errorf("no results in search")
	}

	firstResult := responseList[0].(map[string]any)

	return firstResult["permalink_url"].(string), nil
}

func (s *soundCloudClient) Download(trackURL, name string, downloadPath string, progressCallback func(int, string)) error {
	slog.Debug("downloading set")
	cmd := exec.Command("scdl", "-l", trackURL, "--name-format", name, "--path", downloadPath)
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

	if err := readOutputAndReportProgress(stderr, bar, progressCallback); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("command wait error: %w", err)
	}

	return nil
}

func readOutputAndReportProgress(stderr io.ReadCloser, bar *progressbar.ProgressBar, progressCallback func(int, string)) error {
	re := regexp.MustCompile(`(\d+)%`)
	progressRe := regexp.MustCompile(`\d+%`)

	var lineBuffer bytes.Buffer
	var lastProgress int

	output := make([]byte, 1)
	for {
		_, err := stderr.Read(output)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error: %w", err)
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

	if lastProgress < 100 {
		_ = bar.Set(100)
		progressCallback(100, "Download completed")
	}

	return nil
}
