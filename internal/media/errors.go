package media

import "errors"

var (
	ErrShuttingDown   = errors.New("media processor shutting down")
	errBlockedAddress = errors.New("media: refusing to dial non-public address")
)
