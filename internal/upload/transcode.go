package upload

import (
	"context"
	"errors"
	"path/filepath"

	"umineko_city_of_books/internal/daoerr"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/model/spec"
)

func (s *service) TranscodeMedia(ctx context.Context, subDir string, stored []StoredMedia, rowIDs []int64, updateURL UpdateURLFn, updateThumb UpdateURLFn) {
	for i, m := range stored {
		if m.stagedVideo == "" {
			continue
		}

		if s.mediaProc == nil {
			s.Discard(m.stagedVideo)

			continue
		}

		s.mediaProc.Enqueue(media.Job{
			Type:          media.JobVideo,
			InputPath:     m.stagedVideo,
			Callback:      s.transcodeDone(ctx, subDir, m.URL, m.stagedVideo, rowIDs[i], updateURL, updateThumb),
			ErrorCallback: s.transcodeFailed(ctx, m.URL, m.stagedVideo, rowIDs[i]),
		})
	}
}

func (s *service) transcodeFailed(ctx context.Context, urlPath string, localPath string, rowID int64) func(err error) {
	return func(err error) {
		s.Discard(localPath)
		logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Str("url", urlPath).Msg("video was not transcoded, media row still points at the raw upload")
	}
}

func (s *service) transcodeDone(ctx context.Context, subDir string, urlPath string, localPath string, rowID int64, updateURL UpdateURLFn, updateThumb UpdateURLFn) func(outputPath string) {
	return func(outputPath string) {
		defer s.Discard(outputPath)

		bg := context.Background()

		newURL := urlPath
		if outputPath != localPath {
			stored, err := s.Store(bg, subDir, outputPath)
			if err != nil {
				logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg("failed to store transcoded video, keeping the source file")

				return
			}

			newURL = stored
		}

		if err := updateURL(bg, spec.MediaURLUpdate{ID: rowID, URL: newURL}); err != nil {
			logMissingRow(ctx, err, rowID, "media row was removed before its transcode finished, discarding the transcoded video", "failed to update video media url, keeping the source file")
			if newURL != urlPath {
				s.Delete(newURL)
			}

			return
		}

		if newURL != urlPath {
			s.Delete(urlPath)
		}

		thumbName, err := media.GenerateThumbnail(outputPath, filepath.Dir(outputPath), filepath.Base(outputPath))
		if err != nil {
			logger.Ctx(ctx).Error().Err(err).Msg("failed to generate video thumbnail")

			return
		}

		thumbURL, err := s.Store(bg, subDir, filepath.Join(filepath.Dir(outputPath), thumbName))
		if err != nil {
			logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg("failed to store video thumbnail")

			return
		}

		if err := updateThumb(bg, spec.MediaURLUpdate{ID: rowID, URL: thumbURL}); err != nil {
			logMissingRow(ctx, err, rowID, "media row was removed before its thumbnail was recorded, discarding the thumbnail", "failed to update video thumbnail url")
			s.Delete(thumbURL)
		}
	}
}

func logMissingRow(ctx context.Context, err error, rowID int64, goneMsg string, failedMsg string) {
	if errors.Is(err, daoerr.ErrNotFound) {
		logger.Ctx(ctx).Info().Err(err).Int64("media_id", rowID).Msg(goneMsg)

		return
	}

	logger.Ctx(ctx).Error().Err(err).Int64("media_id", rowID).Msg(failedMsg)
}
