package utils

import (
	"errors"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/upload"
)

func IsUploadRejection(err error) bool {
	return errors.Is(err, upload.ErrFileTooLarge) ||
		errors.Is(err, upload.ErrInvalidFileType) ||
		errors.Is(err, upload.ErrInvalidVideoType) ||
		errors.Is(err, upload.ErrInvalidAudioType) ||
		errors.Is(err, upload.ErrInvalidAttachmentType) ||
		errors.Is(err, bounds.ErrImageBounds)
}
