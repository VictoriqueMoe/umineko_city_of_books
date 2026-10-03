package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsStreamedPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/uploads/posts/a.webp", want: true},
		{path: "/og-image/posts/a.jpg", want: true},
		{path: "/hls/stream/index.m3u8", want: true},
		{path: "/api/v1/posts", want: false},
		{path: "/uploadsish", want: false},
		{path: "/", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			// given
			app := fiber.New()
			app.Get("/*", func(ctx fiber.Ctx) error {
				return ctx.SendString(strconv.FormatBool(isStreamedPath(ctx)))
			})

			// when
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tc.path, nil))
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// then
			assert.Equal(t, strconv.FormatBool(tc.want), string(body))
		})
	}
}
