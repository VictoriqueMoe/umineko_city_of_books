package main

import (
	"context"
	"sync"
	"time"

	"umineko_city_of_books/internal/cache/engines"
	"umineko_city_of_books/internal/chatbot"
	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/dronebl/feed"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/middleware"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/settings"
	"umineko_city_of_books/internal/telemetry"
	"umineko_city_of_books/internal/upload"

	"github.com/gofiber/fiber/v3"
)

func registerListeners(settingsSvc settings.Service, app *fiber.App, svc *services, repos *repository.Repositories) func(ctx context.Context) {
	stop := make(chan struct{})
	wg := new(sync.WaitGroup)

	subscribeToSettingsEvents(settingsSvc, app, svc, repos)
	registerValidators(settingsSvc, svc, repos)

	if err := svc.chat.EnsureSystemRooms(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("ensure system chat rooms at startup")
	}

	uploadDir := svc.upload.GetUploadDir()

	scheduleJob(stop, wg, "clean orphaned uploads", "cleaned orphaned upload files", 24*time.Hour, func() (int, error) {
		return upload.CleanOrphanedFiles(repos.Upload, uploadDir), nil
	})
	scheduleJob(stop, wg, "prune old notifications", "pruned old notifications", 24*time.Hour, func() (int, error) {
		return svc.notification.PruneOld(context.Background())
	})
	scheduleJob(stop, wg, "clean expired sessions", "cleaned expired sessions", 24*time.Hour, func() (int, error) {
		return svc.session.CleanExpired(context.Background())
	})

	// the only job here that fills the cache rather than pruning; see cache.CrawlerRanges
	scheduleJob(stop, wg, "refresh crawler ranges", "refreshed crawler ranges", 24*time.Hour, func() (int, error) {
		return svc.crawlerFeeds.Refresh(context.Background())
	})
	scheduleJob(stop, wg, "archive stale journals", "archived stale journals", time.Hour, func() (int, error) {
		return svc.journal.ArchiveStale(context.Background())
	})
	scheduleJob(stop, wg, "archive stale chat rooms", "archived stale chat rooms", time.Hour, func() (int, error) {
		return svc.chat.ArchiveStale(context.Background())
	})
	scheduleJob(stop, wg, "cancel idle games", "cancelled idle games", 5*time.Minute, func() (int, error) {
		return svc.gameRoom.CancelIdleGames(context.Background())
	})
	scheduleJob(stop, wg, "reconcile voice presence", "reconciled voice presence", 30*time.Second, func() (int, error) {
		return svc.chat.ReconcilePresence(context.Background())
	})
	scheduleJob(stop, wg, "reconcile live streams", "reconciled live streams", time.Minute, func() (int, error) {
		return svc.stream.ReconcileOnce(context.Background())
	})

	return func(ctx context.Context) {
		close(stop)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			logger.Log.Info().Msg("background jobs stopped")
		case <-ctx.Done():
			logger.Log.Warn().Msg("background jobs did not stop in time")
		}
	}
}

func scheduleJob(stop <-chan struct{}, wg *sync.WaitGroup, name string, successMsg string, interval time.Duration, fn func() (int, error)) {
	logger.Log.Info().Str("interval", interval.String()).Msgf("registered job: %s", name)
	wg.Go(func() {
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

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				run()
			}
		}
	})
}

func registerValidators(settingsSvc settings.Service, svc *services, repos *repository.Repositories) {
	settingsSvc.RegisterValidator(config.SettingChatbotModel, chatbot.ModelValidator(svc.openai))
	settingsSvc.RegisterValidator(config.SettingChatbotOptInRole, chatbot.OptInRoleValidator(repos.VanityRole, repos.Permission))
	settingsSvc.RegisterValidator(config.SettingCrawlerFeeds, feed.Validator(svc.crawlerFeeds))
	settingsSvc.RegisterValidator(config.SettingValkeyURL, engines.ProbeURL)
}

func subscribeToSettingsEvents(settingsSvc settings.Service, app *fiber.App, svc *services, repos *repository.Repositories) {
	settingsSvc.Subscribe(logger.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewSettingsListener())
	settingsSvc.Subscribe(telemetry.NewProfilingSettingsListener())
	settingsSvc.Subscribe(middleware.NewBodyLimitListener(app))
	settingsSvc.Subscribe(svc.push)
	settingsSvc.Subscribe(chatbot.NewOptInRoleMigrator(repos.VanityRole, repos.AuditLog, settingsSvc))
	settingsSvc.Subscribe(svc.crawlerFeeds)
	settingsSvc.SubscribeBatch(svc.email)
	settingsSvc.SubscribeBatch(svc.chatbot)
	settingsSvc.SubscribeBatch(svc.openai)

	for _, candidate := range svc.cache.Engines() {
		if listener, ok := candidate.(settings.Listener); ok {
			settingsSvc.Subscribe(listener)
		}
	}
}
