package upload

import (
	"context"
	"fmt"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	mediaSubDir       = "media"
	filesSubDir       = "attachments"
	eventUploadReady  = "upload_ready"
	eventUploadFailed = "upload_failed"
)

func pendingResponse(row model.PendingUploadRow) dto.PendingUploadResponse {
	return dto.PendingUploadResponse{
		ID:           row.ID,
		MediaType:    row.MediaType,
		Filename:     row.Filename,
		Status:       string(row.Status),
		MediaURL:     row.MediaURL,
		ThumbnailURL: row.ThumbnailURL,
	}
}

func (s *service) BeginMediaUpload(ctx context.Context, userID uuid.UUID, file FileUpload) (*dto.PendingUploadResponse, error) {
	if err := s.checkMedia(ctx, file); err != nil {
		return nil, err
	}

	kind := kindOf(file.ContentType)
	maxSize := int64(s.settingsSvc.GetInt(ctx, kind.sizeSetting))
	width, height := mediaDimensions(kind, file)

	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open media: %w", err)
	}
	defer r.Close()

	id := uuid.New()

	staged, err := s.StageMedia(ctx, kind.mediaType, mediaSubDir, id, file.Size, maxSize, r)
	if err != nil {
		return nil, err
	}

	pending := spec.NewPendingUpload{ID: id, UserID: userID, MediaType: kind.mediaType, Filename: media.SafeFilename(file.Filename), FileSize: file.Size}

	return s.begin(ctx, pending, staged, func(bg context.Context) (spec.ProcessedUpload, error) {
		processed, err := s.processMedia(bg, kind.mediaType, staged)
		processed.Width = width
		processed.Height = height

		return processed, err
	})
}

func (s *service) BeginFileUpload(ctx context.Context, userID uuid.UUID, file FileUpload) (*dto.PendingUploadResponse, error) {
	maxSize := int64(s.settingsSvc.GetInt(ctx, config.SettingMaxGeneralSize))
	if err := checkAttachmentSize(file.Size, maxSize); err != nil {
		return nil, err
	}

	r, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open attachment: %w", err)
	}
	defer r.Close()

	ext, wrapped, err := sniffAttachment(r)
	if err != nil {
		return nil, err
	}

	id := uuid.New()

	staged, err := s.Stage(id.String()+ext, wrapped)
	if err != nil {
		return nil, err
	}

	pending := spec.NewPendingUpload{ID: id, UserID: userID, MediaType: MediaTypeFile, Filename: media.SafeFilename(file.Filename), FileSize: file.Size}

	return s.begin(ctx, pending, staged, func(bg context.Context) (spec.ProcessedUpload, error) {
		url, err := s.Store(bg, filesSubDir, staged)

		return spec.ProcessedUpload{MediaURL: url}, err
	})
}

func (s *service) begin(ctx context.Context, pending spec.NewPendingUpload, staged string, process func(context.Context) (spec.ProcessedUpload, error)) (*dto.PendingUploadResponse, error) {
	if err := s.pending.Insert(ctx, pending); err != nil {
		s.Discard(staged)

		return nil, err
	}

	bg := context.WithoutCancel(ctx)
	go s.finish(bg, pending, staged, process)

	return &dto.PendingUploadResponse{
		ID:        pending.ID,
		MediaType: pending.MediaType,
		Filename:  pending.Filename,
		Status:    string(model.PendingUploadProcessing),
	}, nil
}

func (s *service) finish(ctx context.Context, pending spec.NewPendingUpload, staged string, process func(context.Context) (spec.ProcessedUpload, error)) {
	defer s.Discard(staged)

	processed, err := process(ctx)
	if err != nil {
		s.failUpload(ctx, pending, err)

		return
	}

	processed.ID = pending.ID

	marked, err := s.pending.MarkReady(ctx, processed)
	if err != nil || !marked {
		s.Delete(storedPaths(processed)...)

		if err != nil {
			s.failUpload(ctx, pending, err)
		}

		return
	}

	s.hub.SendToUser(pending.UserID, ws.Message{Type: eventUploadReady, Data: dto.PendingUploadResponse{
		ID:           pending.ID,
		MediaType:    pending.MediaType,
		Filename:     pending.Filename,
		Status:       string(model.PendingUploadReady),
		MediaURL:     processed.MediaURL,
		ThumbnailURL: processed.ThumbnailURL,
	}})
}

func (s *service) failUpload(ctx context.Context, pending spec.NewPendingUpload, cause error) {
	logger.Ctx(ctx).Error().Err(cause).Str("upload_id", pending.ID.String()).Str("media_type", pending.MediaType).Msg("upload could not be processed")

	marked, err := s.pending.MarkFailed(ctx, spec.UserUpload{ID: pending.ID, UserID: pending.UserID})
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("upload_id", pending.ID.String()).Msg("failed to record a failed upload")

		return
	}

	if !marked {
		return
	}

	s.hub.SendToUser(pending.UserID, ws.Message{Type: eventUploadFailed, Data: dto.PendingUploadResponse{
		ID:        pending.ID,
		MediaType: pending.MediaType,
		Filename:  pending.Filename,
		Status:    string(model.PendingUploadFailed),
	}})
}

func (s *service) processMedia(ctx context.Context, mediaType string, staged string) (spec.ProcessedUpload, error) {
	switch mediaType {
	case MediaTypeImage:
		url, err := s.storeImage(ctx, mediaSubDir, staged)

		return spec.ProcessedUpload{MediaURL: url}, err
	case MediaTypeVideo:
		return s.storeVideo(ctx, staged)
	default:
		url, err := s.Store(ctx, mediaSubDir, staged)

		return spec.ProcessedUpload{MediaURL: url}, err
	}
}

func (s *service) storeVideo(ctx context.Context, staged string) (spec.ProcessedUpload, error) {
	if s.mediaProc == nil {
		url, err := s.Store(ctx, mediaSubDir, staged)

		return spec.ProcessedUpload{MediaURL: url}, err
	}

	maxPixels := s.settingsSvc.GetInt(ctx, config.SettingMaxImagePixels)

	encoded, err := s.runJob(ctx, media.Job{Type: media.JobVideo, InputPath: staged, MaxPixels: maxPixels})
	if err != nil {
		return spec.ProcessedUpload{}, fmt.Errorf("encode video: %w", err)
	}

	url, err := s.Store(ctx, mediaSubDir, encoded)
	if err != nil {
		return spec.ProcessedUpload{}, err
	}

	thumbPath, err := s.runJob(ctx, media.Job{Type: media.JobThumbnail, InputPath: encoded, MaxPixels: maxPixels})
	if err != nil {
		logger.Ctx(ctx).Error().Err(err).Str("url", url).Msg("failed to generate video thumbnail, keeping the video without one")

		return spec.ProcessedUpload{MediaURL: url}, nil
	}

	thumbURL, err := s.Store(ctx, mediaSubDir, thumbPath)
	if err != nil {
		s.Delete(url)

		return spec.ProcessedUpload{}, err
	}

	return spec.ProcessedUpload{MediaURL: url, ThumbnailURL: thumbURL}, nil
}

func storedPaths(p spec.ProcessedUpload) []string {
	var paths []string
	for _, path := range []string{p.MediaURL, p.ThumbnailURL} {
		if path != "" {
			paths = append(paths, path)
		}
	}

	return paths
}

func rowPaths(rows []model.PendingUploadRow) []string {
	var paths []string
	for _, row := range rows {
		paths = append(paths, storedPaths(spec.ProcessedUpload{MediaURL: row.MediaURL, ThumbnailURL: row.ThumbnailURL})...)
	}

	return paths
}

func (s *service) UploadStatuses(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]dto.PendingUploadResponse, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := s.pending.List(ctx, spec.UserUploads{UserID: userID, IDs: ids})
	if err != nil {
		return nil, err
	}

	var statuses []dto.PendingUploadResponse
	for _, row := range rows {
		statuses = append(statuses, pendingResponse(row))
	}

	return statuses, nil
}

func (s *service) RemoveUpload(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	rows, err := s.pending.Delete(ctx, spec.UserUpload{ID: id, UserID: userID})
	if err != nil {
		return err
	}

	s.Delete(rowPaths(rows)...)

	return nil
}

func (s *service) ExpireUploads(ctx context.Context) (int, error) {
	rows, err := s.pending.DeleteExpired(ctx, time.Now().Add(-PendingUploadLifetime))
	if err != nil {
		return 0, err
	}

	s.Delete(rowPaths(rows)...)

	return len(rows), nil
}

func (s *service) FailInterruptedUploads(ctx context.Context) error {
	_, err := s.pending.FailProcessing(ctx)

	return err
}
