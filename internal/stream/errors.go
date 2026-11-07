package stream

import "errors"

var (
	ErrDisabled       = errors.New("streaming is not configured")
	ErrAtCapacity     = errors.New("maximum concurrent streams reached")
	ErrAlreadyLive    = errors.New("you already have an active stream")
	ErrStreamNotFound = errors.New("stream not found")
	ErrNotOwner       = errors.New("not the stream owner")
	ErrTitleRequired  = errors.New("stream title is required")
	ErrInvalidBitrate = errors.New("stream bitrate must be between 500 and 50000 Kbps")
)
