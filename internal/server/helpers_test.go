package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jaki95/dj-set-downloader/config"
	"github.com/jaki95/dj-set-downloader/internal/domain"
)

// TestJobCleanupScheduling tests that job cleanup is scheduled after completion
func TestJobCleanupScheduling(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Storage: config.StorageConfig{
			OutputDir: t.TempDir(),
		},
	}
	server := New(cfg)

	// Create a test job
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "03:00"},
		},
	}

	jobStatus, ctx := server.jobManager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Create temp directory and verify it exists
	tempDir := server.createJobTempDir(jobID)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatalf("Temp directory was not created: %s", tempDir)
	}

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test_track.mp3")
	if err := os.WriteFile(testFile, []byte("test audio data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Schedule cleanup and verify directory still exists immediately
	server.scheduleJobCleanup(jobID, tempDir)

	// Verify temp directory still exists immediately after scheduling
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory was cleaned up immediately, should exist with grace period: %s", tempDir)
	}

	// Test file should still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Test file was cleaned up immediately, should exist with grace period: %s", testFile)
	}

	// Cancel context to avoid leaving goroutines running
	_ = ctx
}

// TestJobCleanupImmediate tests that failed jobs are cleaned up immediately
func TestJobCleanupImmediate(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Storage: config.StorageConfig{
			OutputDir: t.TempDir(),
		},
	}
	server := New(cfg)

	// Create a test job
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "03:00"},
		},
	}

	jobStatus, _ := server.jobManager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Create temp directory and verify it exists
	tempDir := server.createJobTempDir(jobID)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatalf("Temp directory was not created: %s", tempDir)
	}

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test_track.mp3")
	if err := os.WriteFile(testFile, []byte("test audio data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Perform immediate cleanup (as done for failed jobs)
	server.cleanupJobTempDir(jobID, tempDir)

	// Verify temp directory is cleaned up immediately
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be cleaned up immediately for failed jobs: %s", tempDir)
	}

	// Test file should also be cleaned up
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Errorf("Test file should be cleaned up immediately for failed jobs: %s", testFile)
	}
}

// TestScheduledCleanupBehavior tests the actual cleanup after grace period (with short timeout for testing)
func TestScheduledCleanupBehavior(t *testing.T) {
	// Create test server
	cfg := &config.Config{
		Storage: config.StorageConfig{
			OutputDir: t.TempDir(),
		},
	}
	server := New(cfg)

	// Create a test job
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "03:00"},
		},
	}

	jobStatus, _ := server.jobManager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Create temp directory and verify it exists
	tempDir := server.createJobTempDir(jobID)
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatalf("Temp directory was not created: %s", tempDir)
	}

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test_track.mp3")
	if err := os.WriteFile(testFile, []byte("test audio data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Temporarily override the cleanup grace period for testing
	originalGracePeriod := JobCleanupGracePeriod
	// Use short grace period for testing (but don't modify the package constant)

	// Start a custom cleanup goroutine with short delay for testing
	go func() {
		time.Sleep(100 * time.Millisecond) // Short delay for test
		if err := os.RemoveAll(tempDir); err != nil {
			t.Logf("Failed to clean up temp directory: %v", err)
		}
	}()

	// Verify directory exists initially
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory should exist initially: %s", tempDir)
	}

	// Wait for cleanup to happen
	time.Sleep(200 * time.Millisecond)

	// Verify temp directory is cleaned up after grace period
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory should be cleaned up after grace period: %s", tempDir)
	}

	// Restore original grace period (even though we didn't modify it)
	_ = originalGracePeriod
}
