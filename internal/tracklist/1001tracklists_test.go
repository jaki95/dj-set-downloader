package tracklist

import (
	"context"
	"testing"
	"time"

	"github.com/jaki95/dj-set-downloader/config"
)

func Test1001TracklistsScraping(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode") // Skip this test when -short flag is provided
	}

	cfg := &config.Config{}

	scraper, err := New1001TracklistsScraper(cfg)
	if err != nil {
		t.Fatalf("Failed to create scraper: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tracklist, err := scraper.Scrape(ctx, "https://www.1001tracklists.com/tracklist/2tlrkszk/mathame-afterlife-voyage-012-2018-09-17.html")
	if err != nil {
		t.Fatalf("Failed to scrape tracklist: %v", err)
	}

	// Basic validation
	if tracklist == nil {
		t.Fatal("Tracklist is nil")
	}

	// Verify artist and set name
	expectedArtist := "Mathame"
	// expectedSetName := "Afterlife Voyage 012"
	if tracklist.Artist != expectedArtist {
		t.Errorf("Artist mismatch. Got: %s, Want: %s", tracklist.Artist, expectedArtist)
	}
	// TODO: fix this
	// if tracklist.Name != expectedSetName {
	// 	t.Errorf("Set name mismatch. Got: %s, Want: %s", tracklist.Name, expectedSetName)
	// }

	// Verify number of tracks
	expectedTrackCount := 14
	if len(tracklist.Tracks) != expectedTrackCount {
		t.Errorf("Track count mismatch. Got: %d, Want: %d", len(tracklist.Tracks), expectedTrackCount)
	}

	// Define expected tracks with their exact information
	expectedTracks := []struct {
		number    int
		artist    string
		title     string
		startTime string
		endTime   string
	}{
		{1, "ID", "ID", "00:00", "02:16"},
		{2, "Russell Haswell", "Heavy Handed Sunset (Autechre Conformity Version)", "02:16", "07:46"},
		{3, "ID", "ID", "07:46", "10:25"},
		{4, "SCB", "Test Tubes (Mind Against Celestial Dub)", "10:25", "17:31"},
		{5, "Petter", "Some Polyphony", "17:31", "23:11"},
		{6, "ID", "ID", "23:11", "27:49"},
		{7, "Yotam Avni", "Shtok", "27:49", "31:29"},
		{8, "Mathame", "Lost Mermaid", "31:29", "35:54"},
		{9, "Elektrochemie", "You're My Kind", "35:54", "41:32"},
		{10, "Mathame", "Innerspace", "41:32", "46:56"},
		{11, "Vatican Shadow", "Weapons Inspection", "46:56", "49:29"},
		{12, "Prince Of Denmark", "(In The End) The Ghost Ran Out Of Memory (Mind Against Remix)", "49:29", "54:35"},
		{13, "Mathame", "22", "54:35", "59:30"},
		{14, "Mathame", "Farewell", "59:30", ""},
	}

	// Verify each track's information
	for i, expected := range expectedTracks {
		if i >= len(tracklist.Tracks) {
			t.Errorf("Missing track %d: %s - %s", expected.number, expected.artist, expected.title)
			continue
		}

		track := tracklist.Tracks[i]
		if track.TrackNumber != expected.number {
			t.Errorf("Track %d number mismatch. Got: %d, Want: %d", i+1, track.TrackNumber, expected.number)
		}
		if track.Artist != expected.artist {
			t.Errorf("Track %d artist mismatch. Got: %s, Want: %s", i+1, track.Artist, expected.artist)
		}
		if track.Title != expected.title {
			t.Errorf("Track %d title mismatch. Got: %s, Want: %s", i+1, track.Title, expected.title)
		}
		if track.StartTime != expected.startTime {
			t.Errorf("Track %d start time mismatch. Got: %s, Want: %s", i+1, track.StartTime, expected.startTime)
		}
		if track.EndTime != expected.endTime {
			t.Errorf("Track %d end time mismatch. Got: %s, Want: %s", i+1, track.EndTime, expected.endTime)
		}
	}
}
