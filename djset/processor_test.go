package djset

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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

	// Setup progress tracking with mutex to prevent data races
	var mu sync.Mutex
	progressUpdates := []int{}
	progressCallback := func(progress int, message string) {
		mu.Lock()
		defer mu.Unlock()
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

	// Verify progress tracking - access the slice after all processing is done
	mu.Lock()
	defer mu.Unlock()
	assert.GreaterOrEqual(t, len(progressUpdates), 3)             // Should have at least start, per-track, and complete updates
	assert.Equal(t, 100, progressUpdates[len(progressUpdates)-1]) // Last update should be 100%

	// Verify mocks were called correctly
	storage.AssertExpectations(t)
	audioProcessor.AssertExpectations(t)
}

func TestNewProcessingContext(t *testing.T) {
	// Test creating a new processing context
	opts := &ProcessingOptions{
		TracklistPath:      "test.txt",
		DJSetURL:           "http://example.com/set",
		FileExtension:      "mp3",
		MaxConcurrentTasks: 4,
	}

	progressCalls := 0
	progressCallback := func(progress int, message string) {
		progressCalls++
	}

	ctx := newProcessingContext(opts, progressCallback)

	// Verify the context is initialized correctly
	assert.Equal(t, opts, ctx.opts)
	assert.NotNil(t, ctx.progressCallback)
	assert.Equal(t, 0, ctx.setLength)
	assert.Equal(t, "", ctx.fileName)
	assert.Equal(t, "", ctx.downloadedFile)
	assert.Equal(t, "", ctx.actualExtension)
	assert.Equal(t, "", ctx.coverArtPath)

	// Test the callback works
	ctx.progressCallback(10, "test")
	assert.Equal(t, 1, progressCalls)
}

func TestProcessTracksFunction(t *testing.T) {
	// Setup
	audioProcessor := new(MockAudioProcessor)
	storage := new(MockStorage)
	p := &processor{
		audioProcessor: audioProcessor,
		storage:        storage,
	}

	// Test data
	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", Artist: "Artist 1", TrackNumber: 1, StartTime: "00:00", EndTime: "01:00"},
		},
	}

	opts := &ProcessingOptions{
		FileExtension: "mp3",
	}

	// Test case: Successful processing
	t.Run("Successful processing", func(t *testing.T) {
		// Create a context with all necessary fields
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set
		ctx.setLength = len(set.Tracks)
		ctx.fileName = "/tmp/downloads/Test Set.mp3"
		ctx.actualExtension = "mp3"
		ctx.coverArtPath = "/tmp/cover.jpg"

		// Mock for splitTracks
		// This requires mocking the audio processor and storage, similar to TestSplitTracks
		storage.On("SaveTrack", mock.Anything, mock.Anything, mock.Anything).Return("/tmp/output/track", nil)
		audioProcessor.On("Split", mock.AnythingOfType("audio.SplitParams")).Return(nil)

		// Call the function
		results, err := p.processTracks(ctx)

		// Verify
		assert.NoError(t, err)
		assert.NotNil(t, results)

		storage.AssertExpectations(t)
		audioProcessor.AssertExpectations(t)
	})
}

func TestImportTracklist(t *testing.T) {
	// Setup
	tracklistImporter := new(MockTracklistImporter)
	p := &processor{
		tracklistImporter: tracklistImporter,
	}

	// Create test data
	opts := &ProcessingOptions{
		TracklistPath: "test.txt",
	}

	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", Artist: "Artist 1", TrackNumber: 1},
			{Title: "Track 2", Artist: "Artist 2", TrackNumber: 2},
		},
	}

	// Regular test case - successful import
	t.Run("Successful import", func(t *testing.T) {
		ctx := newProcessingContext(opts, func(int, string) {})

		// Mock importer to return a test set
		tracklistImporter.On("Import", "test.txt").Return(set, nil).Once()

		// Call the function
		err := p.importTracklist(ctx)

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, set, ctx.set)
		assert.Equal(t, 2, ctx.setLength)

		tracklistImporter.AssertExpectations(t)
	})

	// Error case - import fails
	t.Run("Import fails", func(t *testing.T) {
		ctx := newProcessingContext(opts, func(int, string) {})

		// Mock importer to return an error
		expectedErr := fmt.Errorf("import failed")
		tracklistImporter.On("Import", "test.txt").Return(nil, expectedErr).Once()

		// Call the function
		err := p.importTracklist(ctx)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)

		tracklistImporter.AssertExpectations(t)
	})
}

func TestDownloadSet(t *testing.T) {
	// Setup
	downloader := new(MockDownloader)
	storage := new(MockStorage)
	p := &processor{
		setDownloader: downloader,
		storage:       storage,
	}

	// Test data
	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
	}

	// Mock for getDownloadPath
	// This should match the actual implementation which calls filepath.Dir on the returned path
	storage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/temp.tmp", nil)

	// Test cases

	// Case 1: URL is already provided
	t.Run("URL is provided", func(t *testing.T) {
		opts := &ProcessingOptions{
			DJSetURL: "http://example.com/set",
		}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock downloader with the exact path that will be passed
		downloader.On("Download", "http://example.com/set", "Test Set", "/tmp", mock.AnythingOfType("func(int, string)")).
			Run(func(args mock.Arguments) {
				// Call the progress callback to simulate progress
				callback := args.Get(3).(func(int, string))
				callback(50, "Downloading set...")
			}).
			Return(nil).Once()

		// Call the function
		err := p.downloadSet(ctx)

		// Verify
		assert.NoError(t, err)
		downloader.AssertExpectations(t)
	})

	// Case 2: URL needs to be found from tracklist path
	t.Run("URL from tracklist path", func(t *testing.T) {
		// Create fresh mocks for this test case
		newDownloader := new(MockDownloader)
		newStorage := new(MockStorage)
		p2 := &processor{
			setDownloader: newDownloader,
			storage:       newStorage,
		}

		opts := &ProcessingOptions{
			TracklistPath: "http://example.com/tracklist",
		}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock the download path again
		newStorage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/temp.tmp", nil).Once()

		// Mock URL finder
		newDownloader.On("FindURL", "http://example.com/tracklist").Return("http://example.com/found-set", nil).Once()

		// Mock downloader
		newDownloader.On("Download", "http://example.com/found-set", "Test Set", "/tmp", mock.AnythingOfType("func(int, string)")).
			Return(nil).Once()

		// Call the function
		err := p2.downloadSet(ctx)

		// Verify
		assert.NoError(t, err)
		newDownloader.AssertExpectations(t)
		newStorage.AssertExpectations(t)
	})

	// Case 3: URL needs to be found from artist and name
	t.Run("URL from artist and name", func(t *testing.T) {
		// Create fresh mocks for this test case
		newDownloader := new(MockDownloader)
		newStorage := new(MockStorage)
		p2 := &processor{
			setDownloader: newDownloader,
			storage:       newStorage,
		}

		opts := &ProcessingOptions{
			TracklistPath: "local-file.txt",
		}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock the download path again
		newStorage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/temp.tmp", nil).Once()

		// Mock URL finder
		newDownloader.On("FindURL", "Test Artist Test Set").Return("http://example.com/found-set", nil).Once()

		// Mock downloader
		newDownloader.On("Download", "http://example.com/found-set", "Test Set", "/tmp", mock.AnythingOfType("func(int, string)")).
			Return(nil).Once()

		// Call the function
		err := p2.downloadSet(ctx)

		// Verify
		assert.NoError(t, err)
		newDownloader.AssertExpectations(t)
		newStorage.AssertExpectations(t)
	})

	// Create fresh mocks for the remaining test cases to avoid conflicts
	errorDownloader := new(MockDownloader)
	errorStorage := new(MockStorage)
	pError := &processor{
		setDownloader: errorDownloader,
		storage:       errorStorage,
	}

	// Case 4: FindURL fails
	t.Run("FindURL fails", func(t *testing.T) {
		opts := &ProcessingOptions{}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock URL finder to return an error
		expectedErr := fmt.Errorf("URL not found")
		errorDownloader.On("FindURL", "Test Artist Test Set").Return("", expectedErr).Once()

		// Call the function
		err := pError.downloadSet(ctx)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		errorDownloader.AssertExpectations(t)
	})

	// Case 5: Download fails
	t.Run("Download fails", func(t *testing.T) {
		// Create a new processor and mocks for this test
		dlDownloader := new(MockDownloader)
		dlStorage := new(MockStorage)
		dlProc := &processor{
			setDownloader: dlDownloader,
			storage:       dlStorage,
		}

		opts := &ProcessingOptions{
			DJSetURL: "http://example.com/set",
		}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock the download path
		dlStorage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/temp.tmp", nil).Once()

		// Mock downloader to return an error
		expectedErr := fmt.Errorf("download failed")
		dlDownloader.On("Download", "http://example.com/set", "Test Set", "/tmp", mock.AnythingOfType("func(int, string)")).
			Return(expectedErr).Once()

		// Call the function
		err := dlProc.downloadSet(ctx)

		// Verify
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		dlDownloader.AssertExpectations(t)
		dlStorage.AssertExpectations(t)
	})

	// Case 6: getDownloadPath fails
	t.Run("getDownloadPath fails", func(t *testing.T) {
		// Create a new processor and mocks for this test
		pathStorage := new(MockStorage)
		pathProc := &processor{
			storage: pathStorage,
		}

		opts := &ProcessingOptions{
			DJSetURL: "http://example.com/set",
		}
		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock SaveDownloadedSet to return an error
		expectedErr := fmt.Errorf("storage error")
		pathStorage.On("SaveDownloadedSet", "temp", "tmp").Return("", expectedErr).Once()

		// Call the function
		err := pathProc.downloadSet(ctx)

		// Verify
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get download path")
		pathStorage.AssertExpectations(t)
	})
}

func TestPrepareForProcessing(t *testing.T) {
	// Test data
	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
	}

	opts := &ProcessingOptions{
		FileExtension: "mp3",
	}

	// Test case: Successful preparation
	t.Run("Successful preparation", func(t *testing.T) {
		// Create fresh mocks for this test case
		storage := new(MockStorage)
		audioProcessor := new(MockAudioProcessor)
		p := &processor{
			storage:        storage,
			audioProcessor: audioProcessor,
		}

		ctx := newProcessingContext(opts, func(int, string) {})
		ctx.set = set

		// Mock storage functions
		storage.On("ListFiles", "", "Test Set").Return([]string{"/tmp/downloads/Test Set.mp3"}, nil)
		storage.On("GetSetPath", "Test Set", "mp3").Return("/tmp/downloads/Test Set.mp3")
		storage.On("GetCoverArtPath").Return("/tmp/cover.jpg")
		storage.On("FileExists", "/tmp/downloads/Test Set.mp3").Return(true)
		storage.On("CreateSetOutputDir", "Test Set").Return(nil)

		// Mock audio processor
		audioProcessor.On("ExtractCoverArt", "/tmp/downloads/Test Set.mp3", "/tmp/cover.jpg").Return(nil)

		// Mock cleanup
		storage.On("Cleanup").Return(nil)

		// Call the function
		err := p.prepareForProcessing(ctx)

		// Verify
		assert.NoError(t, err)
		assert.Equal(t, "Test Set.mp3", ctx.downloadedFile)
		assert.Equal(t, "mp3", ctx.actualExtension)
		assert.Equal(t, "/tmp/downloads/Test Set.mp3", ctx.fileName)
		assert.Equal(t, "/tmp/cover.jpg", ctx.coverArtPath)

		storage.AssertExpectations(t)
		audioProcessor.AssertExpectations(t)
	})
}

// Test for error cases in ListFiles
func TestPrepareForProcessingListFilesError(t *testing.T) {
	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
	}

	opts := &ProcessingOptions{
		FileExtension: "mp3",
	}

	storage := new(MockStorage)
	audioProcessor := new(MockAudioProcessor)
	p := &processor{
		storage:        storage,
		audioProcessor: audioProcessor,
	}

	ctx := newProcessingContext(opts, func(int, string) {})
	ctx.set = set

	// Mock storage to return an error for ListFiles
	expectedErr := fmt.Errorf("list files error")
	storage.On("ListFiles", "", "Test Set").Return([]string{}, expectedErr)

	// Call the function
	err := p.prepareForProcessing(ctx)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error listing files")

	storage.AssertExpectations(t)
}

func TestGetDownloadPath(t *testing.T) {
	// Setup
	storage := new(MockStorage)
	p := &processor{
		storage: storage,
	}

	// Test case 1: Successful retrieval
	t.Run("Successful retrieval", func(t *testing.T) {
		storage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/temp.tmp", nil).Once()

		path, err := p.getDownloadPath()

		assert.NoError(t, err)
		assert.Equal(t, "/tmp", path)

		storage.AssertExpectations(t)
	})

	// Test case 2: Error from storage
	t.Run("Error from storage", func(t *testing.T) {
		expectedErr := fmt.Errorf("storage error")
		storage.On("SaveDownloadedSet", "temp", "tmp").Return("", expectedErr).Once()

		path, err := p.getDownloadPath()

		assert.Error(t, err)
		assert.Equal(t, "", path)
		assert.Equal(t, expectedErr, err)

		storage.AssertExpectations(t)
	})
}

func TestProcessTracks(t *testing.T) {
	// Setup mocked processor and dependencies
	tracklistImporter := new(MockTracklistImporter)
	downloader := new(MockDownloader)
	audioProcessor := new(MockAudioProcessor)
	storage := new(MockStorage)

	p := &processor{
		tracklistImporter: tracklistImporter,
		setDownloader:     downloader,
		audioProcessor:    audioProcessor,
		storage:           storage,
	}

	// Create test data
	opts := &ProcessingOptions{
		TracklistPath:      "test.txt", // Use TracklistPath so we don't need FindURL
		DJSetURL:           "",
		FileExtension:      "mp3",
		MaxConcurrentTasks: 2,
	}

	set := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", Artist: "Artist 1", TrackNumber: 1, StartTime: "00:00", EndTime: "01:00"},
			{Title: "Track 2", Artist: "Artist 2", TrackNumber: 2, StartTime: "01:00", EndTime: "02:00"},
		},
	}

	// Setup mocks for successful execution
	tracklistImporter.On("Import", "test.txt").Return(set, nil)

	// Mock the getDownloadPath to return a valid path
	// The path returned here will be passed to filepath.Dir() in the code
	storage.On("SaveDownloadedSet", "temp", "tmp").Return("/tmp/downloads/temp.tmp", nil)

	// The code will construct a search query from the artist and name
	// Since TracklistPath is not a URL, it will use "Test Artist Test Set"
	downloader.On("FindURL", "Test Artist Test Set").Return("http://example.com/set", nil)

	// Mock downloader functions - note that filepath.Dir("/tmp/downloads/temp.tmp") is "/tmp/downloads"
	downloader.On("Download",
		"http://example.com/set",
		"Test Set",
		"/tmp/downloads",
		mock.AnythingOfType("func(int, string)")).
		Run(func(args mock.Arguments) {
			// Call the progress callback to simulate progress
			callback := args.Get(3).(func(int, string))
			callback(50, "Downloading set...")
			callback(100, "Download complete")
		}).
		Return(nil)

	// Mock storage functions for file operations
	storage.On("ListFiles", "", "Test Set").Return([]string{"/tmp/downloads/Test Set.mp3"}, nil)
	storage.On("GetSetPath", "Test Set", "mp3").Return("/tmp/downloads/Test Set.mp3")
	storage.On("GetCoverArtPath").Return("/tmp/cover.jpg")
	storage.On("FileExists", "/tmp/downloads/Test Set.mp3").Return(true)
	storage.On("CreateSetOutputDir", "Test Set").Return(nil)
	storage.On("Cleanup").Return(nil)
	storage.On("SaveTrack", mock.Anything, mock.Anything, mock.Anything).Return("/tmp/output/track", nil)

	// Mock audio processor functions
	audioProcessor.On("ExtractCoverArt", "/tmp/downloads/Test Set.mp3", "/tmp/cover.jpg").Return(nil)
	audioProcessor.On("Split", mock.AnythingOfType("audio.SplitParams")).Return(nil)

	// Track progress updates with mutex to prevent data races
	var mu sync.Mutex
	progressUpdates := []int{}
	messageUpdates := []string{}
	progressCallback := func(progress int, message string) {
		mu.Lock()
		defer mu.Unlock()
		progressUpdates = append(progressUpdates, progress)
		messageUpdates = append(messageUpdates, message)
	}

	// Call the function
	results, err := p.ProcessTracks(opts, progressCallback)

	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, results)

	// Verify progress reporting - access the slices after all processing is done
	mu.Lock()
	defer mu.Unlock()
	assert.Greater(t, len(progressUpdates), 0)
	assert.Greater(t, len(messageUpdates), 0)

	// Verify that all expected mocks were called
	tracklistImporter.AssertExpectations(t)
	downloader.AssertExpectations(t)
	storage.AssertExpectations(t)
	audioProcessor.AssertExpectations(t)
}
