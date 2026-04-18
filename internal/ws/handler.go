package ws

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

const (
	maxInboundMessageSize = 8 * 1024
)

type (
	RoomLister interface {
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	}

	ChatMessageSender interface {
		SendChatMessage(ctx context.Context, senderID uuid.UUID, roomID uuid.UUID, body string) error
	}

	incomingMessage struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}

	roomActionData struct {
		RoomID string `json:"room_id"`
	}

	viewerStateData struct {
		RoomID string `json:"room_id"`
		State  string `json:"state"`
	}

	typingData struct {
		RoomID string `json:"room_id"`
	}
)

func originAllowed(origin, allowed string) bool {
	if origin == "" || allowed == "" {
		return false
	}
	return origin == strings.TrimSuffix(allowed, "/")
}

func broadcastPresence(hub *Hub, roomID, userID uuid.UUID, state string) {
	hub.BroadcastToRoom(roomID, Message{
		Type: "chat_presence_changed",
		Data: map[string]interface{}{
			"room_id": roomID.String(),
			"user_id": userID.String(),
			"state":   state,
		},
	}, uuid.Nil)
}

func Handler(hub *Hub, sessionMgr *session.Manager, roomLister RoomLister, allowedOrigin func() string) fiber.Handler {
	upgrader := websocket.FastHTTPUpgrader{
		CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
			origin := string(ctx.Request.Header.Peek("Origin"))
			allowed := ""
			if allowedOrigin != nil {
				allowed = allowedOrigin()
			}
			if originAllowed(origin, allowed) {
				return true
			}
			logger.Log.Warn().Str("origin", origin).Msg("ws upgrade rejected: origin not allowed")
			return false
		},
	}

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
			conn.SetReadLimit(maxInboundMessageSize)
			client := NewClient(userID, conn)

			hub.Register(client)
			defer func() {
				cleared := hub.Unregister(client)
				for _, roomID := range cleared {
					broadcastPresence(hub, roomID, userID, "")
				}
			}()

			if roomLister != nil {
				roomIDs, err := roomLister.GetRoomsByUser(ctx.Context(), userID)
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
					if !hub.IsUserInRoom(roomID, userID) {
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
					if !hub.IsUserInRoom(roomID, userID) {
						continue
					}
					hub.AddViewer(roomID, userID)
					broadcastPresence(hub, roomID, userID, ViewerStateActive)

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
					if !hub.IsUserViewing(roomID, userID) {
						broadcastPresence(hub, roomID, userID, "")
					}

				case "viewer_state":
					var data viewerStateData
					if err := json.Unmarshal(msg.Data, &data); err != nil {
						continue
					}
					roomID, err := uuid.Parse(data.RoomID)
					if err != nil {
						continue
					}
					if !hub.IsUserInRoom(roomID, userID) {
						continue
					}
					if hub.SetViewerState(roomID, userID, data.State) {
						broadcastPresence(hub, roomID, userID, data.State)
					}
				}
			}
		})
	}
}
