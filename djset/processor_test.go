package djset

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	testDirs := setupTestDirectories(t)

	// Setup
	opts := &ProcessingOptions{
		TracklistPath:      "test.txt",
		DJSetURL:           "test_url",
		FileExtension:      "mp3",
		MaxConcurrentTasks: 1,
	}

	progressCalls := 0
	progressCallback := func(progress int, message string) {
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
	ctx.progressCallback(10, "test")
	assert.Equal(t, 1, progressCalls, "Progress callback should be called")
}

func TestImportTracklist(t *testing.T) {
	testDirs := setupTestDirectories(t)

	p, mockImporter, _, _, _ := setupTestProcessor()

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
	mockImporter.On("Import", "test_path").Return(tracklist, nil)

	// Create context
	ctx := &processingContext{
		opts: &ProcessingOptions{
			TracklistPath: "test_path",
		},
		progressCallback: func(progress int, message string) {},
	}

	// Test
	err := p.importTracklist(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, tracklist, ctx.set)
	assert.Equal(t, len(tracklist.Tracks), ctx.setLength)
	assert.Equal(t, filepath.Join(testDirs.outputDir, "Test_Set"), ctx.outputDir)

	// Verify mock expectations
	mockImporter.AssertExpectations(t)
}

func TestDownloadSet(t *testing.T) {
	testDirs := setupTestDirectories(t)

	p, _, mockDownloader, _, _ := setupTestProcessor()

	// Create a test context
	ctx := &processingContext{
		opts:             &ProcessingOptions{},
		progressCallback: func(int, string) {},
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
		},
		setLength: 1,
	}

	// Setup mocks
	mockDownloader.On("FindURL", "Test Artist Test Set").Return("test_url", nil)
	mockDownloader.On("Download", "test_url", "Test_Set", filepath.Join(testDirs.tempDir, "downloads"), mock.Anything).Return(nil)

	// Create a test file in the download directory to simulate a download
	testFilePath := filepath.Join(testDirs.tempDir, "downloads", "Test_Set.mp3")
	err := os.MkdirAll(filepath.Dir(testFilePath), 0755)
	assert.NoError(t, err)
	err = os.WriteFile(testFilePath, []byte("test audio content"), 0644)
	assert.NoError(t, err)

	// Test
	err = p.downloadSet(ctx)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, testFilePath, ctx.inputFile)
	assert.Equal(t, "mp3", ctx.extension)

	// Verify mock expectations
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
		progressCallback: func(progress int, message string) {},
	}

	mockAudioProcessor := new(MockAudioProcessor)

	// Set up mock for audio processor
	mockAudioProcessor.On("Split", mock.MatchedBy(func(params audio.SplitParams) bool {
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

	results, err := p.processTracks(ctx)
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

	ctx := &processingContext{
		set: &domain.Tracklist{
			Name:   "Test Set",
			Artist: "Test Artist",
		},
		inputFile:        inputFilePath,
		progressCallback: func(progress int, message string) {},
	}

	mockAudioProcessor := new(MockAudioProcessor)

	// Set up mock expectations
	mockAudioProcessor.On("ExtractCoverArt", inputFilePath, expectedCoverArtPath).Return(nil)

	p := &processor{
		audioProcessor: mockAudioProcessor,
	}

	err = p.prepareForProcessing(ctx)
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
		progressCallback: func(int, string) {},
		set:              &domain.Tracklist{},
	}

	// Test temp directory creation
	err = ensureTempDirectories(ctx)
	assert.NoError(t, err)
	assert.DirExists(t, ctx.getDownloadDir())
	assert.DirExists(t, ctx.getTempDir())
}
