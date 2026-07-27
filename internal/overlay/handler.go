package overlay

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	localsUserIDKey       = "overlay.user_id"
	localsTokenKey        = "overlay.token"
	maxInboundMessageSize = 4 * 1024
	readDeadline          = 90 * time.Second
	tokenRecheckEvery     = 5 * time.Minute
)

func overlayRecoverHandler(conn *websocket.Conn) {
	r := recover()
	if r == nil {
		return
	}

	userID := ""
	if uid, ok := conn.Locals(localsUserIDKey).(uuid.UUID); ok {
		userID = uid.String()
	}

	logger.Log.Error().
		Str("user_id", userID).
		Str("panic", fmt.Sprintf("%v", r)).
		Bytes("stack", debug.Stack()).
		Msg("overlay ws handler panic")
}

func (s *service) stillAuthorised(token string, userID uuid.UUID) error {
	ctx := context.Background()

	current, err := s.Validate(ctx, token)
	if err != nil {
		return fmt.Errorf("overlay token no longer valid: %w", err)
	}
	if current != userID {
		return fmt.Errorf("overlay token no longer belongs to this user")
	}

	if s.banChecker != nil && s.banChecker.IsBanned(ctx, userID) {
		return fmt.Errorf("account is banned")
	}

	return nil
}

func (s *service) Handler() fiber.Handler {
	wsHandler := websocket.New(func(conn *websocket.Conn) {
		userID, ok := conn.Locals(localsUserIDKey).(uuid.UUID)
		if !ok {
			return
		}

		conn.SetReadLimit(maxInboundMessageSize)
		client := ws.NewClient(userID, conn)
		s.hub.Register(client)
		defer s.hub.Unregister(client)

		token, _ := conn.Locals(localsTokenKey).(string)
		lastRecheck := time.Now()

		conn.SetPongHandler(func(string) error {
			if token != "" && time.Since(lastRecheck) >= tokenRecheckEvery {
				lastRecheck = time.Now()
				if err := s.stillAuthorised(token, userID); err != nil {
					logger.Log.Info().Str("user_id", userID.String()).Msg("overlay token no longer valid, closing socket")

					return err
				}
			}

			return conn.SetReadDeadline(time.Now().Add(readDeadline))
		})
		_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}, websocket.Config{
		Origins:          []string{"*"},
		RecoverHandler:   overlayRecoverHandler,
		HandshakeTimeout: 10 * time.Second,
	})

	return func(ctx fiber.Ctx) error {
		origin := ctx.Get("Origin")
		if origin != "" && !ws.OriginAllowed(origin, s.settingsSvc.Get(ctx.Context(), config.SettingBaseURL)) {
			logger.Log.Warn().Str("origin", origin).Msg("overlay ws upgrade rejected: origin not allowed")

			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "origin not allowed"})
		}

		token := ctx.Query("token")
		if token == "" {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		userID, err := s.Validate(ctx.Context(), token)
		if err != nil {
			logger.Log.Warn().Err(err).Msg("overlay token validation failed")
			return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
		}
		if userID == uuid.Nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		if s.banChecker != nil && s.banChecker.IsBanned(ctx.Context(), userID) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account is banned"})
		}

		ctx.Locals(localsUserIDKey, userID)
		ctx.Locals(localsTokenKey, token)
		return wsHandler(ctx)
	}
}
