package djset

type DJSet interface {
	ProcessTracks(opts *ProcessingOptions, progressCallback func(int, string)) ([]string, error)
}
