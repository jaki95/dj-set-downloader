package domain

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

type Tracklist struct {
	Name   string   `json:"name"`
	Year   int      `json:"year"`
	Artist string   `json:"artist"`
	Genre  string   `json:"genre"`
	Tracks []*Track `json:"tracks"`
}
