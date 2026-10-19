package upload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"

	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

type (
	EntityMediaStore interface {
		AddMedia(ctx context.Context, s spec.NewMedia, tx ...*sql.Tx) (int64, error)
		UpdateMediaURL(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
		UpdateMediaThumbnail(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
	}

	OwnedMediaStore interface {
		DeleteWithMedia(ctx context.Context, s spec.OwnedDeletion, tx ...*sql.Tx) ([]string, error)
	}
)

func (v ValidatedMedia) SaveToEntity(ctx context.Context, recorder MediaRecorder, deleter MediaDeleter, subDir string, store EntityMediaStore, entityID uuid.UUID) error {
	_, err := v.Save(ctx, recorder, deleter, MediaTarget{
		SubDir: subDir,
		Record: func(r MediaRecord) (int64, error) {
			return store.AddMedia(ctx, r.NewMedia(entityID))
		},
		UpdateURL:   store.UpdateMediaURL,
		UpdateThumb: store.UpdateMediaThumbnail,
	})

	return err
}

func DiscardOwned(ctx context.Context, store OwnedMediaStore, deleter MediaDeleter, kind string, deletion spec.OwnedDeletion, cause error) error {
	paths, err := store.DeleteWithMedia(context.WithoutCancel(ctx), deletion)

	deleter.Delete(paths...)

	if err != nil {
		return errors.Join(cause, fmt.Errorf("roll back %s %s: %w", kind, deletion.ID, err))
	}

	return cause
}

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
