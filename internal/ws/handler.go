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
	"umineko_city_of_books/internal/secrets"
	"umineko_city_of_books/internal/session"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/time/rate"
)

const (
	maxInboundMessageSize = 8 * 1024
	wsTracerName          = "umineko_city_of_books/ws"
	localsUserIDKey       = "ws.user_id"
	localsTokenKey        = "ws.token"
	localsAnonKey         = "ws.anon"
	sessionRecheckEvery   = 5 * time.Minute
	readDeadline          = 90 * time.Second
	membershipCheckLimit  = 3 * time.Second

	inboundPerSecond = 100
	inboundBurst     = 200

	gameRoomInputType = "game_room_input"
	rateLimitLogEvery = 5 * time.Second
)

type (
	RoomLister interface {
		GetRoomsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	}

	RoomMembershipChecker interface {
		IsRoomMember(ctx context.Context, roomID, userID uuid.UUID) (bool, error)
	}

	BanChecker interface {
		IsBanned(ctx context.Context, userID uuid.UUID) bool
	}

	GameRoomPresence interface {
		HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID)
		HandleClientLeave(userID, roomID uuid.UUID)
		HandleClientInput(userID, roomID uuid.UUID, payload json.RawMessage)
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

	gameRoomInputData struct {
		RoomID  string          `json:"room_id"`
		Payload json.RawMessage `json:"payload"`
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
		Err(fmt.Errorf("panic: %v", r)).
		Str("user_id", userID).
		Bytes("stack", debug.Stack()).
		Msg("ws handler panic")

	_ = conn.WriteJSON(fiber.Map{"error": "internal error"})
}

func stillAuthorised(sessionMgr *session.Manager, banChecker BanChecker, token string, userID uuid.UUID) error {
	ctx := context.Background()

	current, err := sessionMgr.Validate(ctx, token)
	if err != nil {
		return fmt.Errorf("session no longer valid: %w", err)
	}
	if current != userID {
		return fmt.Errorf("session no longer belongs to this user")
	}

	if banChecker != nil && banChecker.IsBanned(ctx, userID) {
		return fmt.Errorf("account is banned")
	}

	return nil
}

func OriginAllowed(origin, allowed string) bool {
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
		Data: map[string]any{
			"room_id": roomID.String(),
			"user_id": userID.String(),
			"state":   state,
		},
	}, uuid.Nil)
}

func ensureRoomMembership(ctx context.Context, hub *Hub, memberChecker RoomMembershipChecker, roomID, userID uuid.UUID) bool {
	if hub.IsUserInRoom(roomID, userID) {
		return true
	}

	if memberChecker == nil {
		return false
	}

	checkCtx, cancel := context.WithTimeout(ctx, membershipCheckLimit)
	defer cancel()

	isMember, err := memberChecker.IsRoomMember(checkCtx, roomID, userID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("user_id", userID.String()).Str("room_id", roomID.String()).Msg("ws membership check failed")

		return false
	}
	if !isMember {
		return false
	}

	hub.JoinRoom(roomID, userID)

	return true
}

func Handler(hub *Hub, sessionMgr *session.Manager, banChecker BanChecker, roomLister RoomLister, memberChecker RoomMembershipChecker, gamePresence GameRoomPresence, watchPartyDisconnect WatchPartyDisconnectHandler, allowedOrigin func(ctx context.Context) string) fiber.Handler {
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

		token, _ := conn.Locals(localsTokenKey).(string)
		lastRecheck := time.Now()

		conn.SetPongHandler(func(string) error {
			if token != "" && time.Since(lastRecheck) >= sessionRecheckEvery {
				lastRecheck = time.Now()
				if err := stillAuthorised(sessionMgr, banChecker, token, userID); err != nil {
					logger.Log.Info().Str("user_id", userID.String()).Msg("ws session no longer valid, closing socket")

					return err
				}
			}

			return conn.SetReadDeadline(time.Now().Add(readDeadline))
		})
		_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

		limiter := rate.NewLimiter(inboundPerSecond, inboundBurst)
		var lastLimitLog time.Time

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

			tokens := limiter.Tokens()
			if !limiter.Allow() {
				recordDropped(true)

				if now := time.Now(); now.Sub(lastLimitLog) >= rateLimitLogEvery {
					lastLimitLog = now
					logger.Log.Warn().Str("user_id", userID.String()).Float64("tokens", tokens).Msg("ws inbound rate limit exceeded")
				}

				continue
			}

			var msg incomingMessage
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}

			if msg.Type == gameRoomInputType {
				if gamePresence != nil {
					var data gameRoomInputData
					if err := json.Unmarshal(msg.Data, &data); err == nil {
						roomID, perr := uuid.Parse(data.RoomID)
						if perr == nil && joinedGameRooms[roomID] {
							gamePresence.HandleClientInput(userID, roomID, data.Payload)
						}
					}
				}

				recordInputFrame()

				continue
			}

			recordInbound(msg.Type, tokens)

			handleWSMessage(client, msg, hub, memberChecker, gamePresence, joinedGameRooms)
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
			allowed = allowedOrigin(ctx.Context())
		}
		if !OriginAllowed(origin, allowed) {
			logger.Ctx(ctx.Context()).Warn().Str("origin", origin).Msg("ws upgrade rejected: origin not allowed")
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

		if banChecker != nil && banChecker.IsBanned(ctx.Context(), userID) {
			return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "account is banned",
			})
		}

		ctx.Locals(localsUserIDKey, userID)
		ctx.Locals(localsTokenKey, token)
		return wsHandler(ctx)
	}
}

func runAnonReader(hub *Hub, conn *websocket.Conn) {
	client := NewClient(uuid.Nil, conn)
	defer func() { _ = client.Close() }()

	hub.RegisterAnon(client)
	defer hub.UnregisterAnon(client)

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(readDeadline))
	})
	_ = conn.SetReadDeadline(time.Now().Add(readDeadline))

	limiter := rate.NewLimiter(inboundPerSecond, inboundBurst)

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}

		tokens := limiter.Tokens()
		if !limiter.Allow() {
			recordDropped(false)
			continue
		}

		var msg incomingMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		recordInbound(msg.Type, tokens)

		if msg.Type == "ping" {
			if data, marshalErr := json.Marshal(Message{Type: "pong", Data: map[string]any{}}); marshalErr == nil {
				client.enqueue(data)
			}
		}
	}
}

func handleWSMessage(client *Client, msg incomingMessage, hub *Hub, memberChecker RoomMembershipChecker, gamePresence GameRoomPresence, joinedGameRooms map[uuid.UUID]bool) {
	userID := client.UserID

	spanCtx, span := hub.tracer.Start(
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
	case TypingMessageType:
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
		hub.BroadcastToRoom(roomID, TypingMessage(roomID, userID), userID)

	case "join_room":
		var data roomActionData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		roomID, err := uuid.Parse(data.RoomID)
		if err != nil {
			return
		}
		if !ensureRoomMembership(spanCtx, hub, memberChecker, roomID, userID) {
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
		if _, known := secrets.Lookup(data.SecretID); !known {
			return
		}
		hub.JoinTopic("secret:"+data.SecretID, userID)

	case "secret_leave":
		var data secretTopicData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		if _, known := secrets.Lookup(data.SecretID); !known {
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
		if data, marshalErr := json.Marshal(Message{Type: "pong", Data: map[string]any{}}); marshalErr == nil {
			client.enqueue(data)
		}
	}
}
