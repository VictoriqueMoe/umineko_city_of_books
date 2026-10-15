package httpclient

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serve(t *testing.T, handler http.HandlerFunc, opts ...Option) (*Client, string) {
	t.Helper()

	srv := httptest.NewTestServer(t, handler)
	transport := srv.Client().Transport

	return New(time.Second, append([]Option{WithTransport(transport)}, opts...)...), srv.URL
}

func TestJSON_DecodesSuccessfulResponse(t *testing.T) {
	// given
	client, target := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"Beatrice"}`)
	})

	// when
	var out struct {
		Name string `json:"name"`
	}
	err := client.JSON(t.Context(), Request{Method: http.MethodGet, URL: target}, &out)

	// then
	require.NoError(t, err)
	assert.Equal(t, "Beatrice", out.Name)
}

func TestJSON_StatusOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		status     int
		body       string
		out        any
		wantStatus int
	}{
		{name: "no content with an out target is not an error", status: http.StatusNoContent, out: &struct{}{}},
		{name: "success without an out target skips decoding", status: http.StatusOK, body: "not json"},
		{name: "client error becomes a status error", status: http.StatusNotFound, body: `{"code":"missing"}`, out: &struct{}{}, wantStatus: http.StatusNotFound},
		{name: "server error becomes a status error", status: http.StatusBadGateway, body: "upstream down", out: &struct{}{}, wantStatus: http.StatusBadGateway},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			client, target := serve(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "30")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})

			// when
			err := client.JSON(t.Context(), Request{Method: http.MethodGet, URL: target}, tc.out)

			// then
			if tc.wantStatus == 0 {
				require.NoError(t, err)
				return
			}
			statusErr, ok := errors.AsType[*StatusError](err)
			require.True(t, ok, "want a *StatusError, got %v", err)
			assert.Equal(t, tc.wantStatus, statusErr.StatusCode)
			assert.Equal(t, tc.body, string(statusErr.Body))
			assert.Equal(t, "30", statusErr.Header.Get("Retry-After"))
		})
	}
}

func TestJSON_StatusErrorBodyIsCapped(t *testing.T) {
	// given
	client, target := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, strings.Repeat("x", maxErrorBodyBytes*2))
	})

	// when
	err := client.JSON(t.Context(), Request{Method: http.MethodGet, URL: target}, nil)

	// then
	statusErr, ok := errors.AsType[*StatusError](err)
	require.True(t, ok)
	assert.Len(t, statusErr.Body, maxErrorBodyBytes)
}

func TestJSON_RefusesBodyOverTheLimit(t *testing.T) {
	// given
	client, target := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"name":"`+strings.Repeat("x", 64)+`"}`)
	})

	// when
	var out struct {
		Name string `json:"name"`
	}
	err := client.JSON(t.Context(), Request{Method: http.MethodGet, URL: target, MaxBodyBytes: 16}, &out)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestDo_BuildsTheRequest(t *testing.T) {
	cases := []struct {
		name            string
		req             Request
		wantMethod      string
		wantQuery       url.Values
		wantContentType string
		wantAuth        string
		wantHeader      string
		wantBody        string
	}{
		{
			name:       "query merges into a url that already has one",
			req:        Request{Method: http.MethodGet, Query: url.Values{"limit": {"24"}}},
			wantMethod: http.MethodGet,
			wantQuery:  url.Values{"existing": {"1"}, "limit": {"24"}},
		},
		{
			name:            "json body sets the content type",
			req:             Request{Method: http.MethodPost, JSON: map[string]string{"to": "battler"}},
			wantMethod:      http.MethodPost,
			wantQuery:       url.Values{"existing": {"1"}},
			wantContentType: "application/json",
			wantBody:        `{"to":"battler"}`,
		},
		{
			name:            "form body sets the content type",
			req:             Request{Method: http.MethodPost, Form: url.Values{"secret": {"s"}, "response": {"r"}}},
			wantMethod:      http.MethodPost,
			wantQuery:       url.Values{"existing": {"1"}},
			wantContentType: "application/x-www-form-urlencoded",
			wantBody:        "response=r&secret=s",
		},
		{
			name:       "bearer and extra headers are sent",
			req:        Request{Method: http.MethodDelete, Bearer: "token-123", Header: http.Header{"User-Agent": {"UminekoCityOfBooks/1.0"}}},
			wantMethod: http.MethodDelete,
			wantQuery:  url.Values{"existing": {"1"}},
			wantAuth:   "Bearer token-123",
			wantHeader: "UminekoCityOfBooks/1.0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var got *http.Request
			var gotBody string
			client, target := serve(t, func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				got = r
				gotBody = string(raw)
				w.WriteHeader(http.StatusNoContent)
			})
			tc.req.URL = target + "/path?existing=1"

			// when
			resp, err := client.Do(t.Context(), tc.req)

			// then
			require.NoError(t, err)
			_ = resp.Body.Close()
			assert.Equal(t, tc.wantMethod, got.Method)
			assert.Equal(t, "/path", got.URL.Path)
			assert.Equal(t, tc.wantQuery, got.URL.Query())
			assert.Equal(t, tc.wantContentType, got.Header.Get("Content-Type"))
			assert.Equal(t, tc.wantAuth, got.Header.Get("Authorization"))
			if tc.wantHeader != "" {
				assert.Equal(t, tc.wantHeader, got.Header.Get("User-Agent"))
			}
			assert.Equal(t, tc.wantBody, gotBody)
		})
	}
}

func TestDo_RejectsJSONAndFormTogether(t *testing.T) {
	// given
	client := New(time.Second)

	// when
	_, err := client.Do(t.Context(), Request{Method: http.MethodPost, URL: "http://example.invalid", JSON: map[string]string{}, Form: url.Values{}})

	// then
	assert.ErrorIs(t, err, ErrBodyConflict)
}

func TestDo_RedactsTheQueryFromTransportErrors(t *testing.T) {
	// given
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target := srv.URL + "/gifs/search"
	srv.Close()
	client := New(time.Second)

	// when
	_, err := client.Do(t.Context(), Request{Method: http.MethodGet, URL: target, Query: url.Values{"api_key": {"top-secret"}}})

	// then
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "top-secret")
	assert.Contains(t, err.Error(), "/gifs/search")
	_, isURLErr := errors.AsType[*url.Error](err)
	assert.True(t, isURLErr)
}

func TestWithMaxRedirects_StopsAtTheLimit(t *testing.T) {
	// given
	hops := 0
	client, target := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, "/next", http.StatusFound)
	}, WithMaxRedirects(3))

	// when
	resp, err := client.Do(t.Context(), Request{Method: http.MethodGet, URL: target})

	// then
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusFound, resp.StatusCode)
	assert.Equal(t, 4, hops)
}

func TestWithTransport_UsesTheGivenTransport(t *testing.T) {
	// given
	transport := &http.Transport{}
	client := New(time.Second, WithTransport(transport))

	// when
	got := client.HTTP().Transport

	// then
	assert.Same(t, transport, got)
}
