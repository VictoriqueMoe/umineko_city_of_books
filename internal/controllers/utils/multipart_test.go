package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
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

func runBind(t *testing.T, contentType string, body string) (*http.Response, string, bindResult) {
	t.Helper()

	var got bindResult

	app := fiber.New()
	app.Post("/", func(ctx fiber.Ctx) error {
		req, ok := BindWithMedia[bindPayload](ctx, "attachments")
		got.ok = ok
		if !ok {
			return nil
		}

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
		form   *multipart.Form
		want   map[int]struct{}
		wantOK bool
	}{
		{"no form means no spoilers", nil, nil, true},
		{"an absent field means no spoilers", &multipart.Form{Value: map[string][]string{}}, nil, true},
		{"an empty field means no spoilers", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {""}}}, nil, true},
		{"a single index", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0"}}}, map[int]struct{}{0: {}}, true},
		{"several indexes", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0,2"}}}, map[int]struct{}{0: {}, 2: {}}, true},
		{"surrounding space is tolerated", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {" 1 , 3 "}}}, map[int]struct{}{1: {}, 3: {}}, true},
		{"a negative index is refused", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"-1"}}}, nil, false},
		{"a non-numeric token is refused", &multipart.Form{Value: map[string][]string{"spoiler_indexes": {"0,beatrice"}}}, nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given a multipart form carrying the sender's spoiler choices

			// when it is parsed
			got, ok := SpoilerIndexes(tc.form)

			// then only a well-formed list is accepted
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFormFiles_MarksOnlyTheChosenFiles(t *testing.T) {
	// given two attachments where only the second was marked
	form := &multipart.Form{
		File: map[string][]*multipart.FileHeader{
			MediaField: {
				{Filename: "one.png"},
				{Filename: "two.png"},
			},
		},
	}

	// when they are collected
	uploads := FormFiles(form, MediaField, map[int]struct{}{1: {}})

	// then the flag lands on the file the sender picked, and the filename is carried
	require.Len(t, uploads, 2)
	assert.False(t, uploads[0].IsSpoiler)
	assert.True(t, uploads[1].IsSpoiler)
	assert.Equal(t, "one.png", uploads[0].Filename)
	assert.Equal(t, "two.png", uploads[1].Filename)
}
