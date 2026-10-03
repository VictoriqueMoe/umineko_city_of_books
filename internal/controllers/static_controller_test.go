package controllers

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func serveDir(t *testing.T, name, content string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "static")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0644))
	return dir
}

func newUploadsHarness(t *testing.T) (*testutil.Harness, *testutil.LocalStorage) {
	h := testutil.NewHarness(t)
	store := testutil.NewLocalStorage(t)
	s := &Service{StorageService: store.Service}
	for _, setup := range s.getAllUploadRoutes() {
		setup(h.App)
	}

	return h, store
}

func TestStaticController_Uploads_ServesFileAnd404(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	store.Put("posts/pic.txt", "hello", time.Now())
	store.ExpectMissing("posts/missing.txt")

	// when
	okStatus, okBody := h.NewRequest(http.MethodGet, "/uploads/posts/pic.txt").Do()
	missStatus, _ := h.NewRequest(http.MethodGet, "/uploads/posts/missing.txt").Do()
	stagingStatus, _ := h.NewRequest(http.MethodGet, "/uploads/.staging/upload-1/pic.txt").Do()

	// then
	assert.Equal(t, http.StatusOK, okStatus)
	assert.Equal(t, "hello", string(okBody))
	assert.Equal(t, http.StatusNotFound, missStatus)
	assert.Equal(t, http.StatusNotFound, stagingStatus)
}

func TestStaticController_Uploads_Headers(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	modTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	store.Put("posts/pic.webp", "0123456789", modTime)

	// when
	resp := h.NewRequest(http.MethodGet, "/uploads/posts/pic.webp").DoResponse()

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "image/webp", resp.Header.Get("Content-Type"))
	assert.Equal(t, "bytes", resp.Header.Get("Accept-Ranges"))
	assert.Equal(t, modTime.Format(http.TimeFormat), resp.Header.Get("Last-Modified"))
	assert.NotEmpty(t, resp.Header.Get("ETag"))
	assert.Equal(t, "10", resp.Header.Get("Content-Length"))
}

func TestStaticController_Uploads_Ranges(t *testing.T) {
	tests := []struct {
		name         string
		rangeHeader  string
		wantStatus   int
		wantBody     string
		wantContent  string
		wantRangeHdr string
	}{
		{name: "closed range", rangeHeader: "bytes=2-5", wantStatus: http.StatusPartialContent, wantBody: "2345", wantRangeHdr: "bytes 2-5/10"},
		{name: "open ended range", rangeHeader: "bytes=7-", wantStatus: http.StatusPartialContent, wantBody: "789", wantRangeHdr: "bytes 7-9/10"},
		{name: "suffix range", rangeHeader: "bytes=-3", wantStatus: http.StatusPartialContent, wantBody: "789", wantRangeHdr: "bytes 7-9/10"},
		{name: "end past size is clamped", rangeHeader: "bytes=8-100", wantStatus: http.StatusPartialContent, wantBody: "89", wantRangeHdr: "bytes 8-9/10"},
		{name: "start past size", rangeHeader: "bytes=10-", wantStatus: http.StatusRequestedRangeNotSatisfiable, wantRangeHdr: "bytes */10"},
		{name: "multiple ranges serve the whole file", rangeHeader: "bytes=0-1,4-5", wantStatus: http.StatusOK, wantBody: "0123456789"},
		{name: "malformed range is ignored", rangeHeader: "bytes=abc", wantStatus: http.StatusOK, wantBody: "0123456789"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h, store := newUploadsHarness(t)
			store.Put("posts/digits.txt", "0123456789", time.Now())

			// when
			resp := h.NewRequest(http.MethodGet, "/uploads/posts/digits.txt").WithHeader("Range", tc.rangeHeader).DoResponse()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			// then
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
			assert.Equal(t, tc.wantRangeHdr, resp.Header.Get("Content-Range"))
			if tc.wantStatus != http.StatusRequestedRangeNotSatisfiable {
				assert.Equal(t, tc.wantBody, string(body))
			}
		})
	}
}

func TestStaticController_Uploads_ConditionalRequests(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	modTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	store.Put("posts/pic.txt", "hello", modTime)
	first := h.NewRequest(http.MethodGet, "/uploads/posts/pic.txt").DoResponse()
	etag := first.Header.Get("ETag")

	tests := []struct {
		name       string
		header     string
		value      string
		wantStatus int
	}{
		{name: "matching etag", header: "If-None-Match", value: etag, wantStatus: http.StatusNotModified},
		{name: "other etag", header: "If-None-Match", value: `W/"nope"`, wantStatus: http.StatusOK},
		{name: "not modified since", header: "If-Modified-Since", value: modTime.Format(http.TimeFormat), wantStatus: http.StatusNotModified},
		{name: "modified since", header: "If-Modified-Since", value: modTime.Add(-time.Hour).Format(http.TimeFormat), wantStatus: http.StatusOK},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			resp := h.NewRequest(http.MethodGet, "/uploads/posts/pic.txt").WithHeader(tc.header, tc.value).DoResponse()

			// then
			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}

func TestStaticController_Uploads_IfRange(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	modTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	store.Put("posts/digits.txt", "0123456789", modTime)
	etag := h.NewRequest(http.MethodGet, "/uploads/posts/digits.txt").DoResponse().Header.Get("ETag")

	tests := []struct {
		name       string
		ifRange    string
		wantStatus int
		wantBody   string
	}{
		{name: "matching date resumes", ifRange: modTime.Format(http.TimeFormat), wantStatus: http.StatusPartialContent, wantBody: "789"},
		{name: "stale date sends everything", ifRange: modTime.Add(-time.Hour).Format(http.TimeFormat), wantStatus: http.StatusOK, wantBody: "0123456789"},
		{name: "weak etag can never validate a range", ifRange: etag, wantStatus: http.StatusOK, wantBody: "0123456789"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			status, body := h.NewRequest(http.MethodGet, "/uploads/posts/digits.txt").WithHeader("Range", "bytes=7-").WithHeader("If-Range", tc.ifRange).Do()

			// then
			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantBody, string(body))
		})
	}
}

func TestStaticController_Uploads_UnconfiguredBackendIs503(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	store.ExpectOnUnconfiguredS3("posts/in-s3.png")

	// when
	status, _ := h.NewRequest(http.MethodGet, "/uploads/posts/in-s3.png").Do()

	// then
	assert.Equal(t, http.StatusServiceUnavailable, status)
}

func TestStaticController_Uploads_HeadSendsNoBody(t *testing.T) {
	// given
	h, store := newUploadsHarness(t)
	store.Put("posts/pic.txt", "hello", time.Now())

	// when
	resp := h.NewRequest(http.MethodHead, "/uploads/posts/pic.txt").DoResponse()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// then
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "5", resp.Header.Get("Content-Length"))
	assert.Empty(t, body)
}

func TestStaticController_HLS_ServesFile(t *testing.T) {
	// given
	h := testutil.NewHarness(t)
	dir := serveDir(t, "stream.m3u8", "#EXTM3U")
	h.SettingsService.EXPECT().Get(mock.Anything, config.SettingStreamHLSOutputDir).Return(dir)
	s := &Service{SettingsService: h.SettingsService}
	for _, setup := range s.getAllHLSRoutes() {
		setup(h.App)
	}

	// when
	status, body := h.NewRequest(http.MethodGet, "/hls/stream.m3u8").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "#EXTM3U")
}

func TestStaticController_SPA_ServesEmbeddedAssetForDottedPath(t *testing.T) {
	// given
	h := testutil.NewHarness(t)
	s := &Service{
		StaticFS: fstest.MapFS{
			"app.js": &fstest.MapFile{Data: []byte("console.log(1)")},
		},
	}
	for _, setup := range s.getAllSPARoutes() {
		setup(h.App)
	}

	// when
	status, body := h.NewRequest(http.MethodGet, "/app.js").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	assert.Contains(t, string(body), "console.log(1)")
}

func TestStaticController_RegistersWebManifestMimeType(t *testing.T) {
	// given
	ext := ".webmanifest"

	// when
	got := mime.TypeByExtension(ext)

	// then
	assert.Equal(t, "application/manifest+json", got)
}
