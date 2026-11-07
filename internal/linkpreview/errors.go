package linkpreview

import "errors"

var (
	ErrInvalidURL = errors.New("linkpreview: url must be an absolute http or https url")
)
