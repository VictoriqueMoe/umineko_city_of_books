package media

import (
	"context"
	"io"
	"path/filepath"
	"strings"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"

	"github.com/google/uuid"
)

type (
	UpdateURLFn func(ctx context.Context, id int64, url string) error
	AddFn       func(mediaURL, mediaType, thumbURL string, sortOrder int) (int64, error)

	Uploader struct {
		uploadSvc   upload.Service
		settingsSvc settings.Service
		processor   *Processor
	}
)

func NewUploader(uploadSvc upload.Service, settingsSvc settings.Service, processor *Processor) *Uploader {
	return &Uploader{
		uploadSvc:   uploadSvc,
		settingsSvc: settingsSvc,
		processor:   processor,
	}
}

func (u *Uploader) SaveAndRecord(
	ctx context.Context,
	subDir string,
	contentType string,
	fileSize int64,
	reader io.Reader,
	addFn AddFn,
	updateURL UpdateURLFn,
	updateThumb UpdateURLFn,
) (*dto.PostMediaResponse, error) {
	isVideo := strings.HasPrefix(contentType, "video/")
	mediaID := uuid.New()

	var urlPath string
	var err error
	if isVideo {
		maxSize := int64(u.settingsSvc.GetInt(ctx, config.SettingMaxVideoSize))
		logger.Log.Debug().Str("content_type", contentType).Int64("file_size", fileSize).Int64("max_size", maxSize).Msg("uploading video")
		urlPath, err = u.uploadSvc.SaveVideo(ctx, subDir, mediaID, contentType, fileSize, maxSize, reader)
	} else {
		maxSize := int64(u.settingsSvc.GetInt(ctx, config.SettingMaxImageSize))
		logger.Log.Debug().Str("content_type", contentType).Int64("file_size", fileSize).Int64("max_size", maxSize).Msg("uploading image")
		urlPath, err = u.uploadSvc.SaveImage(ctx, subDir, mediaID, contentType, fileSize, maxSize, reader)
	}
	if err != nil {
		return nil, err
	}

	mediaType := "image"
	if isVideo {
		mediaType = "video"
	}

	rowID, err := addFn(urlPath, mediaType, "", 0)
	if err != nil {
		return nil, err
	}

	diskPath := u.uploadSvc.FullDiskPath(urlPath)
	if isVideo {
		u.processor.Enqueue(Job{
			Type:      JobVideo,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/" + subDir + "/" + filepath.Base(outputPath)
				if err := updateURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video media url")
				}
				thumbName, err := GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
				if err != nil {
					logger.Log.Error().Err(err).Msg("failed to generate video thumbnail")
					return
				}
				thumbURL := "/uploads/" + subDir + "/" + thumbName
				if err := updateThumb(context.Background(), rowID, thumbURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update video thumbnail url")
				}
			},
		})
	} else {
		done := make(chan string, 1)
		u.processor.Enqueue(Job{
			Type:      JobImage,
			InputPath: diskPath,
			Callback: func(outputPath string) {
				newURL := "/uploads/" + subDir + "/" + filepath.Base(outputPath)
				if err := updateURL(context.Background(), rowID, newURL); err != nil {
					logger.Log.Error().Err(err).Msg("failed to update image media url")
				}
				done <- newURL
			},
		})
		select {
		case newURL := <-done:
			urlPath = newURL
		case <-ctx.Done():
		}
	}

	return &dto.PostMediaResponse{
		ID:        int(rowID),
		MediaURL:  urlPath,
		MediaType: mediaType,
	}, nil
}
