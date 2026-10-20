package middleware

import (
	"context"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/user"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type (
	LastSeenIP struct {
		userSvc user.Service
	}
)

func NewLastSeenIP(userSvc user.Service) *LastSeenIP {
	return &LastSeenIP{userSvc: userSvc}
}

func (l *LastSeenIP) Record(ctx context.Context, userID uuid.UUID, ip string) {
	if ip == "" || userID == uuid.Nil {
		return
	}

	recordCtx := context.WithoutCancel(ctx)
	go func() {
		if err := l.userSvc.UpdateIP(recordCtx, userID, ip); err != nil {
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
