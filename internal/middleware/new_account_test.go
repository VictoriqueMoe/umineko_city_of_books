package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"umineko_city_of_books/internal/authz"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/settings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newAccountApp(t *testing.T, userID uuid.UUID, restricted bool) *fiber.App {
	t.Helper()

	authzSvc := authz.NewMockService(t)
	authzSvc.EXPECT().IsRestrictedNewAccount(mock.Anything, userID).Return(restricted).Maybe()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().GetInt(mock.Anything, config.SettingNewAccountHours).Return(24).Maybe()

	app := fiber.New()
	app.Post("/upload", func(ctx fiber.Ctx) error {
		ctx.Locals("userID", userID)

		return ctx.Next()
	}, RequireEstablishedAccount(authzSvc, settingsSvc), func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	return app
}

func postWithFile(t *testing.T, app *fiber.App) int {
	t.Helper()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("media", "gore.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("not really an image"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.StatusCode
}

func postWithoutFile(t *testing.T, app *fiber.App) int {
	t.Helper()

	body := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("body", "just text"))
	require.NoError(t, writer.Close())

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	return resp.StatusCode
}

func TestRequireEstablishedAccount_BlocksARestrictedAccountCarryingAFile(t *testing.T) {
	// given a member inside the new account window
	app := newAccountApp(t, uuid.New(), true)

	// when they attach a file
	status := postWithFile(t, app)

	// then the upload never reaches the handler
	assert.Equal(t, fiber.StatusForbidden, status)
}

func TestRequireEstablishedAccount_LetsARestrictedAccountPostTextOnly(t *testing.T) {
	// given the same member, because chat sends text and files through one route
	app := newAccountApp(t, uuid.New(), true)

	// when the request carries no file
	status := postWithoutFile(t, app)

	// then talking is still allowed
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireEstablishedAccount_LetsARestrictedAccountSendAJSONRequest(t *testing.T) {
	// given a request that is not multipart at all
	app := newAccountApp(t, uuid.New(), true)

	req := httptest.NewRequest("POST", "/upload", bytes.NewBufferString(`{"body":"hello"}`))
	req.Header.Set("Content-Type", "application/json")

	// when
	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// then a plain json message must never be mistaken for an upload
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestRequireEstablishedAccount_AllowsAnUnrestrictedAccount(t *testing.T) {
	// given an established member, or staff, which authz decides
	app := newAccountApp(t, uuid.New(), false)

	// when
	status := postWithFile(t, app)

	// then
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireEstablishedAccount_IgnoresAnonymousRequests(t *testing.T) {
	// given no authenticated user, since auth middleware runs first and owns that rejection
	authzSvc := authz.NewMockService(t)
	settingsSvc := settings.NewMockService(t)

	app := fiber.New()
	app.Post("/upload", func(ctx fiber.Ctx) error {
		ctx.Locals("userID", uuid.Nil)

		return ctx.Next()
	}, RequireEstablishedAccount(authzSvc, settingsSvc), func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	// when
	status := postWithFile(t, app)

	// then authz is never consulted
	assert.Equal(t, fiber.StatusOK, status)
}

func TestRequireEstablishedAccount_AllowsEverythingWhenUnwired(t *testing.T) {
	// given a misconfiguration, where refusing every upload site wide would be worse
	app := fiber.New()
	app.Post("/upload", func(ctx fiber.Ctx) error {
		ctx.Locals("userID", uuid.New())

		return ctx.Next()
	}, RequireEstablishedAccount(nil, nil), func(ctx fiber.Ctx) error {
		return ctx.SendString("handler reached")
	})

	// when
	status := postWithFile(t, app)

	// then
	assert.Equal(t, fiber.StatusOK, status)
}
