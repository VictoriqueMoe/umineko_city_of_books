package upload

import "errors"

var (
	ErrFileTooLarge     = errors.New("file size must be under 50MB")
	ErrInvalidFileType  = errors.New("file must be PNG, JPG, GIF, or WebP")
	ErrInvalidVideoType = errors.New("file must be MP4, WebM, MOV, AVI, or MKV")
)
