package server

import "errors"

var (
	ErrInvalidTracklist = errors.New("invalid tracklist")
	ErrJobNotFound      = errors.New("job not found")
	ErrInvalidJobState  = errors.New("invalid job state")
)
