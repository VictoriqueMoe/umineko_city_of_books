package main

import (
	"context"
	"time"

	"umineko_city_of_books/internal/email"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/telemetry"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

func registerListeners(settingsSvc settings.Service, app *fiber.App, svc *services, repos *repository.Repositories) {
	settingsSvc.Subscribe(logger.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewProfilingSettingsListener())
	settingsSvc.Subscribe(middleware.NewBodyLimitListener(app))
	settingsSvc.Subscribe(email.NewMailSettingListener(svc.email))

	if err := svc.chat.EnsureSystemRooms(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("ensure system chat rooms at startup")
	}

	uploadDir := svc.upload.GetUploadDir()

	scheduleJob("refresh stale embeds", "refreshed stale embeds", time.Hour, func() (int, error) {
		return svc.post.RefreshStaleEmbeds(context.Background()), nil
	})
	scheduleJob("clean orphaned uploads", "cleaned orphaned upload files", 24*time.Hour, func() (int, error) {
		return upload.CleanOrphanedFiles(repos.Upload, uploadDir), nil
	})
	scheduleJob("archive stale journals", "archived stale journals", time.Hour, func() (int, error) {
		return svc.journal.ArchiveStale(context.Background())
	})
	scheduleJob("archive stale chat rooms", "archived stale chat rooms", time.Hour, func() (int, error) {
		return svc.chat.ArchiveStale(context.Background())
	})
	scheduleJob("cancel idle games", "cancelled idle games", 5*time.Minute, func() (int, error) {
		return svc.gameRoom.CancelIdleGames(context.Background())
	})
}

func scheduleJob(name string, successMsg string, interval time.Duration, fn func() (int, error)) {
	logger.Log.Info().Str("interval", interval.String()).Msgf("registered job: %s", name)
	go func() {
		run := func() {
			n, err := fn()
			if err != nil {
				logger.Log.Error().Err(err).Msgf("%s failed", name)
				return
			}
			if n > 0 {
				logger.Log.Info().Int("count", n).Msg(successMsg)
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}
