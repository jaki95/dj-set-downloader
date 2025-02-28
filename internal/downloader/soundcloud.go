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

func (s *soundCloudClient) Download(trackURL, name string) error {
	slog.Debug("downloading set")
	cmd := exec.Command("scdl", "-l", trackURL, "--name-format", name, "--path", "data")
	// stdout, err := cmd.StdoutPipe()
	// if err != nil {
	// 	return err
	// }
	cmd.Stderr = cmd.Stdout
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
	// bar := progressbar.NewOptions(
	// 	100,
	// 	progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
	// 	progressbar.OptionEnableColorCodes(true),
	// 	progressbar.OptionSetTheme(progressbar.ThemeASCII),
	// 	progressbar.OptionFullWidth(),
	// 	progressbar.OptionShowCount(),
	// 	progressbar.OptionSetDescription("[cyan][1/2][reset] Downloading set..."),
	// )

	// if err := cmd.Start(); err != nil {
	// 	return err
	// }

	// if err := readOutputAndRenderBar(stdout, bar); err != nil {
	// 	return err
	// }

	// return cmd.Wait()
}

// readOutputAndRenderBar reads the cmd output byte-by-byte and renders a progress bar.
func readOutputAndRenderBar(stdout io.ReadCloser, bar *progressbar.ProgressBar) error {
	re := regexp.MustCompile(`(\d+)%`)
	progressRe := regexp.MustCompile(`\d+%`)

	var buf bytes.Buffer

	output := make([]byte, 1)
	var lineBuffer bytes.Buffer
	var lastProgress float64

	for {
		if lastProgress == 100 {
			fmt.Println()
			break
		}
		n, err := stdout.Read(output)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read error: %w", err)
		}

		buf.Write(output[:n])
		char := output[0]

		if char == '\r' || char == '\n' {
			// Process complete line
			line := lineBuffer.String()
			lineBuffer.Reset()

			// Filter out progress lines
			if !progressRe.MatchString(line) {
				slog.Debug(line)
			}

			// Update progress bar
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				progress, _ := strconv.ParseFloat(matches[1], 64)
				delta := int(progress - lastProgress)
				if delta > 0 {
					bar.Add(delta)
					lastProgress = progress
				}
			}
		} else {
			lineBuffer.WriteByte(char)
		}
	}

	return nil
}
