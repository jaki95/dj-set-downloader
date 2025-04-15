package progress

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestProgressTracker(t *testing.T) {
	tracker := NewProgressTracker()

	// Test progress updates
	var receivedEvents []Event
	tracker.AddListener(func(event Event) {
		receivedEvents = append(receivedEvents, event)
	})

	// Send some progress updates
	tracker.UpdateProgress(StageImporting, 50, "Importing...", nil)
	tracker.UpdateProgress(StageImporting, 100, "Import complete", nil)

	// Wait for events to be received
	time.Sleep(100 * time.Millisecond)

	// Verify received events
	if len(receivedEvents) != 2 {
		t.Errorf("Expected 2 events, got %d", len(receivedEvents))
	}

	// Test error handling
	tracker.SetError(context.Canceled)

	// Wait for error event
	time.Sleep(100 * time.Millisecond)

	// Verify error state
	state := tracker.GetCurrentState()
	if state.Stage != StageError {
		t.Errorf("Expected error stage, got %s", state.Stage)
	}
	if state.Error != context.Canceled.Error() {
		t.Errorf("Expected error %v, got %s", context.Canceled, state.Error)
	}
}

func TestTrackProgress(t *testing.T) {
	tracker := NewProgressTracker()

	// Test track progress updates
	var receivedEvents []Event
	tracker.AddListener(func(event Event) {
		receivedEvents = append(receivedEvents, event)
	})

	// Send track progress updates
	tracker.UpdateTrackProgress(1, 10, 1, "Track 1")
	tracker.UpdateTrackProgress(2, 10, 2, "Track 2")

	// Wait for events to be received
	time.Sleep(100 * time.Millisecond)

	// Verify received events
	if len(receivedEvents) != 2 {
		t.Errorf("Expected 2 events, got %d", len(receivedEvents))
	}

	// Verify track details
	for i, event := range receivedEvents {
		if event.TrackDetails == nil {
			t.Errorf("Event %d: Expected track details, got nil", i)
			continue
		}
		if event.TrackDetails.TrackNumber != i+1 {
			t.Errorf("Event %d: Expected track number %d, got %d", i, i+1, event.TrackDetails.TrackNumber)
		}
		if event.TrackDetails.TotalTracks != 10 {
			t.Errorf("Event %d: Expected total tracks 10, got %d", i, event.TrackDetails.TotalTracks)
		}
	}
}

func TestEventJSON(t *testing.T) {
	// Test JSON marshaling/unmarshaling
	event := Event{
		Stage:     StageProcessing,
		Progress:  50.0,
		Message:   "Processing...",
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var unmarshaled Event
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if unmarshaled.Stage != event.Stage {
		t.Errorf("Expected stage %s, got %s", event.Stage, unmarshaled.Stage)
	}
	if unmarshaled.Progress != event.Progress {
		t.Errorf("Expected progress %f, got %f", event.Progress, unmarshaled.Progress)
	}
	if unmarshaled.Message != event.Message {
		t.Errorf("Expected message %s, got %s", event.Message, unmarshaled.Message)
	}
}

func TestListenerManagement(t *testing.T) {
	tracker := NewProgressTracker()

	// Add a listener
	var receivedEvents []Event
	listener := func(event Event) {
		receivedEvents = append(receivedEvents, event)
	}
	tracker.AddListener(listener)

	// Send an event
	tracker.UpdateProgress(StageProcessing, 50, "Test", nil)

	// Verify event was received
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event, got %d", len(receivedEvents))
	}

	// Remove the listener
	tracker.RemoveListener(listener)

	// Send another event
	tracker.UpdateProgress(StageProcessing, 75, "Test 2", nil)

	// Verify no new events were received
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event after removal, got %d", len(receivedEvents))
	}
}
