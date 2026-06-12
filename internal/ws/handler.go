package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	maxInboundMessageSize = 8 * 1024
	wsTracerName          = "umineko_city_of_books/ws"
	localsUserIDKey       = "ws.user_id"
	localsAnonKey         = "ws.anon"
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

	WatchPartyDisconnectHandler interface {
		HandleClientDisconnect(ctx context.Context, userID uuid.UUID, roomIDs []uuid.UUID)
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

func recoverHandler(conn *websocket.Conn) {
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
		Msg("ws handler panic")

	_ = conn.WriteJSON(fiber.Map{"error": "internal error"})
}

func originAllowed(origin, allowed string) bool {
	if origin == "" {
		return false
	}

	if allowed != "" && origin == strings.TrimSuffix(allowed, "/") {
		return true
	}

	return config.IsAppOrigin(origin)
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

func Handler(hub *Hub, sessionMgr *session.Manager, roomLister RoomLister, gamePresence GameRoomPresence, watchPartyDisconnect WatchPartyDisconnectHandler, allowedOrigin func() string) fiber.Handler {
	wsHandler := websocket.New(func(conn *websocket.Conn) {
		if anon, _ := conn.Locals(localsAnonKey).(bool); anon {
			runAnonReader(hub, conn)
			return
		}

		userID, ok := conn.Locals(localsUserIDKey).(uuid.UUID)
		if !ok {
			return
		}

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
			if watchPartyDisconnect != nil && len(cleared) > 0 {
				watchPartyDisconnect.HandleClientDisconnect(context.Background(), userID, cleared)
			}
		}()

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
	}, websocket.Config{
		Origins:           []string{"*"},
		RecoverHandler:    recoverHandler,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
	})

	return func(ctx fiber.Ctx) error {
		origin := ctx.Get("Origin")
		allowed := ""
		if allowedOrigin != nil {
			allowed = allowedOrigin()
		}
		if !originAllowed(origin, allowed) {
			logger.Log.Warn().Str("origin", origin).Msg("ws upgrade rejected: origin not allowed")
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "origin not allowed",
			})
		}

		token := ctx.Cookies(session.CookieName)
		if token == "" {
			token = ctx.Query("token")
		}
		if token == "" {
			ctx.Locals(localsAnonKey, true)
			return wsHandler(ctx)
		}

		userID, err := sessionMgr.Validate(ctx.Context(), token)
		if err != nil {
			return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired session",
			})
		}

		ctx.Locals(localsUserIDKey, userID)
		return wsHandler(ctx)
	}
}

func runAnonReader(hub *Hub, conn *websocket.Conn) {
	client := NewClient(uuid.Nil, conn)
	defer func() { _ = client.Close() }()

	hub.RegisterAnon(client)
	defer hub.UnregisterAnon(client)

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	})
	_ = conn.SetReadDeadline(time.Now().Add(90 * time.Second))

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg incomingMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		if msg.Type == "ping" {
			if data, marshalErr := json.Marshal(Message{Type: "pong", Data: map[string]interface{}{}}); marshalErr == nil {
				client.enqueue(data)
			}
		}
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
		if !joinedGameRooms[roomID] {
			return
		}
		gamePresence.HandleClientLeave(userID, roomID)
		delete(joinedGameRooms, roomID)

	case "ping":
		hub.SendToUser(userID, Message{Type: "pong", Data: map[string]interface{}{}})
	}
}
