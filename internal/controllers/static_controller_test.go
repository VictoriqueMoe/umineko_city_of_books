package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/upload"

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

func TestStaticController_Uploads_ServesFileAnd404(t *testing.T) {
	// given
	h := testutil.NewHarness(t)
	dir := serveDir(t, "pic.txt", "hello")
	up := upload.NewMockService(t)
	up.EXPECT().GetUploadDir().Return(dir)
	s := &Service{UploadService: up}
	for _, setup := range s.getAllUploadRoutes() {
		setup(h.App)
	}

	// when
	okStatus, okBody := h.NewRequest(http.MethodGet, "/uploads/pic.txt").Do()
	missStatus, _ := h.NewRequest(http.MethodGet, "/uploads/missing.txt").Do()

	// then
	assert.Equal(t, http.StatusOK, okStatus)
	assert.Equal(t, "hello", string(okBody))
	assert.Equal(t, http.StatusNotFound, missStatus)
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
