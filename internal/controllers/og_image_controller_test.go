package controllers

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/settings"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOGImage_NotFound(t *testing.T) {
	// given
	dir := t.TempDir()
	uploads := filepath.Join(dir, "uploads")
	require.NoError(t, os.MkdirAll(uploads, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "outside.webp"), []byte("x"), 0644))
	h := testutil.NewHarness(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()
	NewOGImageHandler(uploads, settingsSvc, og.NewImageService(nil)).Register(h.App)

	tests := []struct {
		name string
		path string
	}{
		{name: "non jpg extension", path: "/og-image/posts/file.webp"},
		{name: "missing file", path: "/og-image/posts/missing.jpg"},
		{name: "path traversal", path: "/og-image/..%2Foutside.jpg"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			status, _ := h.NewRequest("GET", tc.path).Do()

			// then
			assert.Equal(t, http.StatusNotFound, status)
		})
	}
}
