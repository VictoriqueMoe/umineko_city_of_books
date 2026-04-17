package giphy

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrDisabled = errors.New("giphy integration is not configured")
)

type RateLimitError struct {
	ResetAt time.Time
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("giphy rate limited until %s", e.ResetAt.UTC().Format(time.RFC3339))
}
