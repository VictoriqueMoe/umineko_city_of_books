package middleware

import (
	"context"
	"sync"
	"time"

	"umineko_city_of_books/internal/logger"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type IPWriter interface {
	UpdateIP(ctx context.Context, userID uuid.UUID, ip string) error
}

type lastSeenEntry struct {
	ip        string
	lastWrote time.Time
}

type LastSeenIP struct {
	repo     IPWriter
	debounce time.Duration
	cache    sync.Map
}

func NewLastSeenIP(repo IPWriter, debounce time.Duration) *LastSeenIP {
	return &LastSeenIP{repo: repo, debounce: debounce}
}

func (l *LastSeenIP) Record(userID uuid.UUID, ip string) {
	if ip == "" || userID == uuid.Nil {
		return
	}
	now := time.Now()
	if v, ok := l.cache.Load(userID); ok {
		entry := v.(lastSeenEntry)
		if entry.ip == ip && now.Sub(entry.lastWrote) < l.debounce {
			return
		}
	}
	l.cache.Store(userID, lastSeenEntry{ip: ip, lastWrote: now})
	go func() {
		if err := l.repo.UpdateIP(context.Background(), userID, ip); err != nil {
			logger.Log.Warn().Err(err).Str("user_id", userID.String()).Msg("update last seen ip failed")
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
		recorder.Record(userID, ip)
		return err
	}
}
