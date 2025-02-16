package pkg

type Track struct {
	Artist      string `json:"artist"`
	Title       string `json:"title"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	TrackNumber int    `json:"track_number"`
}

type Tracklist struct {
	Name   string   `json:"name"`
	Artist string   `json:"artist"`
	Tracks []*Track `json:"tracks"`
}
