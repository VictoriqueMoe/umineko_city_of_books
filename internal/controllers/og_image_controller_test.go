package controllers

import (
	"net/http"
	"testing"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/og"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOGImage_NotFound(t *testing.T) {
	// given
	store := testutil.NewLocalStorage(t)
	store.ExpectMissing("posts/missing.webp")
	h := testutil.NewHarness(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()
	NewOGImageHandler(store.Service, settingsSvc, og.NewImageService(nil, store.Service)).Register(h.App)

	tests := []struct {
		name string
		path string
	}{
		{name: "non jpg extension", path: "/og-image/posts/file.webp"},
		{name: "missing file", path: "/og-image/posts/missing.jpg"},
		{name: "path traversal", path: "/og-image/..%2Foutside.jpg"},
		{name: "staging area", path: "/og-image/.staging/upload-1/file.jpg"},
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

func TestOGImage_NotFoundSkipsCacheHeaderMiddleware(t *testing.T) {
	// given
	store := testutil.NewLocalStorage(t)
	store.ExpectMissing("posts/missing.webp")
	h := testutil.NewHarness(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels).Maybe()

	stamped := false
	h.App.Use(func(ctx fiber.Ctx) error {
		if err := ctx.Next(); err != nil {
			return err
		}

		stamped = true

		return nil
	})
	NewOGImageHandler(store.Service, settingsSvc, og.NewImageService(nil, store.Service)).Register(h.App)

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
			// given
			stamped = false

			// when
			status, _ := h.NewRequest("GET", tc.path).Do()

			// then
			assert.Equal(t, http.StatusNotFound, status)
			assert.False(t, stamped)
		})
	}
}

func TestOGImage_ConversionFailureServesOriginal(t *testing.T) {
	// given
	store := testutil.NewLocalStorage(t)
	store.Put("posts/broken.webp", "not really a webp", time.Now())
	h := testutil.NewHarness(t)
	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingMaxImagePixels).Return(bounds.FallbackMaxImagePixels)
	NewOGImageHandler(store.Service, settingsSvc, og.NewImageService(nil, store.Service)).Register(h.App)

	// when
	status, body := h.NewRequest("GET", "/og-image/posts/broken.jpg").Do()

	// then
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "not really a webp", string(body))
}
