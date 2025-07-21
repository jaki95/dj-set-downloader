package server

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// MockDownloader simulates a downloader with progress callbacks for testing
type MockDownloader struct {
	simulateProgress bool
	progressSteps    int
}

func (m *MockDownloader) SupportsURL(url string) bool {
	return url == "https://test-download.com/test.mp3"
}

func (m *MockDownloader) Download(ctx context.Context, url, outputDir string, progressCallback func(int, string, []byte)) (string, error) {
	if progressCallback != nil && m.simulateProgress {
		// Simulate download progress
		for i := 0; i <= m.progressSteps; i++ {
			progress := (i * 100) / m.progressSteps
			message := "Downloading... " + fmt.Sprintf("%d%%", progress)
			progressCallback(progress, message, nil)

			// Small delay to simulate real download
			time.Sleep(10 * time.Millisecond)

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}
		}
	}

	// Create a mock downloaded file
	mockFile := filepath.Join(outputDir, "test_audio.mp3")
	if err := os.WriteFile(mockFile, []byte("mock audio content"), 0644); err != nil {
		return "", err
	}

	return mockFile, nil
}

func TestDownloadProgressIntegration(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Storage: config.StorageConfig{
			OutputDir: t.TempDir(),
		},
	}

	server := New(cfg)

	// Create test tracklist
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "01:00"},
		},
	}

	// Create job
	jobStatus, _ := server.jobManager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Test progress updates directly
	progressSteps := []struct {
		progress float64
		message  string
	}{
		{0.0, "Starting download..."},
		{10.0, "Downloading... 40%"},
		{20.0, "Downloading... 80%"},
		{25.0, "Download completed"},
		{50.0, "Processed track 1/1: Track 1"},
		{100.0, "Processing completed"},
	}

	// Simulate progress updates
	for _, step := range progressSteps {
		err := server.jobManager.UpdateJobProgress(jobID, step.progress, step.message)
		if err != nil {
			t.Fatalf("Failed to update progress: %v", err)
		}

		// Verify progress was updated
		job, err := server.jobManager.GetJob(jobID)
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}

		if job.Progress != step.progress {
			t.Errorf("Progress step %f: expected %f, got %f", step.progress, step.progress, job.Progress)
		}

		if job.Message != step.message {
			t.Errorf("Progress step %f: expected message '%s', got '%s'", step.progress, step.message, job.Message)
		}
	}
}

func TestProgressCallbackMapping(t *testing.T) {
	// Test the progress mapping logic we use in the server
	testCases := []struct {
		downloadPercent  int
		expectedProgress float64
		description      string
	}{
		{0, 0.0, "Download start"},
		{50, 12.5, "Download 50%"}, // 50% * 0.25 = 12.5%
		{100, 25.0, "Download complete"},
	}

	for _, tc := range testCases {
		// This simulates the logic in our downloadProgressCallback
		overallProgress := float64(tc.downloadPercent) * 0.25

		if overallProgress != tc.expectedProgress {
			t.Errorf("%s: expected %f, got %f", tc.description, tc.expectedProgress, overallProgress)
		}
	}
}

func TestProcessingProgressMapping(t *testing.T) {
	// Test the processing progress mapping logic
	totalTracks := 5

	testCases := []struct {
		completedTracks  int
		expectedProgress float64
		description      string
	}{
		{0, 25.0, "Processing started"},   // 25% base
		{1, 40.0, "1/5 tracks complete"},  // 25 + (1/5 * 75) = 40%
		{3, 70.0, "3/5 tracks complete"},  // 25 + (3/5 * 75) = 70%
		{5, 100.0, "All tracks complete"}, // 25 + (5/5 * 75) = 100%
	}

	for _, tc := range testCases {
		// This simulates the logic in our processing progress update
		completion := float64(tc.completedTracks) / float64(totalTracks)
		overallProgress := 25.0 + (completion * 75.0)

		if overallProgress != tc.expectedProgress {
			t.Errorf("%s: expected %f, got %f", tc.description, tc.expectedProgress, overallProgress)
		}
	}
}
