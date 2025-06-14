package domain

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrackJSONSerialization(t *testing.T) {
	// Create a track with some test data
	track := &Track{
		Artist:      "Test Artist",
		Name:        "Test Title",
		StartTime:   "00:00:00",
		EndTime:     "00:05:00",
		TrackNumber: 1,
	}

	// Serialize to JSON
	data, err := json.Marshal(track)
	assert.NoError(t, err)

	// Check JSON structure
	expected := `{"artist":"Test Artist","name":"Test Title","start_time":"00:00:00","end_time":"00:05:00","track_number":1}`
	assert.JSONEq(t, expected, string(data))

	// Deserialize back
	var newTrack Track
	err = json.Unmarshal(data, &newTrack)
	assert.NoError(t, err)

	// Verify the deserialized data matches the original
	assert.Equal(t, track.Artist, newTrack.Artist)
	assert.Equal(t, track.Name, newTrack.Name)
	assert.Equal(t, track.StartTime, newTrack.StartTime)
	assert.Equal(t, track.EndTime, newTrack.EndTime)
	assert.Equal(t, track.TrackNumber, newTrack.TrackNumber)
}

func TestTracklistJSONSerialization(t *testing.T) {
	// Create a tracklist with some test data
	tracklist := &Tracklist{
		Name:   "Test Set",
		Artist: "Test DJ",
		Tracks: []*Track{
			{
				Artist:      "Artist 1",
				Name:        "Track 1",
				StartTime:   "00:00:00",
				EndTime:     "00:05:00",
				TrackNumber: 1,
			},
			{
				Artist:      "Artist 2",
				Name:        "Track 2",
				StartTime:   "00:05:00",
				EndTime:     "00:10:00",
				TrackNumber: 2,
			},
		},
	}

	// Serialize to JSON
	data, err := json.Marshal(tracklist)
	assert.NoError(t, err)

	// Check JSON structure - the exact string is long, so we'll check key parts
	jsonStr := string(data)
	assert.Contains(t, jsonStr, `"name":"Test Set"`)
	assert.Contains(t, jsonStr, `"artist":"Test DJ"`)
	assert.Contains(t, jsonStr, `"artist":"Artist 1"`)
	assert.Contains(t, jsonStr, `"name":"Track 1"`)
	assert.Contains(t, jsonStr, `"artist":"Artist 2"`)
	assert.Contains(t, jsonStr, `"name":"Track 2"`)

	// Deserialize back
	var newTracklist Tracklist
	err = json.Unmarshal(data, &newTracklist)
	assert.NoError(t, err)

	// Verify the deserialized data matches the original
	assert.Equal(t, tracklist.Name, newTracklist.Name)
	assert.Equal(t, tracklist.Artist, newTracklist.Artist)
	assert.Equal(t, len(tracklist.Tracks), len(newTracklist.Tracks))

	for i, track := range tracklist.Tracks {
		assert.Equal(t, track.Artist, newTracklist.Tracks[i].Artist)
		assert.Equal(t, track.Name, newTracklist.Tracks[i].Name)
		assert.Equal(t, track.StartTime, newTracklist.Tracks[i].StartTime)
		assert.Equal(t, track.EndTime, newTracklist.Tracks[i].EndTime)
		assert.Equal(t, track.TrackNumber, newTracklist.Tracks[i].TrackNumber)
	}
}
