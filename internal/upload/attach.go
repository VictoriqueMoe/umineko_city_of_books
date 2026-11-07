package upload

import (
	"fmt"
	"io"
)

func checkAttachmentSize(fileSize int64, maxSize int64) error {
	if fileSize > maxSize {
		return fmt.Errorf("%w: file size %dMB exceeds maximum %dMB", ErrFileTooLarge, fileSize/(1024*1024), maxSize/(1024*1024))
	}

	return nil
}

func sniffAttachment(reader io.Reader) (string, io.Reader, error) {
	sniffed, wrapped, err := DetectContentType(reader)
	if err != nil {
		return "", nil, err
	}

	ext, ok := AllowedAttachmentTypes[NormaliseSniffedType(sniffed)]
	if !ok {
		return "", nil, ErrInvalidAttachmentType
	}

	return ext, wrapped, nil
}
