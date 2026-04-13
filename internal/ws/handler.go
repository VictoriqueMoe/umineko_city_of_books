package ws

import (
	"context"
	"encoding/json"
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true
	},
}

type RoomLister interface {
	GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type incomingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type roomActionData struct {
	RoomID string `json:"room_id"`
}

type typingData struct {
	RoomID string `json:"room_id"`
}

type ChatMessageSender interface {
	SendChatMessage(ctx context.Context, senderID uuid.UUID, roomID uuid.UUID, body string) error
}

func Handler(hub *Hub, sessionMgr *session.Manager, roomLister RoomLister) fiber.Handler {
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

			if roomLister != nil {
				roomIDs, err := roomLister.GetRoomsByUser(context.Background(), userID)
				if err == nil {
					for _, roomID := range roomIDs {
						hub.JoinRoom(roomID, userID)
					}
				}
			}

			conn.SetPongHandler(func(string) error {
				return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
			})
			_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			go func() {
				for range ticker.C {
					if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
						return
					}
				}
			}()

			for {
				_, raw, err := conn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err,
						websocket.CloseNormalClosure,
						websocket.CloseGoingAway,
						websocket.CloseNoStatusReceived,
						websocket.CloseAbnormalClosure,
						websocket.CloseServiceRestart,
						websocket.CloseTryAgainLater,
						websocket.CloseTLSHandshake,
					) {
						logger.Log.Warn().Err(err).Str("user_id", userID.String()).Msg("unexpected ws close")
					}
					break
				}

				var msg incomingMessage
				if err := json.Unmarshal(raw, &msg); err != nil {
					continue
				}

				switch msg.Type {
				case "typing":
					var data typingData
					if err := json.Unmarshal(msg.Data, &data); err != nil {
						continue
					}
					roomID, err := uuid.Parse(data.RoomID)
					if err != nil {
						continue
					}
					hub.BroadcastToRoom(roomID, Message{
						Type: "typing",
						Data: map[string]interface{}{
							"room_id": data.RoomID,
							"user_id": userID.String(),
						},
					}, userID)

				case "join_room":
					var data roomActionData
					if err := json.Unmarshal(msg.Data, &data); err != nil {
						continue
					}
					roomID, err := uuid.Parse(data.RoomID)
					if err != nil {
						continue
					}
					hub.AddViewer(roomID, userID)

				case "leave_room":
					var data roomActionData
					if err := json.Unmarshal(msg.Data, &data); err != nil {
						continue
					}
					roomID, err := uuid.Parse(data.RoomID)
					if err != nil {
						continue
					}
					hub.RemoveViewer(roomID, userID)
				}
			}
		})
	}
}
