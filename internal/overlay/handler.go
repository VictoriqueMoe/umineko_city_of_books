package overlay

import (
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/ws"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	localsUserIDKey       = "overlay.user_id"
	maxInboundMessageSize = 4 * 1024
	readDeadline          = 90 * time.Second
)

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

		conn.SetPongHandler(func(string) error {
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
		RecoverHandler:   func(*websocket.Conn) {},
		HandshakeTimeout: 10 * time.Second,
	})

	return func(ctx fiber.Ctx) error {
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

		ctx.Locals(localsUserIDKey, userID)
		return wsHandler(ctx)
	}
}
