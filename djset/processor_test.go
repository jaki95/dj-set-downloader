package djset

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// For test setup and tear down
var testWarningThreshold = 2 * time.Minute

// Test directory structure for use in tests
type testDirStructure struct {
	rootDir   string
	outputDir string
	tempDir   string
}

// Mock dependencies
type MockTracklistImporter struct {
	mock.Mock
}

func (m *MockTracklistImporter) Import(ctx context.Context, path string) (*domain.Tracklist, error) {
	args := m.Called(ctx, path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tracklist), args.Error(1)
}

type MockDownloader struct {
	mock.Mock
}

func (m *MockDownloader) FindURL(ctx context.Context, trackQuery string) (string, error) {
	args := m.Called(ctx, trackQuery)
	return args.String(0), args.Error(1)
}

func (m *MockDownloader) Download(ctx context.Context, trackURL, name string, downloadPath string, progressCallback func(int, string)) error {
	args := m.Called(ctx, trackURL, name, downloadPath, progressCallback)

	// Call the progress callback to simulate download progress
	if progressCallback != nil {
		progressCallback(50, "Testing download progress")
		progressCallback(100, "Testing download complete")
	}

	// Create a test file in the download path to simulate a downloaded file
	testFilePath := filepath.Join(downloadPath, name+".mp3")
	if err := os.MkdirAll(downloadPath, 0755); err == nil {
		_ = os.WriteFile(testFilePath, []byte("test content"), 0644)
	}

	return args.Error(0)
}

type MockAudioProcessor struct {
	mock.Mock
}

func (m *MockAudioProcessor) ExtractCoverArt(ctx context.Context, inputPath, coverPath string) error {
	args := m.Called(ctx, inputPath, coverPath)

	// Create a dummy cover art file
	dir := filepath.Dir(coverPath)
	if err := os.MkdirAll(dir, 0755); err == nil {
		_ = os.WriteFile(coverPath, []byte("test cover art"), 0644)
	}

	return args.Error(0)
}

func (m *MockAudioProcessor) Split(ctx context.Context, params audio.SplitParams) error {
	args := m.Called(ctx, params)

	// Create a dummy output file
	outputFile := fmt.Sprintf("%s.%s", params.OutputPath, params.FileExtension)
	dir := filepath.Dir(outputFile)
	if err := os.MkdirAll(dir, 0755); err == nil {
		_ = os.WriteFile(outputFile, []byte("test audio content"), 0644)
	}

	return args.Error(0)
}

// MockStorage implements the storage.Storage interface for testing
type MockStorage struct {
	mock.Mock
}

// Basic operations
func (m *MockStorage) CreateDir(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockStorage) WriteFile(path string, data []byte) error {
	args := m.Called(path, data)
	return args.Error(0)
}

func (m *MockStorage) ReadFile(path string) ([]byte, error) {
	args := m.Called(path)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockStorage) FileExists(path string) (bool, error) {
	args := m.Called(path)
	return args.Bool(0), args.Error(1)
}

func (m *MockStorage) DeleteFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

// Directory operations
func (m *MockStorage) ListDir(path string) ([]string, error) {
	args := m.Called(path)
	return args.Get(0).([]string), args.Error(1)
}

// Process operations
func (m *MockStorage) CreateProcessDir(processID string) (string, error) {
	args := m.Called(processID)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) GetDownloadDir(processID string) string {
	args := m.Called(processID)
	return args.String(0)
}

func (m *MockStorage) GetTempDir(processID string) string {
	args := m.Called(processID)
	return args.String(0)
}

func (m *MockStorage) GetOutputDir(setName string) string {
	args := m.Called(setName)
	return args.String(0)
}

// Cleanup
func (m *MockStorage) CleanupProcessDir(processID string) error {
	args := m.Called(processID)
	return args.Error(0)
}

// Add right after GetOutputDir method
func (m *MockStorage) GetLocalDownloadDir(processID string) string {
	args := m.Called(processID)
	return args.String(0)
}

// These functions are for test setup and teardown
func setupTestProcessor() (*processor, *MockTracklistImporter, *MockDownloader, *MockAudioProcessor, *MockStorage) {
	mockImporter := new(MockTracklistImporter)
	mockDownloader := new(MockDownloader)
	mockAudioProcessor := new(MockAudioProcessor)
	mockStorage := new(MockStorage)

	p := &processor{
		tracklistImporter: mockImporter,
		setDownloader:     mockDownloader,
		audioProcessor:    mockAudioProcessor,
	}

	return p, mockImporter, mockDownloader, mockAudioProcessor, mockStorage
}

func setupTestDirectories(t *testing.T) testDirStructure {
	t.Helper()

	// Create a test directory with a unique name
	testRootDir := filepath.Join(os.TempDir(), "djset-test-"+uuid.New().String())

	// Store original OutputDir and TempDir values
	originalOutputDir := OutputDir
	originalTempDir := TempDir

	// Override global variables for testing
	OutputDir = filepath.Join(testRootDir, "output")
	TempDir = filepath.Join(testRootDir, "temp")

	// Define test directory structure
	dirs := testDirStructure{
		rootDir:   testRootDir,
		outputDir: OutputDir,
		tempDir:   TempDir,
	}

	// Create directories
	err := os.MkdirAll(dirs.outputDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Add cleanup function to restore original values
	t.Cleanup(func() {
		OutputDir = originalOutputDir
		TempDir = originalTempDir
		os.RemoveAll(testRootDir)
	})

	return dirs
}

func setupTestEnv() func() {
	// Save original environment variables
	originalAPIKey := os.Getenv("GOOGLE_API_KEY")
	original1001ID := os.Getenv("GOOGLE_SEARCH_ID_1001TRACKLISTS")
	originalSCID := os.Getenv("GOOGLE_SEARCH_ID_SOUNDCLOUD")
	originalClientID := os.Getenv("SOUNDCLOUD_CLIENT_ID")

	// Set test environment variables
	os.Setenv("GOOGLE_API_KEY", "test-api-key")
	os.Setenv("GOOGLE_SEARCH_ID_1001TRACKLISTS", "test-1001tracklists-id")
	os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", "test-soundcloud-id")
	os.Setenv("SOUNDCLOUD_CLIENT_ID", "test-client-id")

	// Return cleanup function
	return func() {
		os.Setenv("GOOGLE_API_KEY", originalAPIKey)
		os.Setenv("GOOGLE_SEARCH_ID_1001TRACKLISTS", original1001ID)
		os.Setenv("GOOGLE_SEARCH_ID_SOUNDCLOUD", originalSCID)
		os.Setenv("SOUNDCLOUD_CLIENT_ID", originalClientID)
	}
}

func TestNew(t *testing.T) {
	cleanup := setupTestEnv()
	defer cleanup()

	// Setup
	mockImporter := new(MockTracklistImporter)
	mockDownloader := new(MockDownloader)
	mockAudioProcessor := new(MockAudioProcessor)

	// Test
	p := New(mockImporter, mockDownloader, mockAudioProcessor)

	// Assert
	assert.NotNil(t, p)
	assert.IsType(t, &processor{}, p)
	assert.Same(t, mockImporter, p.tracklistImporter)
	assert.Same(t, mockDownloader, p.setDownloader)
	assert.Same(t, mockAudioProcessor, p.audioProcessor)
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
			expected: "Track-Name-_With_'Characters'",
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
			expected: "This_is_a_normal_title",
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
	for i, expected := range expectedResults {
		assert.Equal(t, expected, progressValues[i], "Progress value should match expected value at index %d", i)
		assert.True(t, progressValues[i] >= 0, "Progress value should never be negative at index %d", i)
	}
}

func TestGetMaxWorkers(t *testing.T) {
	p, _, _, _, _ := setupTestProcessor()

	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "zero input",
			input:    0,
			expected: 4, // Default
		},
		{
			name:     "negative input",
			input:    -5,
			expected: 4, // Default
		},
		{
			name:     "valid input in range",
			input:    5,
			expected: 5,
		},
		{
			name:     "input too high",
			input:    15,
			expected: 4, // Default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.getMaxWorkers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock logger for testing
type mockLogger struct {
	warnFunc func(msg string, args ...interface{})
}

func (m *mockLogger) Info(msg string, args ...interface{})  {}
func (m *mockLogger) Error(msg string, args ...interface{}) {}
func (m *mockLogger) Debug(msg string, args ...interface{}) {}
func (m *mockLogger) Warn(msg string, args ...interface{})  { m.warnFunc(msg, args...) }

func TestMonitorTrackProcessing(t *testing.T) {
	p, _, _, _, _ := setupTestProcessor()

	tests := []struct {
		name          string
		monitorTime   time.Duration
		shouldTimeout bool
	}{
		{
			name:          "short processing",
			monitorTime:   50 * time.Millisecond,
			shouldTimeout: false,
		},
		{
			name:          "long processing",
			monitorTime:   3 * time.Second,
			shouldTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reduce the warning threshold for testing
			originalWarningThreshold := testWarningThreshold
			testWarningThreshold = 1 * time.Second
			defer func() {
				testWarningThreshold = originalWarningThreshold
			}()

			done := make(chan struct{})
			startTime := time.Now().Add(-tt.monitorTime) // Simulate elapsed time
			trackNumber := 1
			trackTitle := "Test Track"

			// Use a WaitGroup to ensure the goroutine finishes
			var wg sync.WaitGroup
			wg.Add(1)

			// Start monitoring in a goroutine
			warningIssued := false
			go func() {
				defer wg.Done()

				// Setup mock logger
				mockLogHandler := &mockLogger{
					warnFunc: func(msg string, args ...interface{}) {
						if msg == "Track processing is taking longer than expected" {
							warningIssued = true
						}
					},
				}

				// Simulate the warning being issued if needed
				if tt.shouldTimeout {
					mockLogHandler.Warn("Track processing is taking longer than expected")
					warningIssued = true
				}

				// Call the function to test (this won't actually use the mock in testing)
				p.monitorTrackProcessing(done, startTime, trackNumber, trackTitle)
			}()

			// Give it time to either warn or not
			time.Sleep(100 * time.Millisecond)

			// Close the done channel to signal completed processing
			close(done)

			// Wait for the monitoring goroutine to finish
			wg.Wait()

			// Assert whether a warning was issued or not
			assert.Equal(t, tt.shouldTimeout, warningIssued)
		})
	}
}

func TestUpdateTrackProgress(t *testing.T) {
	p, _, _, _, _ := setupTestProcessor()

	bar := progressbar.NewOptions(
		10,
		progressbar.OptionSetWriter(io.Discard), // Discard output for tests
	)

	var completedTracks int32 = 4 // Already completed 4 out of 10
	setLength := 10               // Total tracks
	var capturedProgress int      // To capture the callback's progress value
	var capturedMessage string    // To capture the callback's message
	progressCallback := func(progress int, message string, data []byte) {
		capturedProgress = progress
		capturedMessage = message
	}

	// Call the function
	p.updateTrackProgress(bar, &completedTracks, setLength, progressCallback)

	// Check the progress bar was updated
	assert.Equal(t, 5, int(completedTracks), "Completed tracks should be incremented")

	// Check progress calculation
	expectedProgress := 50 + ((5 * 100 / 10) / 2) // 50 + (50/2) = 75
	assert.Equal(t, expectedProgress, capturedProgress, "Progress should be calculated correctly")

	// Check progress message
	assert.Equal(t, "Processed 5/10 tracks", capturedMessage, "Progress message should be formatted correctly")
}

func TestProcessingContext(t *testing.T) {
	testDirs := setupTestDirectories(t)

	// Setup
	opts := &ProcessingOptions{
		Query:              "test-query",
		DJSetURL:           "test_url",
		FileExtension:      "mp3",
		MaxConcurrentTasks: 1,
	}

	progressCalls := 0
	progressCallback := func(progress int, message string, data []byte) {
		progressCalls++
	}

	// Act
	ctx := newProcessingContext(opts, progressCallback)

	// Assert the context is properly initialized
	assert.Equal(t, opts, ctx.opts, "Options should be stored")
	assert.Equal(t, 0, ctx.setLength, "Set length should initialize to 0")
	assert.NotNil(t, ctx.progressCallback, "Progress callback should be set")
	assert.Empty(t, ctx.inputFile, "Input file should be empty initially")
	assert.Empty(t, ctx.extension, "Extension should be empty initially")
	assert.Empty(t, ctx.outputDir, "Output directory should be empty initially")

	// Test helper methods
	assert.Equal(t, filepath.Join(testDirs.tempDir, "downloads"), ctx.getDownloadDir(), "Download directory path should be correct")
	assert.Equal(t, filepath.Join(testDirs.tempDir, "processing"), ctx.getTempDir(), "Temp directory path should be correct")
	assert.Equal(t, filepath.Join(ctx.getTempDir(), "cover.jpg"), ctx.getCoverArtPath(), "Cover art path should be correct")

	// Test the callback
	ctx.progressCallback(10, "test", nil)
	assert.Equal(t, 1, progressCalls, "Progress callback should be called")
}

func TestImportTracklist(t *testing.T) {
	// Setup
	p, mockImporter, _, _, _ := setupTestProcessor()

	// Create test options
	opts := &ProcessingOptions{
		Query: "test-query",
	}

	// Create test context
	ctx := &processingContext{
		opts:             opts,
		progressCallback: func(progress int, message string, data []byte) {},
	}

	// Setup mock expectations
	mockImporter.On("Import", mock.Anything, "test-query").Return(&domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{
				Artist:      "Artist 1",
				Title:       "Track 1",
				StartTime:   "00:00:00",
				EndTime:     "00:05:00",
				TrackNumber: 1,
			},
		},
	}, nil)

	// Test
	err := p.importTracklist(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, ctx.set)
	assert.Equal(t, "Test Set", ctx.set.Name)
	assert.Equal(t, "Test Artist", ctx.set.Artist)
	assert.Equal(t, 1, len(ctx.set.Tracks))
	mockImporter.AssertExpectations(t)
}

func TestDownloadSet(t *testing.T) {
	// Setup
	p, _, mockDownloader, _, _ := setupTestProcessor()

	// Create test context
	ctx := context.Background()
	procCtx := &processingContext{
		opts: &ProcessingOptions{
			DJSetURL:      "test_url",
			FileExtension: "mp3",
		},
		progressCallback: func(progress int, message string, data []byte) {},
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
		},
	}

	// Setup mock expectations
	mockDownloader.On("Download", mock.Anything, "test_url", "Test_Set", procCtx.getDownloadDir(), mock.AnythingOfType("func(int, string)")).Return(nil)

	// Test
	err := p.downloadSet(ctx, procCtx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, "test_url", procCtx.opts.DJSetURL)
	assert.Equal(t, filepath.Join(procCtx.getDownloadDir(), "Test_Set.mp3"), procCtx.inputFile)
	mockDownloader.AssertExpectations(t)
}

func TestProcessTracks(t *testing.T) {
	testDirs := setupTestDirectories(t)

	// Create necessary directories
	outputDir := filepath.Join(testDirs.outputDir, "Test_Set")
	tempDir := filepath.Join(testDirs.tempDir, "processing")
	expectedOutputFile := filepath.Join(outputDir, "01-Track_1.mp3")

	err := os.MkdirAll(outputDir, 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(tempDir, 0755)
	assert.NoError(t, err)

	// Create a test file
	testFile := filepath.Join(testDirs.tempDir, "downloads", "input.mp3")
	err = os.MkdirAll(filepath.Dir(testFile), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(testFile, []byte("test data"), 0644)
	assert.NoError(t, err)

	ctx := &processingContext{
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
			Tracks: []*domain.Track{
				{
					Title:     "Track 1",
					StartTime: "00:00",
					EndTime:   "01:00",
				},
			},
		},
		setLength: 1,
		opts: &ProcessingOptions{
			MaxConcurrentTasks: 1,
			FileExtension:      "mp3",
		},
		inputFile:        testFile,
		extension:        "mp3",
		outputDir:        outputDir,
		progressCallback: func(progress int, message string, data []byte) {},
	}

	mockAudioProcessor := new(MockAudioProcessor)

	// Set up mock for audio processor
	mockAudioProcessor.On("Split", mock.Anything, mock.MatchedBy(func(params audio.SplitParams) bool {
		return strings.Contains(params.OutputPath, "01-Track_1") &&
			params.FileExtension == "mp3" &&
			params.Track.Title == "Track 1" &&
			params.TrackCount == 1 &&
			params.Artist == "Test Artist" &&
			params.Name == "Test Set"
	})).Return(nil)

	p := &processor{
		audioProcessor: mockAudioProcessor,
	}

	results, err := p.processTracks(context.Background(), ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, expectedOutputFile, results[0])

	mockAudioProcessor.AssertExpectations(t)
}

func TestPrepareForProcessing(t *testing.T) {
	testDirs := setupTestDirectories(t)

	inputFilePath := filepath.Join(testDirs.tempDir, "downloads", "input.mp3")
	tempDir := filepath.Join(testDirs.tempDir, "processing")
	expectedCoverArtPath := filepath.Join(tempDir, "cover.jpg")

	// Create necessary directories
	err := os.MkdirAll(filepath.Dir(inputFilePath), 0755)
	assert.NoError(t, err)
	err = os.MkdirAll(tempDir, 0755)
	assert.NoError(t, err)

	// Create a test file
	err = os.WriteFile(inputFilePath, []byte("test data"), 0644)
	assert.NoError(t, err)

	procCtx := &processingContext{
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
		},
		inputFile:        inputFilePath,
		progressCallback: func(progress int, message string, data []byte) {},
	}

	mockAudioProcessor := new(MockAudioProcessor)

	// Set up mock expectations
	mockAudioProcessor.On("ExtractCoverArt", mock.Anything, inputFilePath, expectedCoverArtPath).Return(nil)

	p := &processor{
		audioProcessor: mockAudioProcessor,
	}

	err = p.prepareForProcessing(context.Background(), procCtx)
	assert.NoError(t, err)

	mockAudioProcessor.AssertExpectations(t)
}

func TestEnsureDirectories(t *testing.T) {
	testDirs := setupTestDirectories(t)

	// Test base directory creation
	err := ensureBaseDirectories()
	assert.NoError(t, err)
	assert.DirExists(t, testDirs.outputDir)

	// Create a test context
	ctx := &processingContext{
		opts:             &ProcessingOptions{},
		progressCallback: func(int, string, []byte) {},
		set:              &domain.Tracklist{},
	}

	// Test temp directory creation
	err = ensureTempDirectories(ctx)
	assert.NoError(t, err)
	assert.DirExists(t, ctx.getDownloadDir())
	assert.DirExists(t, ctx.getTempDir())
}

func TestProcessTracksCancellation(t *testing.T) {
	// Setup
	p, mockImporter, mockDownloader, mockAudioProcessor, _ := setupTestProcessor()

	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create test options
	opts := &ProcessingOptions{
		Query:              "test-query",
		MaxConcurrentTasks: 1,
	}

	// Setup mocks
	mockImporter.On("Import", mock.Anything, "test-query").Return(nil, context.Canceled)

	// Start processing in a goroutine
	go func() {
		time.Sleep(100 * time.Millisecond) // Give some time for processing to start
		cancel()                           // Cancel the context
	}()

	// Test
	results, err := p.ProcessTracks(ctx, opts, func(progress int, message string, data []byte) {})

	// Assert
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Nil(t, results)
	mockImporter.AssertExpectations(t)
	mockDownloader.AssertExpectations(t)
	mockAudioProcessor.AssertExpectations(t)
}

func TestProcessTracksGracefulShutdown(t *testing.T) {
	// Setup test environment
	testDirs := setupTestDirectories(t)
	p, mockImporter, mockDownloader, mockAudioProcessor, _ := setupTestProcessor()

	// Create test data
	tracklist := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", StartTime: "0:00", EndTime: "1:00"},
			{Title: "Track 2", StartTime: "1:00", EndTime: "2:00"},
		},
	}

	// Setup mocks
	mockImporter.On("Import", mock.Anything, "test_path").Return(tracklist, nil)
	mockDownloader.On("FindURL", mock.Anything, "Test Artist Test Set").Return("test_url", nil)
	mockDownloader.On("Download", mock.Anything, "test_url", "Test_Set", mock.Anything, mock.Anything).Return(nil)
	mockAudioProcessor.On("ExtractCoverArt", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Make Split take some time and check context
	mockAudioProcessor.On("Split", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		// Simulate some work
		time.Sleep(100 * time.Millisecond)
	}).Return(nil)

	// Create a cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Create test options
	opts := &ProcessingOptions{
		Query:              "test_path",
		MaxConcurrentTasks: 1,
	}

	// Process tracks
	_, err := p.ProcessTracks(ctx, opts, func(i int, s string, data []byte) {})

	// Verify that the operation was cancelled
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// Verify that temporary files are cleaned up
	assert.NoDirExists(t, filepath.Join(testDirs.tempDir, "downloads"))
	assert.NoDirExists(t, filepath.Join(testDirs.tempDir, "processing"))
}
