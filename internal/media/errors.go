package media

import "errors"

var (
	ErrShuttingDown  = errors.New("media processor shutting down")
	errMalformedExif = errors.New("media: malformed exif data")
)
