package job

import "errors"

var (
	ErrInvalidTracklist = errors.New("invalid tracklist")
	ErrNotFound         = errors.New("job not found")
	ErrInvalidState     = errors.New("invalid job state")
)
