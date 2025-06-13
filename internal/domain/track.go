package domain

type Track struct {
	Name        string `json:"name"`
	Artist      string `json:"artist"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TrackNumber int    `json:"track_number"` // This will be set during processing
}

type Tracklist struct {
	Name   string   `json:"name"`
	Year   int      `json:"year"`
	Artist string   `json:"artist"`
	Genre  string   `json:"genre"`
	Tracks []*Track `json:"tracks"`
}
