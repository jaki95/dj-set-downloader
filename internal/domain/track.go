package domain

// Track represents an individual track in a tracklist.
type Track struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TrackNumber int    `json:"track_number"`

	DownloadURL string `json:"download_url,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Available   bool   `json:"available,omitempty"`
}

// Tracklist represents a collection of tracks in a DJ set.
type Tracklist struct {
	Name   string   `json:"name"`
	Year   int      `json:"year,omitempty"`
	Artist string   `json:"artist"`
	Genre  string   `json:"genre,omitempty"`
	Tracks []*Track `json:"tracks"`
}
