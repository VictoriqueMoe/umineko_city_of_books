package media

import (
	"context"
	"database/sql"
	"io"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/settings"

	"github.com/google/uuid"
)

const (
	MediaTypeImage = "image"
	MediaTypeVideo = "video"
	MediaTypeAudio = "audio"
)

type (
	UpdateURLFn func(ctx context.Context, s spec.MediaURLUpdate, tx ...*sql.Tx) error
	AddFn       func(mediaURL, mediaType, thumbURL, filename string, sortOrder int) (int64, error)
	uploadSvc   interface {
		SaveImage(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		StageMedia(ctx context.Context, mediaType string, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		SaveAudio(ctx context.Context, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, error)
		Store(ctx context.Context, subDir string, localPath string) (string, error)
		Discard(localPath string)
		Delete(urlPaths ...string)
	}

	Uploader struct {
		uploadSvc   uploadSvc
		settingsSvc settings.Service
		processor   *Processor
	}
)

func NewUploader(uploadSvc uploadSvc, settingsSvc settings.Service, processor *Processor) *Uploader {
	return &Uploader{
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		processor:   processor,
	}
}

func kindFor(contentType string) (string, *config.SiteSettingDef) {
	switch {
	case strings.HasPrefix(contentType, "video/"):
		return MediaTypeVideo, config.SettingMaxVideoSize
	case strings.HasPrefix(contentType, "audio/"):
		return MediaTypeAudio, config.SettingMaxAudioSize
	default:
		return MediaTypeImage, config.SettingMaxImageSize
	}
}

func (u *Uploader) save(ctx context.Context, mediaType string, subDir string, id uuid.UUID, fileSize int64, maxSize int64, reader io.Reader) (string, string, error) {
	switch mediaType {
	case MediaTypeVideo:
		staged, err := u.uploadSvc.StageMedia(ctx, MediaTypeVideo, subDir, id, fileSize, maxSize, reader)
		if err != nil {
			return "", "", err
		}

		urlPath, err := u.uploadSvc.Store(ctx, subDir, staged)
		if err != nil {
			u.uploadSvc.Discard(staged)

			return "", "", err
		}

		return urlPath, staged, nil
	case MediaTypeAudio:
		urlPath, err := u.uploadSvc.SaveAudio(ctx, subDir, id, fileSize, maxSize, reader)

		return urlPath, "", err
	default:
		urlPath, err := u.uploadSvc.SaveImage(ctx, subDir, id, fileSize, maxSize, reader)

		return urlPath, "", err
	}
}

func (u *Uploader) SaveAndRecord(
	ctx context.Context,
	subDir string,
	contentType string,
	filename string,
	fileSize int64,
	reader io.Reader,
	isSpoiler bool,
	addFn AddFn,
	updateURL UpdateURLFn,
	updateThumb UpdateURLFn,
) (*dto.PostMediaResponse, error) {
	mediaType, sizeSetting := kindFor(contentType)
	mediaID := uuid.New()
	maxSize := int64(u.settingsSvc.GetInt(ctx, sizeSetting))

	logger.Ctx(ctx).Debug().Str("content_type", contentType).Str("media_type", mediaType).Int64("file_size", fileSize).Int64("max_size", maxSize).Msg("uploading media")

	urlPath, stagedVideo, err := u.save(ctx, mediaType, subDir, mediaID, fileSize, maxSize, reader)
	if err != nil {
		return nil, err
	}

	safeName := SafeFilename(filename)

	rowID, err := addFn(urlPath, mediaType, "", safeName, 0)
	if err != nil {
		if stagedVideo != "" {
			u.uploadSvc.Discard(stagedVideo)
		}
		u.uploadSvc.Delete(urlPath)

		return nil, err
	}

	if stagedVideo != "" {
		u.enqueueTranscode(ctx, subDir, urlPath, stagedVideo, rowID, updateURL, updateThumb)
	}

	return &dto.PostMediaResponse{
		ID:        int(rowID),
		MediaURL:  urlPath,
		MediaType: mediaType,
		Filename:  safeName,
		IsSpoiler: isSpoiler,
	}, nil
}

func (u *Uploader) enqueueTranscode(ctx context.Context, subDir string, urlPath string, localPath string, rowID int64, updateURL UpdateURLFn, updateThumb UpdateURLFn) {
	u.processor.Enqueue(Job{
		Type:      JobVideo,
		InputPath: localPath,
		Callback: func(outputPath string) {
			defer u.uploadSvc.Discard(outputPath)

			bg := context.Background()

			newURL := urlPath
			if outputPath != localPath {
				stored, err := u.uploadSvc.Store(bg, subDir, outputPath)
				if err != nil {
					logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg("failed to store transcoded video, keeping the source file")

					return
				}

				newURL = stored
			}

			if err := updateURL(bg, spec.MediaURLUpdate{ID: rowID, URL: newURL}); err != nil {
				logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg("failed to update video media url, keeping the source file")
				if newURL != urlPath {
					u.uploadSvc.Delete(newURL)
				}

				return
			}

			if newURL != urlPath {
				u.uploadSvc.Delete(urlPath)
			}

			thumbName, err := GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
			if err != nil {
				logger.Ctx(ctx).Error().Err(err).Msg("failed to generate video thumbnail")
				return
			}

			thumbURL, err := u.uploadSvc.Store(bg, subDir, filepath.Join(filepath.Dir(outputPath), thumbName))
			if err != nil {
				logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg("failed to store video thumbnail")
				return
			}

			if err := updateThumb(bg, spec.MediaURLUpdate{ID: rowID, URL: thumbURL}); err != nil {
				logger.Ctx(ctx).Error().Err(err).Msg("failed to update video thumbnail url")
				u.uploadSvc.Delete(thumbURL)
			}
		},
		ErrorCallback: func(err error) {
			u.uploadSvc.Discard(localPath)
			logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Str("url", urlPath).Msg("video was not transcoded, media row still points at the raw upload")
		},
	})
}
