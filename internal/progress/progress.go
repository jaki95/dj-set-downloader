package progress

import (
	"encoding/json"
	"reflect"
	"sync"
	"time"
)

// Stage represents the current stage of processing
type Stage string

const (
	StageInitializing Stage = "initializing"
	StageImporting    Stage = "importing"
	StageDownloading  Stage = "downloading"
	StageProcessing   Stage = "processing"
	StageComplete     Stage = "complete"
	StageError        Stage = "error"
)

// Event represents a progress event
type Event struct {
	Stage        Stage         `json:"stage"`
	Progress     float64       `json:"progress"`
	Message      string        `json:"message"`
	Data         []byte        `json:"data,omitempty"`
	Timestamp    time.Time     `json:"timestamp"`
	TrackDetails *TrackDetails `json:"trackDetails,omitempty"`
	Error        string        `json:"error,omitempty"`
}

// TrackDetails contains information about the current track being processed
type TrackDetails struct {
	TrackNumber     int    `json:"trackNumber"`
	TotalTracks     int    `json:"totalTracks"`
	CurrentTrack    string `json:"currentTrack"`
	ProcessedTracks int    `json:"processedTracks"`
}

// ProgressTracker manages progress tracking
type ProgressTracker struct {
	mu           sync.RWMutex
	stage        Stage
	progress     float64
	message      string
	trackDetails *TrackDetails
	error        error
	listeners    []func(Event)
}

// NewProgressTracker creates a new ProgressTracker instance
func NewProgressTracker() *ProgressTracker {
	return &ProgressTracker{
		stage:     StageInitializing,
		listeners: make([]func(Event), 0),
	}
}

// AddListener adds a new progress event listener
func (pt *ProgressTracker) AddListener(listener func(Event)) {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.listeners = append(pt.listeners, listener)
}

// RemoveListener removes a progress event listener
func (pt *ProgressTracker) RemoveListener(listener func(Event)) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	listenerPtr := reflect.ValueOf(listener).Pointer()
	for i := range pt.listeners {
		if reflect.ValueOf(pt.listeners[i]).Pointer() == listenerPtr {
			pt.listeners = append(pt.listeners[:i], pt.listeners[i+1:]...)
			break
		}
	}
}

// UpdateProgress updates the progress and notifies all listeners
func (pt *ProgressTracker) UpdateProgress(stage Stage, progress float64, message string, data []byte) {
	pt.mu.Lock()
	pt.stage = stage
	pt.progress = progress
	pt.message = message
	pt.mu.Unlock()

	pt.notifyListeners(Event{
		Stage:     stage,
		Progress:  progress,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// UpdateTrackProgress updates track-specific progress
func (pt *ProgressTracker) UpdateTrackProgress(trackNumber, totalTracks, processedTracks int, currentTrack string) {
	pt.mu.Lock()
	pt.trackDetails = &TrackDetails{
		TrackNumber:     trackNumber,
		TotalTracks:     totalTracks,
		CurrentTrack:    currentTrack,
		ProcessedTracks: processedTracks,
	}
	pt.mu.Unlock()

	pt.notifyListeners(Event{
		Stage:        pt.stage,
		Progress:     pt.progress,
		Message:      pt.message,
		Timestamp:    time.Now(),
		TrackDetails: pt.trackDetails,
	})
}

// SetError sets an error state and notifies all listeners
func (pt *ProgressTracker) SetError(err error) {
	pt.mu.Lock()
	pt.stage = StageError
	pt.error = err
	pt.mu.Unlock()

	pt.notifyListeners(Event{
		Stage:     StageError,
		Progress:  pt.progress,
		Message:   err.Error(),
		Timestamp: time.Now(),
		Error:     err.Error(),
	})
}

// notifyListeners sends an event to all registered listeners
func (pt *ProgressTracker) notifyListeners(event Event) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	for _, listener := range pt.listeners {
		listener(event)
	}
}

// GetCurrentState returns the current progress state
func (pt *ProgressTracker) GetCurrentState() Event {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	return Event{
		Stage:        pt.stage,
		Progress:     pt.progress,
		Message:      pt.message,
		Timestamp:    time.Now(),
		TrackDetails: pt.trackDetails,
		Error:        pt.error.Error(),
	}
}

// MarshalJSON implements json.Marshaler for Event
func (e Event) MarshalJSON() ([]byte, error) {
	type Alias Event
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Alias:     (*Alias)(&e),
	})
}

// UnmarshalJSON implements json.Unmarshaler for Event
func (e *Event) UnmarshalJSON(data []byte) error {
	type Alias Event
	aux := &struct {
		Timestamp string `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t, err := time.Parse(time.RFC3339, aux.Timestamp)
	if err != nil {
		return err
	}
	e.Timestamp = t
	return nil
}
