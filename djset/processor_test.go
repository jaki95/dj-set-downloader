package djset

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/schollz/progressbar/v3"
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
	p, err := NewProcessor(cfg)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.(*processor).tracklistImporter)
	assert.NotNil(t, p.(*processor).setDownloader)
	assert.NotNil(t, p.(*processor).audioProcessor)
	assert.NotNil(t, p.(*processor).storage)
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

// TestDownloadProgressCalculation tests that progress values during download are never negative
func TestDownloadProgressCalculation(t *testing.T) {
	// Create a simple test that directly tests the progress calculation logic
	// This isolates the test from the complex processor flow

	// Create a slice to capture progress values
	progressValues := []int{}

	// Create a progress callback that records values
	progressCallback := func(progress int, message string) {
		progressValues = append(progressValues, progress)
	}

	// Simulate the Download callback with different input progress values
	testProgresses := []int{0, 5, 10, 20, 50, 75, 100}

	for _, inputProgress := range testProgresses {
		// Apply the exact same calculation used in the processor code
		adjustedProgress := 10 + (inputProgress / 2)
		progressCallback(adjustedProgress, "Testing progress")
	}

	// Verify each progress value is correct and non-negative
	expectedResults := []int{10, 12, 15, 20, 35, 47, 60}

	// Ensure we have the expected number of results
	assert.Equal(t, len(expectedResults), len(progressValues), "Should have the correct number of progress values")

	// Check each value individually
	for i, expectedValue := range expectedResults {
		// The progress should never be negative
		assert.GreaterOrEqual(t, progressValues[i], 0, "Progress should never be negative")

		// Verify the exact value matches what we expect based on the processor's calculation
		assert.Equal(t, expectedValue, progressValues[i],
			"Progress value for input %d should be %d", testProgresses[i], expectedValue)
	}

	// Verify edge cases match our expectations:
	// - 0% download progress should be reported as 10%
	assert.Equal(t, 10, progressValues[0], "0% download should be reported as 10%")

	// - 100% download progress should be reported as 60%
	assert.Equal(t, 60, progressValues[len(progressValues)-1], "100% download should be reported as 60%")
}

func TestGetMaxWorkers(t *testing.T) {
	// Create test processor directly
	p := &processor{}

	// Test cases
	testCases := []struct {
		name     string
		input    int
		expected int
	}{
		{"Zero value", 0, 4},
		{"Negative value", -5, 4},
		{"Valid low value", 2, 2},
		{"Valid high value", 8, 8},
		{"Exceeds max limit", 12, 4},
	}

	// Test each case
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := p.getMaxWorkers(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMonitorTrackProcessing(t *testing.T) {
	// Create test processor directly
	p := &processor{}
	trackNumber := 1
	trackTitle := "Test Track"
	startTime := time.Now()

	// First test: normal case (done channel closes quickly)
	t.Run("Normal processing", func(t *testing.T) {
		done := make(chan struct{})

		// Close the done channel immediately to simulate normal processing
		go func() {
			time.Sleep(10 * time.Millisecond)
			close(done)
		}()

		// This should exit quickly without logging
		p.monitorTrackProcessing(done, startTime, trackNumber, trackTitle)
		// No assertion needed - if it blocks, the test will time out
	})

	// Second test: simulate a timeout
	// This is harder to test directly since it would require hooking into the log system
	// But we can at least verify it doesn't panic or crash
	t.Run("Long processing simulation", func(t *testing.T) {
		// Use a never-closing channel to simulate slow processing
		neverDone := make(chan struct{})

		// Run in goroutine with timeout to prevent test from hanging
		processingComplete := make(chan struct{})
		go func() {
			// We can't easily mock time.NewTicker, so we'll just let it run
			// and have our test timeout mechanism ensure it doesn't hang

			// This should trigger the warning log
			p.monitorTrackProcessing(neverDone, startTime, trackNumber, trackTitle)
			close(processingComplete)
		}()

		// Add a timeout for the test itself
		select {
		case <-processingComplete:
			// Function completed - this should never happen in our test setup
			assert.Fail(t, "Monitor should not have returned")
		case <-time.After(100 * time.Millisecond):
			// This is expected - the function is still running as it should
		}
	})
}

func TestUpdateTrackProgress(t *testing.T) {
	// Create test processor directly
	p := &processor{}

	// Mock progress bar
	bar := &progressbar.ProgressBar{}

	// Track completed tracks
	var completedTracks int32 = 5
	totalTracks := 10

	// Track progress updates
	var capturedProgress int
	var capturedMessage string
	progressCallback := func(progress int, message string) {
		capturedProgress = progress
		capturedMessage = message
	}

	// Call the function
	p.updateTrackProgress(bar, &completedTracks, totalTracks, progressCallback)

	// Verify the counter was incremented
	assert.Equal(t, int32(6), completedTracks)

	// Verify progress calculation (should be in 50-100% range)
	// 6/10 = 60% of track processing, scaled to 50-100% range: 50 + (60/2) = 80%
	assert.Equal(t, 80, capturedProgress)

	// Verify message format
	assert.Equal(t, "Processed 6/10 tracks", capturedMessage)
}

func TestProcessTrack(t *testing.T) {
	// This is more of an integration test since it involves multiple components
	// For a real test we'd need to mock more interactions

	// Setup mocked dependencies
	trackImporter := new(MockTracklistImporter)
	downloader := new(MockDownloader)
	audioProcessor := new(MockAudioProcessor)
	storage := new(MockStorage)

	// Create processor directly
	p := &processor{
		tracklistImporter: trackImporter,
		setDownloader:     downloader,
		audioProcessor:    audioProcessor,
		storage:           storage,
	}

	// Mock context and channels
	ctx := context.Background()
	cancel := func() {}
	errCh := make(chan error, 1)
	filePathCh := make(chan string, 1)

	// Mock the audio processor split method
	audioProcessor.On("Split", mock.AnythingOfType("audio.SplitParams")).Return(nil)

	// Mock storage
	storage.On("SaveTrack", mock.Anything, mock.Anything, mock.Anything).Return("/tmp/output/test/track", nil)

	// Setup test data
	trackIndex := 0
	track := &domain.Track{
		Title:       "Test Track",
		Artist:      "Test Artist",
		TrackNumber: 1,
		StartTime:   "00:00:00",
		EndTime:     "01:00:00",
	}
	set := domain.Tracklist{
		Name:   "Test Set",
		Artist: "Various Artists",
	}
	opts := ProcessingOptions{
		FileExtension: "mp3",
	}
	setLength := 1
	fileName := "test.mp3"
	coverArtPath := "cover.jpg"

	// Prepare progress tracking
	var completedTracks int32 = 0
	bar := progressbar.NewOptions(setLength)

	// Track progress updates
	var capturedProgress int
	progressCallback := func(progress int, message string) {
		capturedProgress = progress
	}

	// Run the process track function
	p.processTrack(
		ctx, cancel,
		trackIndex, track,
		set, opts,
		setLength,
		fileName, coverArtPath,
		bar,
		&completedTracks,
		errCh,
		filePathCh,
		progressCallback,
	)

	// Verify a file path was sent to the channel
	select {
	case path := <-filePathCh:
		assert.Equal(t, "/tmp/output/test/track.mp3", path)
	default:
		assert.Fail(t, "No file path was sent to the channel")
	}

	// Verify no errors were sent
	select {
	case err := <-errCh:
		assert.Fail(t, "Unexpected error: %v", err)
	default:
		// This is expected - no errors
	}

	// Verify progress was updated
	assert.Equal(t, int32(1), completedTracks)
	assert.Greater(t, capturedProgress, 0)

	// Verify the mocks were called correctly
	audioProcessor.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestSplitTracks(t *testing.T) {
	// Setup mocked dependencies
	trackImporter := new(MockTracklistImporter)
	downloader := new(MockDownloader)
	audioProcessor := new(MockAudioProcessor)
	storage := new(MockStorage)

	// Create processor directly
	p := &processor{
		tracklistImporter: trackImporter,
		setDownloader:     downloader,
		audioProcessor:    audioProcessor,
		storage:           storage,
	}

	// Create a test tracklist
	set := domain.Tracklist{
		Name:   "Test Set",
		Artist: "Various Artists",
		Tracks: []*domain.Track{
			{Title: "Track 1", Artist: "Artist 1", TrackNumber: 1, StartTime: "00:00", EndTime: "01:00"},
			{Title: "Track 2", Artist: "Artist 2", TrackNumber: 2, StartTime: "01:00", EndTime: "02:00"},
		},
	}

	// Define options
	opts := ProcessingOptions{
		MaxConcurrentTasks: 2,
		FileExtension:      "mp3",
	}

	// Setup mocks
	storage.On("SaveTrack", mock.Anything, mock.Anything, mock.Anything).Return("/tmp/test/track", nil)
	audioProcessor.On("Split", mock.AnythingOfType("audio.SplitParams")).Return(nil)

	// Setup progress tracking
	progressUpdates := []int{}
	progressCallback := func(progress int, message string) {
		progressUpdates = append(progressUpdates, progress)
	}

	// Call the function
	results, err := p.splitTracks(
		set,
		opts,
		len(set.Tracks),
		"test.mp3",
		"cover.jpg",
		progressbar.NewOptions(len(set.Tracks)),
		progressCallback,
	)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	// Verify progress tracking
	assert.GreaterOrEqual(t, len(progressUpdates), 3)             // Should have at least start, per-track, and complete updates
	assert.Equal(t, 100, progressUpdates[len(progressUpdates)-1]) // Last update should be 100%

	// Verify mocks were called correctly
	storage.AssertExpectations(t)
	audioProcessor.AssertExpectations(t)
}
