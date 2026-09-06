package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRejectPlainTextBodies(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		contentType string
		body        string
		wantStatus  int
	}{
		{"json body passes", "POST", "application/json", `{"username":"a"}`, http.StatusOK},
		{"json body with charset passes", "POST", "application/json; charset=utf-8", `{"username":"a"}`, http.StatusOK},
		{"multipart body passes", "POST", "multipart/form-data; boundary=x", "--x--", http.StatusOK},
		{"empty body passes regardless of type", "POST", "text/plain", "", http.StatusOK},
		{"text/plain body is rejected", "POST", "text/plain", `{"username":"a","password":"b="}`, http.StatusUnsupportedMediaType},
		{"text/plain with charset is rejected", "POST", "Text/Plain; charset=UTF-8", `{"username":"a"}`, http.StatusUnsupportedMediaType},
		{"text/plain on PUT is rejected", "PUT", "text/plain", `{}`, http.StatusUnsupportedMediaType},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := fiber.New()
			app.Use(RejectPlainTextBodies())
			app.All("/target", func(ctx fiber.Ctx) error {
				return ctx.JSON(fiber.Map{"status": "ok"})
			})
			req := httptest.NewRequest(tc.method, "/target", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)

			// when
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			// then
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
