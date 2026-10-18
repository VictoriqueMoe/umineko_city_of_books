package utils

import (
	"io"
	"mime"
	"mime/multipart"
	"strconv"
	"strings"

	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

const (
	PayloadField        = "payload"
	MediaField          = "media"
	SpoilerIndexesField = "spoiler_indexes"
)

type (
	MediaRequest[T any] struct {
		Payload T
		Media   []upload.FileUpload
		Files   map[string][]upload.FileUpload
	}
)

func BindWithMedia[T any](ctx fiber.Ctx, fileFields ...string) (MediaRequest[T], bool) {
	var req MediaRequest[T]

	if !isMultipart(ctx) {
		payload, ok := BindJSON[T](ctx)
		req.Payload = payload

		return req, ok
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		_ = BadRequest(ctx, "invalid form data")
		return req, false
	}

	if err := decodePayload(ctx, FormValue(form, PayloadField), &req.Payload); err != nil {
		_ = BadRequest(ctx, "invalid request body")
		return req, false
	}

	spoilers, ok := SpoilerIndexes(form)
	if !ok {
		_ = BadRequest(ctx, "invalid spoiler_indexes")
		return req, false
	}

	req.Media = FormFiles(form, MediaField, spoilers)

	if len(fileFields) > 0 {
		req.Files = make(map[string][]upload.FileUpload, len(fileFields))
		for _, field := range fileFields {
			req.Files[field] = FormFiles(form, field, nil)
		}
	}

	return req, true
}

func FormFiles(form *multipart.Form, field string, spoilers map[int]struct{}) []upload.FileUpload {
	if form == nil {
		return nil
	}

	headers := form.File[field]
	if len(headers) == 0 {
		return nil
	}

	uploads := make([]upload.FileUpload, 0, len(headers))
	for i, h := range headers {
		_, isSpoiler := spoilers[i]

		uploads = append(uploads, upload.FileUpload{
			ContentType: h.Header.Get(fiber.HeaderContentType),
			Filename:    h.Filename,
			Size:        h.Size,
			IsSpoiler:   isSpoiler,
			Open: func() (io.ReadCloser, error) {
				return h.Open()
			},
		})
	}

	return uploads
}

func SpoilerIndexes(form *multipart.Form) (map[int]struct{}, bool) {
	raw := FormValue(form, SpoilerIndexesField)
	if raw == "" {
		return nil, true
	}

	indexes := make(map[int]struct{})
	for token := range strings.SplitSeq(raw, ",") {
		i, err := strconv.Atoi(strings.TrimSpace(token))
		if err != nil || i < 0 {
			return nil, false
		}

		indexes[i] = struct{}{}
	}

	return indexes, true
}

func FormValue(form *multipart.Form, key string) string {
	if form == nil {
		return ""
	}

	values := form.Value[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func isMultipart(ctx fiber.Ctx) bool {
	mediaType, _, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil {
		return false
	}

	return mediaType == fiber.MIMEMultipartForm
}

func decodePayload(ctx fiber.Ctx, raw string, out any) error {
	cfg := ctx.App().Config()

	if err := cfg.JSONDecoder([]byte(raw), out); err != nil {
		return err
	}

	if cfg.StructValidator == nil {
		return nil
	}

	return cfg.StructValidator.Validate(out)
}
