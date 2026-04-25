package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/db"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/media"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/upload"
)

type (
	stats struct {
		usersScanned    int
		fieldsChecked   int
		filesProcessed  int
		rowsUpdated     int
		filesConverted  int
		filesChanged    int
		bytesBefore     int64
		bytesAfter      int64
		skippedMissing  int
		skippedNotOwned int
		errors          int
	}
)

const (
	perFileTimeout = 3 * time.Minute
)

func main() {
	logger.Init(config.SettingLogLevel.Default)
	defer logger.Shutdown()

	database, err := db.Open(config.Cfg.PostgresDSN())
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to open database")
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to run migrations")
	}

	repos := repository.New(database)
	settingsSvc := settings.NewService(repos.Settings)
	if err := settingsSvc.Refresh(context.Background()); err != nil {
		logger.Log.Fatal().Err(err).Msg("failed to load settings")
	}

	uploadSvc := upload.NewService(settingsSvc)

	result, err := backfill(context.Background(), database, uploadSvc)
	if err != nil {
		logger.Log.Fatal().Err(err).Msg("backfill failed")
	}

	logger.Log.Info().
		Int("users_scanned", result.usersScanned).
		Int("fields_checked", result.fieldsChecked).
		Int("files_processed", result.filesProcessed).
		Int("rows_updated", result.rowsUpdated).
		Int("files_converted", result.filesConverted).
		Int("files_changed", result.filesChanged).
		Int64("bytes_before", result.bytesBefore).
		Int64("bytes_after", result.bytesAfter).
		Int64("bytes_saved", result.bytesBefore-result.bytesAfter).
		Int("skipped_missing", result.skippedMissing).
		Int("skipped_not_owned", result.skippedNotOwned).
		Int("errors", result.errors).
		Msg("profile media webp backfill complete")
}

func backfill(ctx context.Context, database *sql.DB, uploadSvc upload.Service) (stats, error) {
	result := stats{}

	rows, err := database.QueryContext(ctx, `SELECT id, avatar_url, banner_url FROM users`)
	if err != nil {
		return result, err
	}
	defer rows.Close()

	for rows.Next() {
		result.usersScanned++

		var userID string
		var avatarURL sql.NullString
		var bannerURL sql.NullString
		if err := rows.Scan(&userID, &avatarURL, &bannerURL); err != nil {
			return result, err
		}

		newAvatar := avatarURL.String
		avatarChanged := false
		if avatarURL.Valid {
			result.fieldsChecked++
			converted, changed, err := convertProfileURL(ctx, uploadSvc, avatarURL.String, "avatars", &result)
			if err != nil {
				result.errors++
				logger.Log.Warn().Err(err).Str("user_id", userID).Str("avatar_url", avatarURL.String).Msg("failed avatar conversion")
			} else {
				newAvatar = converted
				avatarChanged = changed
			}
		}

		newBanner := bannerURL.String
		bannerChanged := false
		if bannerURL.Valid {
			result.fieldsChecked++
			converted, changed, err := convertProfileURL(ctx, uploadSvc, bannerURL.String, "banners", &result)
			if err != nil {
				result.errors++
				logger.Log.Warn().Err(err).Str("user_id", userID).Str("banner_url", bannerURL.String).Msg("failed banner conversion")
			} else {
				newBanner = converted
				bannerChanged = changed
			}
		}

		if !avatarChanged && !bannerChanged {
			continue
		}

		if _, err := database.ExecContext(ctx, `UPDATE users SET avatar_url = ?, banner_url = ? WHERE id = ?`, newAvatar, newBanner, userID); err != nil {
			result.errors++
			logger.Log.Warn().Err(err).Str("user_id", userID).Msg("failed to update user profile urls")
			continue
		}

		result.rowsUpdated++
	}

	if err := rows.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func convertProfileURL(ctx context.Context, uploadSvc upload.Service, urlPath string, subDir string, result *stats) (string, bool, error) {
	normalizedPath := normalizeUploadPath(urlPath)
	lower := strings.ToLower(normalizedPath)
	if strings.TrimSpace(lower) == "" {
		return urlPath, false, nil
	}

	prefix := "/uploads/" + subDir + "/"
	if !strings.HasPrefix(lower, prefix) {
		result.skippedNotOwned++
		return urlPath, false, nil
	}

	diskPath := uploadSvc.FullDiskPath(normalizedPath)
	beforeInfo, err := os.Stat(diskPath)
	if err != nil {
		if os.IsNotExist(err) {
			result.skippedMissing++
			return urlPath, false, nil
		}
		return "", false, err
	}

	opts := media.WebPOptions{}
	switch subDir {
	case "avatars":
		opts.MaxWidth = media.AvatarMaxWidth
		opts.MaxHeight = media.AvatarMaxHeight
		opts.Quality = media.AvatarQuality
		opts.SquareCrop = true
	case "banners":
		opts.MaxWidth = media.BannerMaxWidth
		opts.MaxHeight = media.BannerMaxHeight
		opts.Quality = media.BannerQuality
	}

	jobCtx, cancel := context.WithTimeout(ctx, perFileTimeout)
	outputPath, err := media.EncodeWebP(jobCtx, diskPath, opts)
	cancel()
	if err != nil {
		return "", false, err
	}

	result.filesProcessed++
	result.bytesBefore += beforeInfo.Size()

	afterInfo, statErr := os.Stat(outputPath)
	afterSize := beforeInfo.Size()
	if statErr == nil {
		afterSize = afterInfo.Size()
	}
	result.bytesAfter += afterSize

	newURL := fmt.Sprintf("/uploads/%s/%s", subDir, filepath.Base(outputPath))
	if newURL == normalizedPath {
		logger.Log.Info().
			Str("sub_dir", subDir).
			Str("url", normalizedPath).
			Int64("bytes", afterSize).
			Msg("kept in place (animated webp or already optimised)")
		return urlPath, false, nil
	}

	result.filesConverted++
	result.filesChanged++
	logger.Log.Info().
		Str("sub_dir", subDir).
		Str("from", normalizedPath).
		Str("to", newURL).
		Int64("bytes_before", beforeInfo.Size()).
		Int64("bytes_after", afterSize).
		Msg("converted to webp")
	return newURL, true, nil
}

func normalizeUploadPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	if parsed, err := url.Parse(trimmed); err == nil {
		if parsed.Path != "" {
			return parsed.Path
		}
	}

	if i := strings.Index(trimmed, "?"); i >= 0 {
		trimmed = trimmed[:i]
	}
	if i := strings.Index(trimmed, "#"); i >= 0 {
		trimmed = trimmed[:i]
	}

	return trimmed
}
