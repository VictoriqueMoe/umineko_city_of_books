package openai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIError_MessageKeepsOnlyTheHumanSentence(t *testing.T) {
	// given
	cases := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{
			name:       "well formed provider body",
			statusCode: 400,
			body:       `{"error": {"message": "The requested model 'gpe' does not exist.", "type": "invalid_request_error", "param": "model", "code": "model_not_found"}}`,
			want:       "The requested model 'gpe' does not exist.",
		},
		{
			name:       "malformed body falls back to the status",
			statusCode: 502,
			body:       `<html><body>Bad Gateway</body></html>`,
			want:       "the provider rejected the request with status 502",
		},
		{
			name:       "empty body falls back to the status",
			statusCode: 500,
			body:       "",
			want:       "the provider rejected the request with status 500",
		},
		{
			name:       "json without an error message falls back to the status",
			statusCode: 404,
			body:       `{"error": {"type": "invalid_request_error"}}`,
			want:       "the provider rejected the request with status 404",
		},
		{
			name:       "blank error message falls back to the status",
			statusCode: 401,
			body:       `{"error": {"message": "   "}}`,
			want:       "the provider rejected the request with status 401",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apiErr := &APIError{StatusCode: tc.statusCode, Body: tc.body}

			// when
			message := apiErr.Message()

			// then
			assert.Equal(t, tc.want, message)
			assert.NotContains(t, message, "https://")
		})
	}
}
