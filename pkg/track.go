package pkg

type Track struct {
	Artist      string
	Title       string
	StartTime   string
	EndTime     string
	TrackNumber int
}

type Tracklist struct {
	Name   string
	Artist string
	Tracks []*Track
}
