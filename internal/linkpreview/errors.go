package linkpreview

import "errors"

var (
	ErrInvalidURL = errors.New("linkpreview: url must be an absolute http or https url")
	ErrBusy       = errors.New("linkpreview: every fetch slot is in use")
)
