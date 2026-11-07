package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
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

func TestLimitRequestBodies(t *testing.T) {
	cases := []struct {
		name          string
		contentType   string
		contentLength int
		wantStatus    int
	}{
		{name: "a body within the limit passes", contentType: "application/json", contentLength: 32, wantStatus: http.StatusOK},
		{name: "a body over the limit is refused", contentType: "application/json", contentLength: 33, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "an upload over the limit is refused", contentType: "multipart/form-data; boundary=x", contentLength: 33, wantStatus: http.StatusRequestEntityTooLarge},
		{name: "a chunked json body of unknown size is refused", contentType: "application/json", contentLength: contentLengthChunked, wantStatus: http.StatusLengthRequired},
		{name: "a chunked upload of unknown size is refused too, so no handler can buffer an endless body", contentType: "multipart/form-data; boundary=x", contentLength: contentLengthChunked, wantStatus: http.StatusLengthRequired},
		{name: "a request without a body passes", contentType: "", contentLength: contentLengthNone, wantStatus: http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			app := fiber.New()
			app.Server().MaxRequestBodySize = 32
			app.Use(LimitRequestBodies())
			app.All("/target", func(ctx fiber.Ctx) error {
				return ctx.SendStatus(http.StatusOK)
			})
			var reqCtx fasthttp.RequestCtx
			reqCtx.Request.Header.SetMethod(http.MethodPost)
			reqCtx.Request.SetRequestURI("/target")
			reqCtx.Request.Header.SetContentType(tc.contentType)
			reqCtx.Request.Header.SetContentLength(tc.contentLength)

			// when
			app.Handler()(&reqCtx)

			// then
			assert.Equal(t, tc.wantStatus, reqCtx.Response.StatusCode())
		})
	}
}
