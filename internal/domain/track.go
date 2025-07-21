package domain

// Track represents an individual track in a tracklist.
type Track struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TrackNumber int    `json:"track_number"`
}

// Tracklist represents a collection of tracks in a DJ set.
type Tracklist struct {
	Name   string   `json:"name"`
	Year   int      `json:"year,omitempty"`
	Artist string   `json:"artist"`
	Genre  string   `json:"genre,omitempty"`
	Tracks []*Track `json:"tracks"`
}
