package utils

import (
	"errors"
)

var (
	ErrNotMultipart     = errors.New("expected a multipart form")
	ErrUploadTooLarge   = errors.New("the upload is larger than allowed")
	ErrNoFile           = errors.New("no file provided")
	ErrNewAccountUpload = errors.New("new accounts cannot post attachments")
)
