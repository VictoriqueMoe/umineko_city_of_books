package controllers

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/controllers/utils/testutil"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newOGImageHarness(t *testing.T) (*testutil.Harness, *upload.MockService, *settings.MockService) {
	h := testutil.NewHarness(t)
	us := upload.NewMockService(t)
	ss := settings.NewMockService(t)

	s := &Service{
		UploadService:   us,
		SettingsService: ss,
		AuthSession:     h.SessionManager,
		AuthzService:    h.AuthzService,
	}
	s.setupAdminUploadOGImage(h.App)
	return h, us, ss
}

func ogImageFactory(t *testing.T) (*testutil.Harness, *upload.MockService) {
	h, us, _ := newOGImageHarness(t)
	return h, us
}

func multipartImageBody(t *testing.T, imageBytes []byte) (string, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("image", "test.jpg")
	require.NoError(t, err)
	_, err = part.Write(imageBytes)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.String(), w.FormDataContentType()
}

func encodeJPEG(t *testing.T) []byte {
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8)), nil))
	return buf.Bytes()
}

func encodePNG(t *testing.T) []byte {
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	return buf.Bytes()
}

func TestAdminUploadOGImage_PermissionFailures(t *testing.T) {
	testutil.RunPermissionFailureSuite(t, ogImageFactory, "POST", "/admin/settings/og-image", nil, authz.PermManageSettings)
}

func TestAdminUploadOGImage_OK(t *testing.T) {
	// given
	h, us, ss := newOGImageHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	ss.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(10 * 1024 * 1024)
	us.EXPECT().SaveFile("branding", mock.MatchedBy(func(name string) bool {
		return len(name) > 0
	}), mock.Anything).Return("/uploads/branding/og_default_123.jpg", nil)
	body, contentType := multipartImageBody(t, encodeJPEG(t))

	// when
	status, respBody := h.NewRequest("POST", "/admin/settings/og-image").WithCookie("valid-cookie").WithRawBody(body, contentType).Do()

	// then
	require.Equal(t, http.StatusOK, status)
	got := testutil.UnmarshalJSON[map[string]string](t, respBody)
	assert.Equal(t, "/uploads/branding/og_default_123.jpg", got["url"])
}

func TestAdminUploadOGImage_RejectsNonJPEG(t *testing.T) {
	// given
	h, _, ss := newOGImageHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)
	ss.EXPECT().GetInt(mock.Anything, config.SettingMaxImageSize).Return(10 * 1024 * 1024)
	body, contentType := multipartImageBody(t, encodePNG(t))

	// when
	status, _ := h.NewRequest("POST", "/admin/settings/og-image").WithCookie("valid-cookie").WithRawBody(body, contentType).Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}

func TestAdminUploadOGImage_MissingFile(t *testing.T) {
	// given
	h, _, _ := newOGImageHarness(t)
	userID := uuid.New()
	h.ExpectValidSession("valid-cookie", userID)
	h.ExpectHasPermission(userID, authz.PermManageSettings, true)

	// when
	status, _ := h.NewRequest("POST", "/admin/settings/og-image").WithCookie("valid-cookie").Do()

	// then
	require.Equal(t, http.StatusBadRequest, status)
}
