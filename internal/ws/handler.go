package ws

import (
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

func Handler(hub *Hub, sessionMgr *session.Manager) fiber.Handler {
	return func(ctx fiber.Ctx) error {
		cookie := ctx.Cookies(session.CookieName)
		if cookie == "" {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		userID, err := sessionMgr.Validate(ctx.Context(), cookie)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired session",
			})
		}

		return upgrader.Upgrade(ctx.RequestCtx(), func(conn *websocket.Conn) {
			logger.Log.Debug().Str("user_id", userID.String()).Msg("ws client connected")
			client := &Client{
				UserID: userID,
				Conn:   conn,
			}

			hub.Register(client)
			defer hub.Unregister(client)
			defer conn.Close()

			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
						logger.Log.Warn().Err(err).Str("user_id", userID.String()).Msg("unexpected ws close")
					}
					break
				}
			}
		})
	}
}
