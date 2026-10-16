package middleware

import (
	"context"
	"database/sql"
	"time"

	"umineko_city_of_books/internal/cache"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	IPWriter interface {
		UpdateIP(ctx context.Context, s spec.UserIPUpdate, tx ...*sql.Tx) error
	}

	LastSeenIP struct {
		repo     IPWriter
		cache    *cache.Manager
		debounce time.Duration
	}
)

func NewLastSeenIP(repo IPWriter, cacheManager *cache.Manager, debounce time.Duration) *LastSeenIP {
	return &LastSeenIP{repo: repo, cache: cacheManager, debounce: debounce}
}

func (l *LastSeenIP) Record(ctx context.Context, userID uuid.UUID, ip string) {
	if ip == "" || userID == uuid.Nil {
		return
	}

	claimed, err := l.cache.SetIfAbsent(ctx, cache.LastSeenIP.Key(userID.String(), ip), ip, l.debounce)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("user_id", userID.String()).Msg("last seen ip debounce store failed, writing the ip anyway")
	}

	if err == nil && !claimed {
		return
	}

	writeCtx := context.WithoutCancel(ctx)
	go func() {
		err := l.repo.UpdateIP(writeCtx, spec.UserIPUpdate{UserID: userID, IP: ip})
		if err != nil {
			logger.Ctx(writeCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("update last seen ip failed")
		}
	}()
}

func RecordLastSeenIP(recorder *LastSeenIP) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		err := ctx.Next()
		if recorder == nil {
			return err
		}

		userID, ok := ctx.Locals("userID").(uuid.UUID)
		if !ok || userID == uuid.Nil {
			return err
		}

		ip, _ := ctx.Locals("client_ip").(string)
		recorder.Record(ctx.Context(), userID, ip)

		return err
	}
}
