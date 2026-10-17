package middleware

import (
	"context"
	"database/sql"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	IPStore interface {
		GetIP(ctx context.Context, userID uuid.UUID, tx ...*sql.Tx) (string, error)
		UpdateIP(ctx context.Context, s spec.UserIPUpdate, tx ...*sql.Tx) error
	}

	LastSeenIP struct {
		repo IPStore
	}
)

func NewLastSeenIP(repo IPStore) *LastSeenIP {
	return &LastSeenIP{repo: repo}
}

func (l *LastSeenIP) Record(ctx context.Context, userID uuid.UUID, ip string) {
	if ip == "" || userID == uuid.Nil {
		return
	}

	recordCtx := context.WithoutCancel(ctx)
	go func() {
		current, err := l.repo.GetIP(recordCtx, userID)
		if err != nil {
			logger.Ctx(recordCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("read last seen ip failed")
			return
		}

		if current == ip {
			return
		}

		err = l.repo.UpdateIP(recordCtx, spec.UserIPUpdate{UserID: userID, IP: ip})
		if err != nil {
			logger.Ctx(recordCtx).Warn().Err(err).Str("user_id", userID.String()).Msg("update last seen ip failed")
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
