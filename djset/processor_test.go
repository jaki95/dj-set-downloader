package djset

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock dependencies
type MockTracklistImporter struct {
	mock.Mock
}

func (m *MockTracklistImporter) Import(path string) (*domain.Tracklist, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tracklist), args.Error(1)
}

type MockDownloader struct {
	mock.Mock
}

func (m *MockDownloader) FindURL(trackQuery string) (string, error) {
	args := m.Called(trackQuery)
	return args.String(0), args.Error(1)
}

func (m *MockDownloader) Download(trackURL, name string, downloadPath string, progressCallback func(int, string)) error {
	args := m.Called(trackURL, name, downloadPath, progressCallback)

	// Call the progress callback to simulate download progress
	if progressCallback != nil {
		progressCallback(50, "Testing download progress")
		progressCallback(100, "Testing download complete")
	}

	return args.Error(0)
}

type MockAudioProcessor struct {
	mock.Mock
}

func (m *MockAudioProcessor) ExtractCoverArt(inputPath, coverPath string) error {
	args := m.Called(inputPath, coverPath)
	return args.Error(0)
}

func (m *MockAudioProcessor) Split(params audio.SplitParams) error {
	args := m.Called(params)
	return args.Error(0)
}

// MockStorage implements the storage.Storage interface for testing
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) SaveDownloadedSet(setName string, originalExt string) (string, error) {
	args := m.Called(setName, originalExt)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) SaveTrack(setName, trackName string, ext string) (string, error) {
	args := m.Called(setName, trackName, ext)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetSetPath(setName string, ext string) string {
	args := m.Called(setName, ext)
	return args.String(0)
}

func (m *MockStorage) GetCoverArtPath() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockStorage) CreateSetOutputDir(setName string) error {
	args := m.Called(setName)
	return args.Error(0)
}

func (m *MockStorage) Cleanup() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockStorage) GetReader(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) GetWriter(path string) (io.WriteCloser, error) {
	args := m.Called(path)
	return args.Get(0).(io.WriteCloser), args.Error(1)
}

func (m *MockStorage) FileExists(path string) bool {
	args := m.Called(path)
	return args.Bool(0)
}

func (m *MockStorage) ListFiles(dir string, pattern string) ([]string, error) {
	args := m.Called(dir, pattern)
	return args.Get(0).([]string), args.Error(1)
}

// These functions are unused in current tests but kept for future use
// nolint:unused
func setupTestProcessor() (*processor, *MockTracklistImporter, *MockDownloader, *MockAudioProcessor, *MockStorage) {
	mockImporter := new(MockTracklistImporter)
	mockDownloader := new(MockDownloader)
	mockAudioProcessor := new(MockAudioProcessor)
	mockStorage := new(MockStorage)

	p := &processor{
		tracklistImporter: mockImporter,
		setDownloader:     mockDownloader,
		audioProcessor:    mockAudioProcessor,
		storage:           mockStorage,
	}

	return p, mockImporter, mockDownloader, mockAudioProcessor, mockStorage
}

// nolint:unused
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file %s: %v", path, err)
	}
}

func TestNew(t *testing.T) {
	// Setup environment for SoundCloud
	originalClientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
	defer os.Setenv("SOUNDCLOUD_CLIENT_ID", originalClientID) // Restore original value
	os.Setenv("SOUNDCLOUD_CLIENT_ID", "test_client_id")

	// Setup
	cfg := &config.Config{
		AudioSource:     "soundcloud",
		TracklistSource: "trackids",
		FileExtension:   "m4a",
		AudioProcessor:  "ffmpeg",
		Storage: config.StorageConfig{
			Type:      "local",
			DataDir:   "data",
			OutputDir: "output",
			TempDir:   "temp",
		},
	}

	// Test
	p, err := New(cfg)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.tracklistImporter)
	assert.NotNil(t, p.setDownloader)
	assert.NotNil(t, p.audioProcessor)
	assert.NotNil(t, p.storage)
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic sanitization",
			input:    "Track/Name: With \"Characters\"?",
			expected: "Track-Name- With 'Characters'",
		},
		{
			name:     "with backslash",
			input:    "Track\\Name",
			expected: "Track-Name",
		},
		{
			name:     "with pipe character",
			input:    "Track|Name",
			expected: "Track-Name",
		},
		{
			name:     "normal title",
			input:    "This is a normal title",
			expected: "This is a normal title",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeTitle(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
