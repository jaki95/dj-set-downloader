package downloader

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"

	"github.com/schollz/progressbar/v3"
)

func Download(setName, url string) error {
	fmt.Println("Downloading set...")

	cmd := exec.Command("scdl", "-l", url, "--name-format", setName, "--path", "data")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	bar := progressbar.NewOptions(100,
		progressbar.OptionFullWidth(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Downloading set..."),
	)

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := readOutputAndRenderBar(stdout, bar); err != nil {
		return err
	}

	return cmd.Wait()
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
				fmt.Println(line)
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
