package storage

import "errors"

var (
	ErrBackendUnavailable = errors.New("storage: backend is not enabled")
	ErrFlushInterrupted   = errors.New("storage: s3 is unavailable, local fallback flush stopped early")
	ErrDeleteRaced        = errors.New("storage: stored files kept changing backend while they were deleted")
)
