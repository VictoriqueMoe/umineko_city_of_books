package utils

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	receiveTestTimeout = 10 * time.Second
	fileField          = "file"
)

type (
	formPart struct {
		field       string
		filename    string
		contentType string
		data        string
	}
)

func newStaging(t *testing.T) upload.Service {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(t.TempDir()).Maybe()

	return upload.NewService(settingsSvc, nil, nil, nil, nil)
}

func multipartBody(t *testing.T, values map[string]string, parts []formPart) (string, string) {
	t.Helper()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	for name, value := range values {
		require.NoError(t, w.WriteField(name, value))
	}

	for _, p := range parts {
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Disposition": {`form-data; name="` + p.field + `"; filename="` + p.filename + `"`},
			"Content-Type":        {p.contentType},
		})
		require.NoError(t, err)

		_, err = part.Write([]byte(p.data))
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	return buf.String(), w.FormDataContentType()
}

func receive(t *testing.T, staging upload.Service, limit int64, restrictedHours int, contentType string, body string) (*Upload, error) {
	t.Helper()

	var (
		received *Upload
		err      error
	)

	app := fiber.New()
	app.Post("/", func(ctx fiber.Ctx) error {
		if restrictedHours > 0 {
			ctx.Locals(middleware.RestrictedUploadHoursLocal, restrictedHours)
		}

		received, err = ReceiveUpload(ctx, staging, limit)

		return ctx.SendStatus(fiber.StatusOK)
	})

	httpReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	httpReq.Header.Set(fiber.HeaderContentType, contentType)

	resp, testErr := app.Test(httpReq, fiber.TestConfig{Timeout: receiveTestTimeout, FailOnTimeout: true})
	require.NoError(t, testErr)
	require.NoError(t, resp.Body.Close())

	return received, err
}

func TestReceiveUpload_StagesEachFileAndCloseRemovesThem(t *testing.T) {
	// given an upload carrying one file
	staging := newStaging(t)
	body, contentType := multipartBody(t, nil, []formPart{{field: fileField, filename: "one.png", contentType: "image/png", data: "first"}})

	// when it is received
	received, err := receive(t, staging, int64(len(body)), 0, contentType, body)

	// then the file sits in the shared staging area until the request lets it go
	require.NoError(t, err)
	file, ok := received.File(fileField)
	require.True(t, ok)
	assert.Equal(t, int64(len("first")), file.Size)
	assert.FileExists(t, file.Path)

	received.Close()
	assert.NoFileExists(t, file.Path)
}

func TestReceiveUpload_RefusesBeforeStagingAnything(t *testing.T) {
	tests := []struct {
		name            string
		filename        string
		restrictedHours int
		wantErr         error
	}{
		{name: "a file from a restricted new account is refused at its first part", filename: "one.png", restrictedHours: 24, wantErr: ErrNewAccountUpload},
		{name: "a file named dot cannot become a staging path", filename: ".", wantErr: ErrNotMultipart},
		{name: "a file named dot dot cannot become a staging path", filename: "..", wantErr: ErrNotMultipart},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given a strict staging mock that fails the test if anything is staged
			staging := upload.NewMockService(t)
			body, contentType := multipartBody(t, nil, []formPart{{field: fileField, filename: tt.filename, contentType: "image/png", data: "x"}})

			// when
			received, err := receive(t, staging, 1<<10, tt.restrictedHours, contentType, body)

			// then
			require.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, received)
		})
	}
}

func TestReceiveUpload_AStagingFailureIsAServerFaultNotBadInput(t *testing.T) {
	// given a staging area whose disk is full
	diskFull := errors.New("no space left on device")
	staging := upload.NewMockService(t)
	staging.EXPECT().Stage("one.png", mock.Anything).Return("", diskFull).Once()
	body, contentType := multipartBody(t, nil, []formPart{{field: fileField, filename: "one.png", contentType: "image/png", data: "x"}})

	// when
	_, err := receive(t, staging, 1<<10, 0, contentType, body)

	// then the cause survives and it is not reported as a malformed form
	require.ErrorIs(t, err, diskFull)
	assert.NotErrorIs(t, err, ErrNotMultipart)
}

func TestReceiveUpload_Refusals(t *testing.T) {
	fileBody, fileType := multipartBody(t, nil, []formPart{{field: fileField, filename: "one.png", contentType: "image/png", data: strings.Repeat("x", 64)}})
	textBody, textType := multipartBody(t, map[string]string{"body": "just text"}, nil)

	tests := []struct {
		name            string
		body            string
		contentType     string
		limit           int64
		restrictedHours int
		wantErr         error
	}{
		{name: "a body larger than the limit", body: fileBody, contentType: fileType, limit: 16, wantErr: ErrUploadTooLarge},
		{name: "a request that is not multipart", body: `{"a":1}`, contentType: fiber.MIMEApplicationJSON, limit: 1 << 10, wantErr: ErrNotMultipart},
		{name: "a file from a restricted new account", body: fileBody, contentType: fileType, limit: 1 << 10, restrictedHours: 24, wantErr: ErrNewAccountUpload},
		{name: "text only from a restricted new account", body: textBody, contentType: textType, limit: 1 << 10, restrictedHours: 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			staging := newStaging(t)

			// when
			received, err := receive(t, staging, tt.limit, tt.restrictedHours, tt.contentType, tt.body)

			// then
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				assert.Nil(t, received)
			}
		})
	}
}
