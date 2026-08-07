package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type (
	APIError struct {
		StatusCode int
		Body       string
	}

	RateLimitError struct {
		ResetAt time.Time
	}

	errorPayload struct {
		Error errorDetail `json:"error"`
	}

	errorDetail struct {
		Message string `json:"message"`
	}
)

const (
	genericMessageFormat = "the provider rejected the request with status %d"

	noKeyReason             = "No OpenAI API key is saved on the server."
	unreachableReason       = "OpenAI could not be reached from the server."
	rateLimitedReasonFormat = "OpenAI rate limited this server until %s."
	providerReasonFormat    = "OpenAI answered %d: %s"
)

var (
	ErrDisabled    = errors.New("openai integration is not configured")
	ErrRateLimited = errors.New("openai rate limit in effect")
)

func (e *APIError) Error() string {
	return fmt.Sprintf("openai api %d: %s", e.StatusCode, e.Body)
}

func (e *APIError) Message() string {
	var payload errorPayload

	if err := json.Unmarshal([]byte(e.Body), &payload); err != nil {
		return fmt.Sprintf(genericMessageFormat, e.StatusCode)
	}

	message := strings.TrimSpace(payload.Error.Message)
	if message == "" {
		return fmt.Sprintf(genericMessageFormat, e.StatusCode)
	}

	return message
}

func Reason(err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, ErrDisabled) {
		return noKeyReason
	}

	if rateErr, ok := errors.AsType[*RateLimitError](err); ok {
		return fmt.Sprintf(rateLimitedReasonFormat, rateErr.ResetAt.UTC().Format(time.RFC3339))
	}

	if apiErr, ok := errors.AsType[*APIError](err); ok {
		return fmt.Sprintf(providerReasonFormat, apiErr.StatusCode, apiErr.Message())
	}

	return unreachableReason
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("openai rate limited until %s", e.ResetAt.UTC().Format(time.RFC3339))
}

func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}
