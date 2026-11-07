package utils

import (
	"strconv"
	"strings"

	"umineko_city_of_books/internal/upload"
	kernel "umineko_city_of_books/internal/utils"

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
		upload  *Upload
	}
)

func BindWithMedia[T any](ctx fiber.Ctx, staging upload.Service, fileFields ...string) (MediaRequest[T], bool) {
	var req MediaRequest[T]

	if !kernel.IsMultipart(ctx) {
		payload, ok := BindJSON[T](ctx)
		req.Payload = payload

		return req, ok
	}

	received, err := ReceiveUpload(ctx, staging, int64(ctx.App().Server().MaxRequestBodySize))
	if err != nil {
		_ = UploadFailed(ctx, err)
		return req, false
	}
	req.upload = received

	if err := decodePayload(ctx, received.Value(PayloadField), &req.Payload); err != nil {
		received.Close()
		_ = BadRequest(ctx, "invalid request body")
		return req, false
	}

	spoilers, ok := SpoilerIndexes(received.Value(SpoilerIndexesField))
	if !ok {
		received.Close()
		_ = BadRequest(ctx, "invalid spoiler_indexes")
		return req, false
	}

	req.Media = received.FileUploads(MediaField, spoilers)

	if len(fileFields) > 0 {
		req.Files = make(map[string][]upload.FileUpload, len(fileFields))
		for _, field := range fileFields {
			req.Files[field] = received.FileUploads(field, nil)
		}
	}

	return req, true
}

func (r MediaRequest[T]) Close() {
	if r.upload != nil {
		r.upload.Close()
	}
}

func SpoilerIndexes(raw string) (map[int]struct{}, bool) {
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
