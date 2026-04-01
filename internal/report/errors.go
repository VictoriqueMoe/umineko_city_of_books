package report

import "errors"

var (
	ErrMissingFields = errors.New("target_type, target_id, and reason are required")
)
