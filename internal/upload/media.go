package upload

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"

	"github.com/google/uuid"
)

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
)

type (
	FileUpload struct {
		ContentType string
		Filename    string
		Size        int64
		IsSpoiler   bool
		Open        func() (io.ReadCloser, error)
	}

	UpdateURLFn func(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error

	StoredMedia struct {
		URL         string
		MediaType   string
		Filename    string
		SortOrder   int
		Width       int
		Height      int
		IsSpoiler   bool
		stagedVideo string
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

func (s *service) StoreMedia(ctx context.Context, subDir string, files []FileUpload) ([]StoredMedia, error) {
	if len(files) == 0 {
		return nil, nil
	}

	for _, f := range files {
		if err := s.checkMedia(ctx, f); err != nil {
			return nil, err
		}
	}

	stored := make([]StoredMedia, 0, len(files))
	for i, f := range files {
		m, err := s.storeMedia(ctx, subDir, i, f)
		if err != nil {
			s.DiscardMedia(stored...)

			return nil, err
		}

		stored = append(stored, m)
	}

	return stored, nil
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

func (s *service) storeMedia(ctx context.Context, subDir string, sortOrder int, f FileUpload) (StoredMedia, error) {
	kind := kindOf(f.ContentType)
	maxSize := int64(s.settingsSvc.GetInt(ctx, kind.sizeSetting))

	width, height := mediaDimensions(kind, f)

	r, err := f.Open()
	if err != nil {
		return StoredMedia{}, fmt.Errorf("open media: %w", err)
	}
	defer r.Close()

	logger.Ctx(ctx).Debug().Str("content_type", f.ContentType).Str("media_type", kind.mediaType).Int64("file_size", f.Size).Int64("max_size", maxSize).Msg("storing media")

	stored := StoredMedia{
		MediaType: kind.mediaType,
		Filename:  media.SafeFilename(f.Filename),
		SortOrder: sortOrder,
		Width:     width,
		Height:    height,
		IsSpoiler: f.IsSpoiler,
	}

	id := uuid.New()

	switch kind.mediaType {
	case MediaTypeVideo:
		staged, err := s.StageMedia(ctx, MediaTypeVideo, subDir, id, f.Size, maxSize, r)
		if err != nil {
			return StoredMedia{}, err
		}

		stored.URL, err = s.Store(ctx, subDir, staged)
		if err != nil {
			s.Discard(staged)

			return StoredMedia{}, err
		}

		stored.stagedVideo = staged
	case MediaTypeAudio:
		stored.URL, err = s.SaveAudio(ctx, subDir, id, f.Size, maxSize, r)
	default:
		stored.URL, err = s.SaveImage(ctx, subDir, id, f.Size, maxSize, r)
	}
	if err != nil {
		return StoredMedia{}, err
	}

	return stored, nil
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

func (s *service) DiscardMedia(stored ...StoredMedia) {
	for _, m := range stored {
		if m.stagedVideo != "" {
			s.Discard(m.stagedVideo)
		}

		s.Delete(m.URL)
	}
}

func (m StoredMedia) NewMedia(targetID uuid.UUID) spec.NewMedia {
	return spec.NewMedia{
		TargetID:  targetID,
		MediaURL:  m.URL,
		MediaType: m.MediaType,
		Filename:  m.Filename,
		SortOrder: m.SortOrder,
		IsSpoiler: m.IsSpoiler,
	}
}

func (m StoredMedia) Response(rowID int64) dto.PostMediaResponse {
	return dto.PostMediaResponse{
		ID:        int(rowID),
		MediaURL:  m.URL,
		MediaType: m.MediaType,
		Filename:  m.Filename,
		SortOrder: m.SortOrder,
		Width:     m.Width,
		Height:    m.Height,
		IsSpoiler: m.IsSpoiler,
	}
}

func MediaRows(stored []StoredMedia) []spec.NewMedia {
	var rows []spec.NewMedia
	for _, m := range stored {
		rows = append(rows, m.NewMedia(uuid.Nil))
	}

	return rows
}

func MediaResponses(stored []StoredMedia, rowIDs []int64) []dto.PostMediaResponse {
	var responses []dto.PostMediaResponse
	for i, m := range stored {
		responses = append(responses, m.Response(rowIDs[i]))
	}

	return responses
}
