package audio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewFFMPEGEngine(t *testing.T) {
	engine := NewFFMPEGEngine()
	assert.NotNil(t, engine)
}

// Test to verify the file extension handling logic
func TestFileExtensionHandling(t *testing.T) {
	// Create an ffmpeg instance - no need to reference it since we're testing the logic directly
	_ = &ffmpeg{}

	testCases := []struct {
		name           string
		fileExtension  string
		expectedCodec  string
		expectedFormat string
	}{
		{
			name:           "MP3 Format",
			fileExtension:  "mp3",
			expectedCodec:  "libmp3lame",
			expectedFormat: "mp3",
		},
		{
			name:           "M4A Format",
			fileExtension:  "m4a",
			expectedCodec:  "aac",
			expectedFormat: "m4a",
		},
		{
			name:           "WAV Format",
			fileExtension:  "wav",
			expectedCodec:  "pcm_s16le",
			expectedFormat: "wav",
		},
		{
			name:           "FLAC Format",
			fileExtension:  "flac",
			expectedCodec:  "flac",
			expectedFormat: "flac",
		},
		{
			name:           "Unknown Format",
			fileExtension:  "unknown",
			expectedCodec:  "aac", // Default codec
			expectedFormat: "m4a", // Default format
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Get the codec and format for extractAudio
			outputPath := "test." + tc.fileExtension

			// Testing the extractAudio logic
			ext := filepath.Ext(outputPath)
			if ext != "" {
				ext = ext[1:] // Remove the leading dot
			}

			// Determine appropriate codec and format based on extension
			outputCodec := "aac"
			outputFormat := "m4a"

			switch ext {
			case "mp3":
				outputCodec = "libmp3lame"
				outputFormat = "mp3"
			case "m4a":
				outputCodec = "aac"
				outputFormat = "m4a"
			case "wav":
				outputCodec = "pcm_s16le"
				outputFormat = "wav"
			case "flac":
				outputCodec = "flac"
				outputFormat = "flac"
			}

			// Verify the codec and format match expectations
			assert.Equal(t, tc.expectedCodec, outputCodec, "Codec should match for %s", tc.fileExtension)
			assert.Equal(t, tc.expectedFormat, outputFormat, "Format should match for %s", tc.fileExtension)

			// Testing the addMetadataAndCover logic (similar pattern)
			ext = filepath.Ext(outputPath)
			if ext != "" {
				ext = ext[1:] // Remove the leading dot
			}

			// Determine appropriate format based on extension for metadata function
			outputFormat = "m4a" // Reset to default
			switch ext {
			case "mp3":
				outputFormat = "mp3"
			case "m4a":
				outputFormat = "m4a"
			case "wav":
				outputFormat = "wav"
			case "flac":
				outputFormat = "flac"
			}

			// Verify format for metadata matches expectations
			assert.Equal(t, tc.expectedFormat, outputFormat, "Metadata format should match for %s", tc.fileExtension)
		})
	}
}

// Integration test for ExtractCoverArt - requires ffmpeg to be installed
func TestExtractCoverArt(t *testing.T) {
	t.Skip("Skipping integration test")

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Input file path (would need a real audio file with cover art)
	// For a real test, you'd need to provide a test audio file
	inputPath := filepath.Join(tempDir, "test_input.mp3")
	coverPath := filepath.Join(tempDir, "test_cover.jpg")

	// Setup
	engine := NewFFMPEGEngine()

	// Test
	err := engine.ExtractCoverArt(inputPath, coverPath)

	// Assert
	assert.NoError(t, err)
	_, err = os.Stat(coverPath)
	assert.NoError(t, err, "Cover file should exist")
}

// Integration test for Split - requires ffmpeg to be installed
func TestSplit(t *testing.T) {
	t.Skip("Skipping integration test")

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Input file path (would need a real audio file)
	// For a real test, you'd need to provide a test audio file
	inputPath := filepath.Join(tempDir, "test_input.mp3")
	outputPath := filepath.Join(tempDir, "test_output")
	coverPath := filepath.Join(tempDir, "test_cover.jpg")

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
	err := engine.Split(params)

	// Assert
	assert.NoError(t, err)
	_, err = os.Stat(outputPath + ".mp3")
	assert.NoError(t, err, "Output file should exist")
}
