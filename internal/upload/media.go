package upload

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
	MediaTypeFile  = "file"

	PendingUploadLifetime = time.Hour
)

type (
	FileUpload struct {
		ContentType string
		Filename    string
		Size        int64
		Open        func() (io.ReadCloser, error)
	}

	mediaKind struct {
		mediaType   string
		sizeSetting *config.SiteSettingDef
		allowed     map[string]string
		typeErr     error
	}
)

func kindOf(contentType string) mediaKind {
	switch {
	case strings.HasPrefix(contentType, "video/"):
		return mediaKind{mediaType: MediaTypeVideo, sizeSetting: config.SettingMaxVideoSize, allowed: AllowedVideoTypes, typeErr: ErrInvalidVideoType}
	case strings.HasPrefix(contentType, "audio/"):
		return mediaKind{mediaType: MediaTypeAudio, sizeSetting: config.SettingMaxAudioSize, allowed: AllowedAudioTypes, typeErr: ErrInvalidAudioType}
	default:
		return mediaKind{mediaType: MediaTypeImage, sizeSetting: config.SettingMaxImageSize, allowed: AllowedImageTypes, typeErr: ErrInvalidFileType}
	}
}

func (s *service) checkMedia(ctx context.Context, f FileUpload) error {
	kind := kindOf(f.ContentType)

	maxSize := int64(s.settingsSvc.GetInt(ctx, kind.sizeSetting))
	if f.Size > maxSize {
		return fmt.Errorf("%w: file size %dMB exceeds maximum %dMB", ErrFileTooLarge, f.Size/(1024*1024), maxSize/(1024*1024))
	}

	r, err := f.Open()
	if err != nil {
		return fmt.Errorf("open media: %w", err)
	}
	defer r.Close()

	sniffed, wrapped, err := DetectContentType(r)
	if err != nil {
		return err
	}
	if _, ok := kind.allowed[NormaliseSniffedType(sniffed)]; !ok {
		return kind.typeErr
	}

	if kind.mediaType != MediaTypeImage {
		return nil
	}

	width, height := media.DecodeDimensions(wrapped)
	if width == 0 || height == 0 {
		return nil
	}

	return bounds.ImagePixels(width, height, s.settingsSvc.GetInt(ctx, config.SettingMaxImagePixels))
}

func mediaDimensions(kind mediaKind, f FileUpload) (int, int) {
	if kind.mediaType != MediaTypeImage {
		return 0, 0
	}

	r, err := f.Open()
	if err != nil {
		return 0, 0
	}
	defer r.Close()

	return media.DecodeDimensions(r)
}

func claimOf(userID uuid.UUID, refs []dto.UploadRef, mediaTypes []string) spec.ClaimedUploads {
	var uploads []spec.UploadRef
	for _, ref := range refs {
		uploads = append(uploads, spec.UploadRef{ID: ref.ID, IsSpoiler: ref.IsSpoiler})
	}

	return spec.ClaimedUploads{
		UserID:     userID,
		Uploads:    uploads,
		MediaTypes: mediaTypes,
		MaxAge:     PendingUploadLifetime,
	}
}

func MediaClaim(userID uuid.UUID, refs []dto.UploadRef) spec.ClaimedUploads {
	return claimOf(userID, refs, []string{MediaTypeImage, MediaTypeVideo, MediaTypeAudio})
}

func FileClaim(userID uuid.UUID, refs []dto.UploadRef) spec.ClaimedUploads {
	return claimOf(userID, refs, []string{MediaTypeFile})
}
