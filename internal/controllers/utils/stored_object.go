package utils

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/storage"
	"umineko_city_of_books/internal/storage/engine"

	"github.com/gofiber/fiber/v3"
)

func StoredObjectError(ctx fiber.Ctx, err error) error {
	if errors.Is(err, engine.ErrNotFound) || errors.Is(err, engine.ErrInvalidKey) {
		return fiber.ErrNotFound
	}

	if errors.Is(err, storage.ErrBackendUnavailable) || errors.Is(err, engine.ErrDisabled) {
		logger.Ctx(ctx.Context()).Error().Err(err).Str("path", ctx.Path()).Msg("upload is held by a storage backend that is not configured")

		return ServiceUnavailable(ctx, "file storage is unavailable")
	}

	return InternalError(ctx, "failed to read stored file", err)
}

func SendStoredObject(ctx fiber.Ctx, storageSvc storage.Service, info engine.ObjectInfo) error {
	lastModified := info.ModTime.UTC().Truncate(time.Second)
	lastModifiedHeader := lastModified.Format(http.TimeFormat)
	etag := fmt.Sprintf(`W/"%x-%x"`, info.Size, info.ModTime.UnixNano())

	contentType := mime.TypeByExtension(path.Ext(info.Key))
	if contentType == "" {
		contentType = fiber.MIMEOctetStream
	}

	ctx.Set(fiber.HeaderContentType, contentType)
	ctx.Set(fiber.HeaderAcceptRanges, "bytes")
	ctx.Set(fiber.HeaderLastModified, lastModifiedHeader)
	ctx.Set(fiber.HeaderETag, etag)

	if notModified(ctx, etag, lastModified) {
		return ctx.SendStatus(fiber.StatusNotModified)
	}

	var rng *engine.ByteRange
	ifRange := ctx.Get(fiber.HeaderIfRange)
	if ifRange == "" || ifRange == lastModifiedHeader {
		parsed, satisfiable := ParseByteRange(ctx.Get(fiber.HeaderRange), info.Size)
		if !satisfiable {
			ctx.Set(fiber.HeaderContentRange, "bytes */"+strconv.FormatInt(info.Size, 10))

			return ctx.SendStatus(fiber.StatusRequestedRangeNotSatisfiable)
		}

		rng = parsed
	}

	status := fiber.StatusOK
	length := info.Size
	if rng != nil {
		status = fiber.StatusPartialContent
		length = rng.Length()
		ctx.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", rng.Start, rng.End, info.Size))
	}

	ctx.Status(status)

	if ctx.Method() == fiber.MethodHead {
		ctx.Response().Header.SetContentLength(int(length))

		return nil
	}

	body, err := storageSvc.Open(context.WithoutCancel(ctx.Context()), info, rng)
	if err != nil {
		return StoredObjectError(ctx, err)
	}

	return ctx.SendStream(body, int(length))
}

func ParseByteRange(header string, size int64) (*engine.ByteRange, bool) {
	spec, ok := strings.CutPrefix(strings.TrimSpace(header), "bytes=")
	if !ok || strings.Contains(spec, ",") || size <= 0 {
		return nil, true
	}

	startText, endText, ok := strings.Cut(strings.TrimSpace(spec), "-")
	if !ok {
		return nil, true
	}

	if startText == "" {
		suffix, err := strconv.ParseInt(endText, 10, 64)
		if err != nil || suffix < 0 {
			return nil, true
		}

		if suffix == 0 {
			return nil, false
		}

		return &engine.ByteRange{Start: max(0, size-suffix), End: size - 1}, true
	}

	start, err := strconv.ParseInt(startText, 10, 64)
	if err != nil || start < 0 {
		return nil, true
	}

	if start >= size {
		return nil, false
	}

	end := size - 1
	if endText != "" {
		parsedEnd, err := strconv.ParseInt(endText, 10, 64)
		if err != nil || parsedEnd < start {
			return nil, true
		}

		end = min(parsedEnd, size-1)
	}

	return &engine.ByteRange{Start: start, End: end}, true
}

func notModified(ctx fiber.Ctx, etag string, lastModified time.Time) bool {
	if ifNoneMatch := ctx.Get(fiber.HeaderIfNoneMatch); ifNoneMatch != "" {
		for candidate := range strings.SplitSeq(ifNoneMatch, ",") {
			candidate = strings.TrimSpace(candidate)
			if candidate == "*" || strings.TrimPrefix(candidate, "W/") == strings.TrimPrefix(etag, "W/") {
				return true
			}
		}

		return false
	}

	since, err := http.ParseTime(ctx.Get(fiber.HeaderIfModifiedSince))
	if err != nil {
		return false
	}

	return !lastModified.After(since)
}
