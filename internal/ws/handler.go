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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxInboundMessageSize = 8 * 1024
	wsTracerName          = "umineko_city_of_books/ws"
)

type (
	RoomLister interface {
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	}

	ChatMessageSender interface {
		SendChatMessage(ctx context.Context, senderID uuid.UUID, roomID uuid.UUID, body string) error
	}

	GameRoomPresence interface {
		HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID)
		HandleClientLeave(userID, roomID uuid.UUID)
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

	secretTopicData struct {
		SecretID string `json:"secret_id"`
	}

	gameRoomTopicData struct {
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

func Handler(hub *Hub, sessionMgr *session.Manager, roomLister RoomLister, gamePresence GameRoomPresence, allowedOrigin func() string) fiber.Handler {
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
			joinedGameRooms := make(map[uuid.UUID]bool)
			defer func() {
				cleared := hub.Unregister(client)
				for _, roomID := range cleared {
					broadcastPresence(hub, roomID, userID, "")
				}
				if gamePresence != nil {
					for roomID := range joinedGameRooms {
						gamePresence.HandleClientLeave(userID, roomID)
					}
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

				handleWSMessage(userID, msg, hub, gamePresence, joinedGameRooms)
			}
		})
	}
}

func handleWSMessage(userID uuid.UUID, msg incomingMessage, hub *Hub, gamePresence GameRoomPresence, joinedGameRooms map[uuid.UUID]bool) {
	spanCtx, span := otel.Tracer(wsTracerName).Start(
		context.Background(),
		"ws."+msg.Type,
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("ws.user_id", userID.String()),
			attribute.String("ws.message_type", msg.Type),
		),
	)
	defer span.End()

	switch msg.Type {
	case "typing":
		var data typingData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		if !hub.IsUserInRoom(roomID, userID) {
			return
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
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		if !hub.IsUserInRoom(roomID, userID) {
			return
		}
		hub.AddViewer(roomID, userID)
		broadcastPresence(hub, roomID, userID, ViewerStateActive)

	case "leave_room":
		var data roomActionData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		hub.RemoveViewer(roomID, userID)
		if !hub.IsUserViewing(roomID, userID) {
			broadcastPresence(hub, roomID, userID, "")
		}

	case "viewer_state":
		var data viewerStateData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		if !hub.IsUserInRoom(roomID, userID) {
			return
		}
		if hub.SetViewerState(roomID, userID, data.State) {
			broadcastPresence(hub, roomID, userID, data.State)
		}

	case "secret_join":
		var data secretTopicData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		if data.SecretID == "" {
			return
		}
		hub.JoinTopic("secret:"+data.SecretID, userID)

	case "secret_leave":
		var data secretTopicData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		if data.SecretID == "" {
			return
		}
		hub.LeaveTopic("secret:"+data.SecretID, userID)

	case "game_room_join":
		if gamePresence == nil {
			return
		}
		var data gameRoomTopicData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		gamePresence.HandleClientJoin(spanCtx, userID, roomID)
		joinedGameRooms[roomID] = true

	case "game_room_leave":
		if gamePresence == nil {
			return
		}
		var data gameRoomTopicData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		gamePresence.HandleClientLeave(userID, roomID)
		delete(joinedGameRooms, roomID)
	}
}
