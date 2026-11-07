package email

import "errors"

var (
	ErrNotConfigured = errors.New("email provider is not configured")
)
