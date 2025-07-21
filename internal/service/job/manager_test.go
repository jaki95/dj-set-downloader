package job

import (
	"testing"

	"github.com/jaki95/dj-set-downloader/internal/domain"
)

func TestJobProgressTracking(t *testing.T) {
	manager := NewManager()

	// Create a test tracklist
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "03:00"},
			{Name: "Track 2", StartTime: "03:00", EndTime: "06:00"},
		},
	}

	// Create a job
	jobStatus, _ := manager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Test initial state
	if jobStatus.Progress != 0 {
		t.Errorf("Expected initial progress 0, got %f", jobStatus.Progress)
	}

	// Test download progress update
	err := manager.UpdateJobProgress(jobID, 15.0, "Downloading... 60%")
	if err != nil {
		t.Fatalf("Failed to update job progress: %v", err)
	}

	// Get updated job status
	updatedJob, err := manager.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	// Verify download progress was updated
	if updatedJob.Progress != 15.0 {
		t.Errorf("Expected progress 15.0, got %f", updatedJob.Progress)
	}
	if updatedJob.Message != "Downloading... 60%" {
		t.Errorf("Expected message 'Downloading... 60%%', got %s", updatedJob.Message)
	}

	// Test processing progress update
	err = manager.UpdateJobProgress(jobID, 50.0, "Processed track 1/2: Track 1")
	if err != nil {
		t.Fatalf("Failed to update processing progress: %v", err)
	}

	// Get updated job status
	updatedJob, err = manager.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get updated job: %v", err)
	}

	// Verify processing progress was updated
	if updatedJob.Progress != 50.0 {
		t.Errorf("Expected progress 50.0, got %f", updatedJob.Progress)
	}
	if updatedJob.Message != "Processed track 1/2: Track 1" {
		t.Errorf("Expected processing message, got %s", updatedJob.Message)
	}

	// Test completion
	err = manager.UpdateJobStatus(jobID, StatusCompleted, []string{"track1.mp3", "track2.mp3"}, "Processing completed")
	if err != nil {
		t.Fatalf("Failed to update job to completed: %v", err)
	}

	// Get final job status
	finalJob, err := manager.GetJob(jobID)
	if err != nil {
		t.Fatalf("Failed to get final job: %v", err)
	}

	// Verify completion
	if finalJob.Progress != 100.0 {
		t.Errorf("Expected final progress 100.0, got %f", finalJob.Progress)
	}
	if finalJob.Status != StatusCompleted {
		t.Errorf("Expected status completed, got %s", finalJob.Status)
	}
	if finalJob.EndTime == nil {
		t.Error("Expected end time to be set for completed job")
	}
	if len(finalJob.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(finalJob.Results))
	}
}

func TestJobProgressEdgeCases(t *testing.T) {
	manager := NewManager()

	// Test with non-existent job
	err := manager.UpdateJobProgress("non-existent", 50.0, "Test message")
	if err == nil {
		t.Error("Expected error for non-existent job")
	}

	// Create a real job
	tracklist := domain.Tracklist{
		Artist: "Test Artist",
		Name:   "Test Mix",
		Tracks: []*domain.Track{
			{Name: "Track 1", StartTime: "00:00", EndTime: "03:00"},
		},
	}

	jobStatus, _ := manager.CreateJob(tracklist)
	jobID := jobStatus.ID

	// Test progress boundary values
	testCases := []struct {
		progress float64
		message  string
	}{
		{0.0, "Starting..."},
		{25.5, "Download progress"},
		{75.0, "Processing..."},
		{100.0, "Complete"},
	}

	for _, tc := range testCases {
		err := manager.UpdateJobProgress(jobID, tc.progress, tc.message)
		if err != nil {
			t.Fatalf("Failed to update progress to %f: %v", tc.progress, err)
		}

		job, err := manager.GetJob(jobID)
		if err != nil {
			t.Fatalf("Failed to get job: %v", err)
		}

		if job.Progress != tc.progress {
			t.Errorf("Expected progress %f, got %f", tc.progress, job.Progress)
		}
		if job.Message != tc.message {
			t.Errorf("Expected message '%s', got '%s'", tc.message, job.Message)
		}
	}
}
