package upload

import (
	"fmt"
	"io"
)

func ValidateAttachmentFile(f FileUpload, maxSize int64) error {
	if err := checkAttachmentSize(f.Size, maxSize); err != nil {
		return err
	}

	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open attachment: %w", err)
	}
	defer r.Close()

	_, _, err = sniffAttachment(r)

	return err
}

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
