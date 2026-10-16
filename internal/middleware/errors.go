package middleware

import "errors"

var (
	ErrRateLimitWithoutUser = errors.New("middleware: a per user rate limit must run after authentication")
)
