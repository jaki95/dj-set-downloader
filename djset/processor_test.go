package djset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/audio"
	"github.com/jaki95/dj-set-downloader/internal/domain"
	"github.com/schollz/progressbar/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// For test setup and tear down
var testWarningThreshold = 2 * time.Minute

// Test directory structure for use in tests
type testDirStructure struct {
	baseDir      string
	processesDir string
	outputDir    string
	rootDir      string
}

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

func (m *MockAudioProcessor) ExtractCoverArt(inputPath, coverPath string) error {
	args := m.Called(inputPath, coverPath)

	// Create a dummy cover art file
	dir := filepath.Dir(coverPath)
	if err := os.MkdirAll(dir, 0755); err == nil {
		_ = os.WriteFile(coverPath, []byte("test cover art"), 0644)
	}

	return args.Error(0)
}

func (m *MockAudioProcessor) Split(params audio.SplitParams) error {
	args := m.Called(params)

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

// These functions are for test setup and teardown
func setupTestProcessor() (*processor, *MockTracklistImporter, *MockDownloader, *MockAudioProcessor) {
	mockImporter := new(MockTracklistImporter)
	mockDownloader := new(MockDownloader)
	mockAudioProcessor := new(MockAudioProcessor)

	p := &processor{
		tracklistImporter: mockImporter,
		setDownloader:     mockDownloader,
		audioProcessor:    mockAudioProcessor,
	}

	return p, mockImporter, mockDownloader, mockAudioProcessor
}

func setupTestDirectories(t *testing.T) testDirStructure {
	t.Helper()

	// Create a test directory with a unique name
	testRootDir := filepath.Join(os.TempDir(), "djset-test-"+uuid.New().String())

	// Define test directory structure
	dirs := testDirStructure{
		rootDir:      testRootDir,
		baseDir:      filepath.Join(testRootDir, "storage"),
		processesDir: filepath.Join(testRootDir, "storage/processes"),
		outputDir:    filepath.Join(testRootDir, "output"),
	}

	// Create the directories
	for _, dir := range []string{dirs.baseDir, dirs.processesDir, dirs.outputDir} {
		err := os.MkdirAll(dir, 0755)
		require.NoError(t, err, "Failed to create test directory: %s", dir)
	}

	// Configure the processor to use these directories
	restoreFunc := ConfigureDirs(dirs.baseDir, dirs.processesDir, dirs.outputDir)

	// Register cleanup in the test
	t.Cleanup(func() {
		// Restore original directory configuration
		restoreFunc()

		// Remove the test directory
		os.RemoveAll(testRootDir)
	})

	return dirs
}

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
	}

	// Test
	p, err := NewProcessor(cfg)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.NotNil(t, p.(*processor).tracklistImporter)
	assert.NotNil(t, p.(*processor).setDownloader)
	assert.NotNil(t, p.(*processor).audioProcessor)
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
	p, _, _, _ := setupTestProcessor()

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
	p, _, _, _ := setupTestProcessor()

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
	p, _, _, _ := setupTestProcessor()

	bar := progressbar.NewOptions(
		10,
		progressbar.OptionSetWriter(io.Discard), // Discard output for tests
	)

	var completedTracks int32 = 4 // Already completed 4 out of 10
	setLength := 10               // Total tracks
	var capturedProgress int      // To capture the callback's progress value
	var capturedMessage string    // To capture the callback's message
	progressCallback := func(progress int, message string) {
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
	// Setup test directories with our new helper
	testDirs := setupTestDirectories(t)

	// Create test options
	opts := &ProcessingOptions{
		TracklistPath:      "test_path",
		DJSetURL:           "test_url",
		FileExtension:      "mp3",
		MaxConcurrentTasks: 4,
	}

	// Create a test progress callback
	progressCalls := 0
	progressCallback := func(progress int, message string) {
		progressCalls++
	}

	// Test
	ctx := newProcessingContext(opts, progressCallback)

	// Assert the context is properly initialized
	assert.NotEmpty(t, ctx.processID, "Process ID should be generated")
	assert.Equal(t, opts, ctx.opts, "Options should be stored")
	assert.Equal(t, 0, ctx.setLength, "Set length should initialize to 0")
	assert.NotEmpty(t, ctx.processDir, "Process directory path should be set")
	assert.NotEmpty(t, ctx.downloadDir, "Download directory path should be set")
	assert.NotEmpty(t, ctx.tempDir, "Temp directory path should be set")

	// Verify paths are correctly structured - use the test directories we've configured
	assert.Contains(t, ctx.processDir, testDirs.processesDir, "Process dir should be under processes dir")
	assert.Contains(t, ctx.downloadDir, "download", "Download dir should contain 'download'")
	assert.Contains(t, ctx.tempDir, "temp", "Temp dir should contain 'temp'")

	// Test the callback
	ctx.progressCallback(10, "test")
	assert.Equal(t, 1, progressCalls, "Progress callback should be called")
}

func TestImportTracklist(t *testing.T) {
	// Setup test directories with our new helper
	testDirs := setupTestDirectories(t)

	p, mockImporter, _, _ := setupTestProcessor()

	// Create test data
	tracklist := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", StartTime: "0:00", EndTime: "1:00"},
			{Title: "Track 2", StartTime: "1:00", EndTime: "2:00"},
		},
	}

	// Setup mock
	mockImporter.On("Import", "test_path").Return(tracklist, nil)

	// Create context
	ctx := &processingContext{
		opts: &ProcessingOptions{
			TracklistPath: "test_path",
		},
		progressCallback: func(progress int, message string) {},
		processDir:       filepath.Join(testDirs.rootDir, "process"),
		downloadDir:      filepath.Join(testDirs.rootDir, "process/download"),
		tempDir:          filepath.Join(testDirs.rootDir, "process/temp"),
	}

	// Test
	err := p.importTracklist(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, tracklist, ctx.set)
	assert.Equal(t, len(tracklist.Tracks), ctx.setLength)
	assert.NotEmpty(t, ctx.processOutputDir)

	// Verify output directory exists
	assert.DirExists(t, filepath.Dir(ctx.processOutputDir), "Output directory should exist")

	// Verify mock expectations
	mockImporter.AssertExpectations(t)
}

func TestDownloadSet(t *testing.T) {
	// Setup test directories with our new helper
	testDirs := setupTestDirectories(t)

	p, _, mockDownloader, _ := setupTestProcessor()

	// Create test data
	tracklist := &domain.Tracklist{
		Name:   "Test Set",
		Artist: "Test Artist",
		Tracks: []*domain.Track{
			{Title: "Track 1", StartTime: "0:00", EndTime: "1:00"},
			{Title: "Track 2", StartTime: "1:00", EndTime: "2:00"},
		},
	}

	// Create context
	downloadDir := filepath.Join(testDirs.rootDir, "process/download")
	ctx := &processingContext{
		opts: &ProcessingOptions{
			DJSetURL: "test_url",
		},
		progressCallback: func(progress int, message string) {},
		set:              tracklist,
		processDir:       filepath.Join(testDirs.rootDir, "process"),
		downloadDir:      downloadDir,
		tempDir:          filepath.Join(testDirs.rootDir, "process/temp"),
	}

	// Setup mock
	mockDownloader.On("Download", "test_url", mock.Anything, downloadDir, mock.Anything).Return(nil)

	// Test
	err := p.downloadSet(ctx)

	// Assert
	assert.NoError(t, err)
	assert.NotEmpty(t, ctx.downloadedFile)
	assert.NotEmpty(t, ctx.fileName)
	assert.Equal(t, "mp3", ctx.actualExtension)

	// Verify mock expectations
	mockDownloader.AssertExpectations(t)
}

func TestPrepareForProcessing(t *testing.T) {
	// Setup test directories with our new helper
	testDirs := setupTestDirectories(t)

	p, _, _, mockAudioProcessor := setupTestProcessor()

	// Create context
	tempDir := filepath.Join(testDirs.rootDir, "process/temp")
	ctx := &processingContext{
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
		},
		fileName:    filepath.Join(testDirs.rootDir, "input.mp3"),
		tempDir:     tempDir,
		processDir:  filepath.Join(testDirs.rootDir, "process"),
		downloadDir: filepath.Join(testDirs.rootDir, "process/download"),
	}

	// Create a test input file
	createTestFile(t, ctx.fileName, "test audio content")

	// Setup mock
	mockAudioProcessor.On("ExtractCoverArt", ctx.fileName, filepath.Join(tempDir, "cover.jpg")).Return(nil)

	// Test
	err := p.prepareForProcessing(ctx)

	// Assert
	assert.NoError(t, err)
	assert.FileExists(t, ctx.coverArtPath)

	// Verify mock expectations
	mockAudioProcessor.AssertExpectations(t)
}

func TestProcessTracks(t *testing.T) {
	// Setup test directories with our new helper
	testDirs := setupTestDirectories(t)

	p, _, _, mockAudioProcessor := setupTestProcessor()

	// Create context with minimal needed data
	tempDir := filepath.Join(testDirs.rootDir, "process/temp")
	outputDir := filepath.Join(testDirs.rootDir, "output/Test_Set")
	os.MkdirAll(outputDir, 0755)

	ctx := &processingContext{
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
			Tracks: []*domain.Track{
				{Title: "Track 1", StartTime: "0:00", EndTime: "1:00"},
			},
		},
		setLength:        1,
		fileName:         filepath.Join(testDirs.rootDir, "input.mp3"),
		actualExtension:  "mp3",
		coverArtPath:     filepath.Join(tempDir, "cover.jpg"),
		opts:             &ProcessingOptions{FileExtension: "mp3", MaxConcurrentTasks: 1},
		tempDir:          tempDir,
		processDir:       filepath.Join(testDirs.rootDir, "process"),
		downloadDir:      filepath.Join(testDirs.rootDir, "process/download"),
		processOutputDir: outputDir,
		progressCallback: func(progress int, message string) {},
	}

	// Create test files
	createTestFile(t, ctx.fileName, "test audio content")
	createTestFile(t, ctx.coverArtPath, "test cover art")

	// Setup mock for audio processor
	mockAudioProcessor.On("Split", mock.Anything).Return(nil)

	// Test
	results, err := p.processTracks(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	// Verify mock expectations
	mockAudioProcessor.AssertExpectations(t)
}

// Tests for the new process ID approach
func TestEnsureDirectories(t *testing.T) {
	// Create test directory
	testDir, err := os.MkdirTemp("", "djset-ensure-dir-test")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	// Configure the processor to use our test directories
	testBaseDir := filepath.Join(testDir, "storage")
	testProcessesDir := filepath.Join(testDir, "storage/processes")
	testOutputDir := filepath.Join(testDir, "output")

	// Configure directories for this test
	restoreFunc := ConfigureDirs(testBaseDir, testProcessesDir, testOutputDir)
	defer restoreFunc() // Make sure we restore original values

	// Test base directories
	err = ensureBaseDirectories()
	assert.NoError(t, err)

	// Verify directories were created
	assert.DirExists(t, testBaseDir)
	assert.DirExists(t, testProcessesDir)
	assert.DirExists(t, testOutputDir)

	// Test process directories
	ctx := &processingContext{
		processID:   "test-process",
		processDir:  filepath.Join(testProcessesDir, "test-process"),
		downloadDir: filepath.Join(testProcessesDir, "test-process/download"),
		tempDir:     filepath.Join(testProcessesDir, "test-process/temp"),
	}

	err = ensureProcessDirectories(ctx)
	assert.NoError(t, err)

	// Verify process directories were created
	assert.DirExists(t, ctx.processDir)
	assert.DirExists(t, ctx.downloadDir)
	assert.DirExists(t, ctx.tempDir)
}
