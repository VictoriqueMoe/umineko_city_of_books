package utils

import (
	"errors"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

func IsUploadRejection(err error) bool {
	return errors.Is(err, upload.ErrFileTooLarge) ||
		errors.Is(err, upload.ErrInvalidFileType) ||
		errors.Is(err, upload.ErrInvalidVideoType) ||
		errors.Is(err, upload.ErrInvalidAudioType) ||
		errors.Is(err, upload.ErrInvalidAttachmentType) ||
		errors.Is(err, bounds.ErrImageBounds)
}

func UploadRejected(ctx fiber.Ctx, err error) error {
	msg := rejectionMessage(err)

	if msg != err.Error() {
		logger.Ctx(ctx.Context()).Error().
			Err(err).
			Str("method", ctx.Method()).
			Str("path", ctx.Path()).
			Msg("upload rejection carried a further failure")
	}

	return BadRequest(ctx, msg)
}

func rejectionMessage(err error) string {
	switch wrapped := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range wrapped.Unwrap() {
			if IsUploadRejection(child) {
				return rejectionMessage(child)
			}
		}
	case interface{ Unwrap() error }:
		inner := wrapped.Unwrap()
		if carriesJoin(inner) {
			return rejectionMessage(inner)
		}
	}

	return err.Error()
}

func carriesJoin(err error) bool {
	for err != nil {
		if _, ok := err.(interface{ Unwrap() []error }); ok {
			return true
		}

		err = errors.Unwrap(err)
	}

	return false
}
