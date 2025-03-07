package audio

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewFFMPEGEngine(t *testing.T) {
	engine := NewFFMPEGEngine()
	assert.NotNil(t, engine)
}

// Integration test for ExtractCoverArt - requires ffmpeg to be installed
func TestExtractCoverArt(t *testing.T) {
	// Skip test if ffmpeg is not available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not found in PATH, skipping test")
	}

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Input file path (would need a real audio file with cover art)
	// For a real test, you'd need to provide a test audio file
	inputPath := filepath.Join(tempDir, "test_input.mp3")
	coverPath := filepath.Join(tempDir, "test_cover.jpg")

	// This test would require a real audio file with cover art
	// Since we don't have one for the test, we'll mark it as skipped
	t.Skip("This test requires a real audio file with cover art")

	// Setup
	engine := NewFFMPEGEngine()

	// Test
	err = engine.ExtractCoverArt(inputPath, coverPath)

	// Assert
	assert.NoError(t, err)
	_, err = os.Stat(coverPath)
	assert.NoError(t, err, "Cover file should exist")
}

// Integration test for Split - requires ffmpeg to be installed
func TestSplit(t *testing.T) {
	// Skip test if ffmpeg is not available
	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not found in PATH, skipping test")
	}

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Input file path (would need a real audio file)
	// For a real test, you'd need to provide a test audio file
	inputPath := filepath.Join(tempDir, "test_input.mp3")
	outputPath := filepath.Join(tempDir, "test_output")
	coverPath := filepath.Join(tempDir, "test_cover.jpg")

	// This test would require a real audio file
	// Since we don't have one for the test, we'll mark it as skipped
	t.Skip("This test requires a real audio file")

	// Setup
	engine := NewFFMPEGEngine()

	params := SplitParams{
		InputPath:     inputPath,
		OutputPath:    outputPath,
		FileExtension: "mp3",
		Track: domain.Track{
			Title:       "Test Track",
			Artist:      "Test Artist",
			TrackNumber: 1,
			StartTime:   "00:00:00",
			EndTime:     "00:01:00",
		},
		TrackCount:   1,
		Artist:       "Test Artist",
		Name:         "Test Album",
		CoverArtPath: coverPath,
	}

	// Test
	err = engine.Split(params)

	// Assert
	assert.NoError(t, err)
	_, err = os.Stat(outputPath + ".mp3")
	assert.NoError(t, err, "Output file should exist")
}
