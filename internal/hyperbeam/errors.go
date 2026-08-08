package hyperbeam

import (
	"errors"
	"fmt"
)

var (
	ErrDisabled          = errors.New("hyperbeam integration is not configured")
	ErrMissingAdminToken = errors.New("hyperbeam: this watch party has no stored admin token")
)

type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("hyperbeam api %d: %s", e.StatusCode, e.Body)
}
