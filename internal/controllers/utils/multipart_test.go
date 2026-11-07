package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	bindTestTimeout = 10 * time.Second
)

type (
	bindPayload struct {
		Body     string `json:"body"`
		ParentID string `json:"parent_id"`
	}

	formPart struct {
		field       string
		filename    string
		contentType string
		data        string
	}

	boundFile struct {
		filename    string
		contentType string
		isSpoiler   bool
		data        string
	}

	bindResult struct {
		ok          bool
		payload     bindPayload
		media       []boundFile
		attachments []boundFile
	}
)

func readBoundFiles(files []upload.FileUpload) ([]boundFile, error) {
	out := make([]boundFile, 0, len(files))
	for _, f := range files {
		r, err := f.Open()
		if err != nil {
			return nil, err
		}

		data, err := io.ReadAll(r)
		closeErr := r.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}

		out = append(out, boundFile{filename: f.Filename, contentType: f.ContentType, isSpoiler: f.IsSpoiler, data: string(data)})
	}

	return out, nil
}

func newStaging(t *testing.T) upload.Service {
	t.Helper()

	settingsSvc := settings.NewMockService(t)
	settingsSvc.EXPECT().Get(mock.Anything, config.SettingUploadDir).Return(t.TempDir()).Maybe()

	return upload.NewService(settingsSvc, nil, nil)
}

func runBind(t *testing.T, contentType string, body string) (*http.Response, string, bindResult) {
	t.Helper()

	var got bindResult

	staging := newStaging(t)
	app := fiber.New()
	app.Post("/", func(ctx fiber.Ctx) error {
		req, ok := BindWithMedia[bindPayload](ctx, staging, "attachments")
		got.ok = ok
		if !ok {
			return nil
		}
		defer req.Close()

		got.payload = req.Payload

		mediaFiles, err := readBoundFiles(req.Media)
		if err != nil {
			return err
		}
		got.media = mediaFiles

		attachments, err := readBoundFiles(req.Files["attachments"])
		if err != nil {
			return err
		}
		got.attachments = attachments

		return ctx.SendStatus(fiber.StatusCreated)
	})

	httpReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	httpReq.Header.Set(fiber.HeaderContentType, contentType)

	resp, err := app.Test(httpReq, fiber.TestConfig{Timeout: bindTestTimeout, FailOnTimeout: true})
	require.NoError(t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(data), got
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

func mustJSON(t *testing.T, v any) string {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err)

	return string(data)
}

func TestBindWithMedia_JSONBodyBindsThePayloadWithNoFiles(t *testing.T) {
	// given an old client sending the plain json create body
	body := mustJSON(t, bindPayload{Body: "the culprit is the butler", ParentID: "p1"})

	// when it is bound
	resp, _, got := runBind(t, fiber.MIMEApplicationJSON, body)

	// then the payload is decoded and there are no files
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.True(t, got.ok)
	assert.Equal(t, bindPayload{Body: "the culprit is the butler", ParentID: "p1"}, got.payload)
	assert.Empty(t, got.media)
	assert.Empty(t, got.attachments)
}

func TestBindWithMedia_MultipartBindsPayloadMediaAndNamedFiles(t *testing.T) {
	// given a multipart create carrying the payload, two media files with the second spoilered, and an attachment
	body, contentType := multipartBody(t,
		map[string]string{
			PayloadField:        mustJSON(t, bindPayload{Body: "see attached", ParentID: "p2"}),
			SpoilerIndexesField: "1",
		},
		[]formPart{
			{field: MediaField, filename: "one.png", contentType: "image/png", data: "first"},
			{field: MediaField, filename: "two.webm", contentType: "video/webm", data: "second"},
			{field: "attachments", filename: "notes.pdf", contentType: "application/pdf", data: "third"},
		},
	)

	// when it is bound
	resp, _, got := runBind(t, contentType, body)

	// then every part lands where the contract says, in order
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.True(t, got.ok)
	assert.Equal(t, bindPayload{Body: "see attached", ParentID: "p2"}, got.payload)
	assert.Equal(t, []boundFile{
		{filename: "one.png", contentType: "image/png", data: "first"},
		{filename: "two.webm", contentType: "video/webm", isSpoiler: true, data: "second"},
	}, got.media)
	assert.Equal(t, []boundFile{
		{filename: "notes.pdf", contentType: "application/pdf", data: "third"},
	}, got.attachments)
}

func TestBindWithMedia_Rejections(t *testing.T) {
	validPayload := mustJSON(t, bindPayload{Body: "hello"})

	tests := []struct {
		name       string
		jsonBody   string
		formValues map[string]string
		wantError  string
	}{
		{
			name:      "a malformed json body",
			jsonBody:  "{not json",
			wantError: "invalid request body",
		},
		{
			name:       "a malformed multipart payload",
			formValues: map[string]string{PayloadField: "{not json"},
			wantError:  "invalid request body",
		},
		{
			name:       "a multipart request with no payload",
			formValues: map[string]string{SpoilerIndexesField: "0"},
			wantError:  "invalid request body",
		},
		{
			name:       "a non-numeric spoiler index",
			formValues: map[string]string{PayloadField: validPayload, SpoilerIndexesField: "0,beatrice"},
			wantError:  "invalid spoiler_indexes",
		},
		{
			name:       "a negative spoiler index",
			formValues: map[string]string{PayloadField: validPayload, SpoilerIndexesField: "-1"},
			wantError:  "invalid spoiler_indexes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			body, contentType := tt.jsonBody, fiber.MIMEApplicationJSON
			if tt.formValues != nil {
				body, contentType = multipartBody(t, tt.formValues, []formPart{
					{field: MediaField, filename: "one.png", contentType: "image/png", data: "first"},
				})
			}

			// when
			resp, respBody, got := runBind(t, contentType, body)

			// then
			assert.False(t, got.ok)
			assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
			assert.Contains(t, respBody, tt.wantError)
		})
	}
}

func TestSpoilerIndexes(t *testing.T) {
	tests := []struct {
		name   string
		raw    string
		want   map[int]struct{}
		wantOK bool
	}{
		{"an empty field means no spoilers", "", nil, true},
		{"a single index", "0", map[int]struct{}{0: {}}, true},
		{"several indexes", "0,2", map[int]struct{}{0: {}, 2: {}}, true},
		{"surrounding space is tolerated", " 1 , 3 ", map[int]struct{}{1: {}, 3: {}}, true},
		{"a negative index is refused", "-1", nil, false},
		{"a non-numeric token is refused", "0,beatrice", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given the sender's spoiler choices as sent in the form

			// when they are parsed
			got, ok := SpoilerIndexes(tc.raw)

			// then only a well-formed list is accepted
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFileUploads_MarksOnlyTheChosenFiles(t *testing.T) {
	// given two attachments where only the second was marked
	received := &Upload{files: map[string][]ReceivedFile{
		MediaField: {
			{Filename: "one.png"},
			{Filename: "two.png"},
		},
	}}

	// when they are collected
	uploads := received.FileUploads(MediaField, map[int]struct{}{1: {}})

	// then the flag lands on the file the sender picked, and the filename is carried
	require.Len(t, uploads, 2)
	assert.False(t, uploads[0].IsSpoiler)
	assert.True(t, uploads[1].IsSpoiler)
	assert.Equal(t, "one.png", uploads[0].Filename)
	assert.Equal(t, "two.png", uploads[1].Filename)
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

	resp, testErr := app.Test(httpReq, fiber.TestConfig{Timeout: bindTestTimeout, FailOnTimeout: true})
	require.NoError(t, testErr)
	require.NoError(t, resp.Body.Close())

	return received, err
}

func TestReceiveUpload_StagesEachFileAndCloseRemovesThem(t *testing.T) {
	// given an upload carrying one file
	staging := newStaging(t)
	body, contentType := multipartBody(t, nil, []formPart{{field: MediaField, filename: "one.png", contentType: "image/png", data: "first"}})

	// when it is received
	received, err := receive(t, staging, int64(len(body)), 0, contentType, body)

	// then the file sits in the shared staging area until the request lets it go
	require.NoError(t, err)
	file, ok := received.File(MediaField)
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
			body, contentType := multipartBody(t, nil, []formPart{{field: MediaField, filename: tt.filename, contentType: "image/png", data: "x"}})

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
	body, contentType := multipartBody(t, nil, []formPart{{field: MediaField, filename: "one.png", contentType: "image/png", data: "x"}})

	// when
	_, err := receive(t, staging, 1<<10, 0, contentType, body)

	// then the cause survives and it is not reported as a malformed form
	require.ErrorIs(t, err, diskFull)
	assert.NotErrorIs(t, err, ErrNotMultipart)
}

func TestReceiveUpload_Refusals(t *testing.T) {
	fileBody, fileType := multipartBody(t, nil, []formPart{{field: MediaField, filename: "one.png", contentType: "image/png", data: strings.Repeat("x", 64)}})
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
