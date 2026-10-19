package upload

import (
	"context"
	"fmt"
	"io"
	"strings"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

type (
	FileUpload struct {
		ContentType string
		Filename    string
		Size        int64
		IsSpoiler   bool
		Open        func() (io.ReadCloser, error)
	}

	MediaRecorder interface {
		SaveAndRecord(
			ctx context.Context,
			subDir string,
			contentType string,
			filename string,
			fileSize int64,
			reader io.Reader,
			isSpoiler bool,
			addFn media.AddFn,
			updateURL media.UpdateURLFn,
			updateThumb media.UpdateURLFn,
		) (*dto.PostMediaResponse, error)
	}

	MediaDeleter interface {
		Delete(urlPaths ...string)
	}

	MediaRecord struct {
		File         FileUpload
		MediaURL     string
		MediaType    string
		ThumbnailURL string
		Filename     string
		SortOrder    int
		Width        int
		Height       int
	}

	MediaTarget struct {
		SubDir      string
		Record      func(r MediaRecord) (int64, error)
		UpdateURL   media.UpdateURLFn
		UpdateThumb media.UpdateURLFn
	}

	ValidatedMedia struct {
		files []FileUpload
	}
)

func ValidateMedia(ctx context.Context, settingsSvc settings.Service, files []FileUpload) (ValidatedMedia, error) {
	for _, f := range files {
		if err := ValidateMediaFile(ctx, settingsSvc, f); err != nil {
			return ValidatedMedia{}, err
		}
	}

	return ValidatedMedia{files: files}, nil
}

func ValidateMediaFile(ctx context.Context, settingsSvc settings.Service, f FileUpload) error {
	var maxSize int64
	var allowed map[string]string
	var typeErr error

	isImage := false
	switch {
	case strings.HasPrefix(f.ContentType, "video/"):
		maxSize = int64(settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		allowed = AllowedVideoTypes
		typeErr = ErrInvalidVideoType
	case strings.HasPrefix(f.ContentType, "audio/"):
		maxSize = int64(settingsSvc.GetInt(ctx, config.SettingMaxAudioSize))
		allowed = AllowedAudioTypes
		typeErr = ErrInvalidAudioType
	default:
		maxSize = int64(settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		allowed = AllowedImageTypes
		typeErr = ErrInvalidFileType
		isImage = true
	}

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
	if _, ok := allowed[NormaliseSniffedType(sniffed)]; !ok {
		return typeErr
	}

	if !isImage {
		return nil
	}

	width, height := media.DecodeDimensions(wrapped)
	if width == 0 || height == 0 {
		return nil
	}

	return bounds.ImagePixels(width, height, settingsSvc.GetInt(ctx, config.SettingMaxImagePixels))
}

func ProbeMediaDimensions(f FileUpload) (int, int) {
	if strings.HasPrefix(f.ContentType, "video/") {
		return 0, 0
	}

	r, err := f.Open()
	if err != nil {
		return 0, 0
	}
	defer r.Close()

	return media.DecodeDimensions(r)
}

func (r MediaRecord) NewMedia(targetID uuid.UUID) spec.NewMedia {
	return spec.NewMedia{
		TargetID:     targetID,
		MediaURL:     r.MediaURL,
		MediaType:    r.MediaType,
		ThumbnailURL: r.ThumbnailURL,
		Filename:     r.Filename,
		SortOrder:    r.SortOrder,
		IsSpoiler:    r.File.IsSpoiler,
	}
}

func (v ValidatedMedia) Save(ctx context.Context, recorder MediaRecorder, deleter MediaDeleter, target MediaTarget) ([]dto.PostMediaResponse, error) {
	if len(v.files) == 0 {
		return nil, nil
	}

	saved := make([]dto.PostMediaResponse, 0, len(v.files))
	for i, f := range v.files {
		resp, err := saveMediaFile(ctx, recorder, target, i, f)
		if err != nil {
			DiscardSaved(deleter, saved)

			return nil, err
		}

		saved = append(saved, *resp)
	}

	return saved, nil
}

func DiscardSaved(deleter MediaDeleter, saved []dto.PostMediaResponse) {
	if len(saved) == 0 {
		return
	}

	stored := make([]string, 0, len(saved))
	for _, s := range saved {
		stored = append(stored, s.MediaURL)
	}

	deleter.Delete(stored...)
}

func saveMediaFile(ctx context.Context, recorder MediaRecorder, target MediaTarget, sortOrder int, f FileUpload) (*dto.PostMediaResponse, error) {
	width, height := ProbeMediaDimensions(f)

	r, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open media: %w", err)
	}
	defer r.Close()

	saved, err := recorder.SaveAndRecord(ctx, target.SubDir, f.ContentType, f.Filename, f.Size, r, f.IsSpoiler,
		func(mediaURL, mediaType, thumbURL, filename string, _ int) (int64, error) {
			return target.Record(MediaRecord{
				File:         f,
				MediaURL:     mediaURL,
				MediaType:    mediaType,
				ThumbnailURL: thumbURL,
				Filename:     filename,
				SortOrder:    sortOrder,
				Width:        width,
				Height:       height,
			})
		},
		target.UpdateURL,
		target.UpdateThumb,
	)
	if err != nil {
		return nil, err
	}

	saved.SortOrder = sortOrder
	saved.Width = width
	saved.Height = height
	saved.IsSpoiler = f.IsSpoiler

	return saved, nil
}
