package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandler(t *testing.T) {
	cases := []struct {
		name       string
		body       []byte
		route      fiber.Handler
		wantStatus int
		wantError  string
		wantJSON   bool
	}{
		{
			name:       "a request too large is refused with the limit spelled out",
			body:       []byte("small"),
			route:      func(fiber.Ctx) error { return fiber.ErrRequestEntityTooLarge },
			wantStatus: fiber.StatusRequestEntityTooLarge,
			wantError:  "This is more than the 2 MB one request can carry. Every file is sent together with the post, so remove or shrink some files and try again.",
			wantJSON:   true,
		},
		{
			name:       "a body within the limit reaches the route",
			body:       []byte("small"),
			route:      func(ctx fiber.Ctx) error { return ctx.SendStatus(fiber.StatusCreated) },
			wantStatus: fiber.StatusCreated,
		},
		{
			name:       "any other error keeps fiber's default handling",
			body:       []byte("small"),
			route:      func(fiber.Ctx) error { return fiber.ErrTeapot },
			wantStatus: fiber.StatusTeapot,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := fiber.New(fiber.Config{ErrorHandler: ErrorHandler})
			app.Server().MaxRequestBodySize = 2 * 1024 * 1024
			app.Post("/upload", tc.route)
			req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader(tc.body))

			// when
			resp, err := app.Test(req)

			// then
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if !tc.wantJSON {
				return
			}

			raw, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got map[string]string
			require.NoError(t, json.Unmarshal(raw, &got))
			assert.Equal(t, tc.wantError, got["error"])
		})
	}
}
