package ws

import (
	"log"

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
						log.Printf("[ws] unexpected close for user %d: %v", userID, err)
					}
					break
				}
			}
		})
	}
}
