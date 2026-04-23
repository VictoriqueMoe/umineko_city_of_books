package gameroom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/repository"
	"umineko_city_of_books/internal/repository/model"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	disconnectGracePeriod = 60 * time.Second
	idleGameTimeout       = 10 * time.Minute
	maxChatMessages       = 200
	maxChatBodyLen        = 500
)

type (
	Notifier interface {
		Notify(ctx context.Context, params dto.NotifyParams) error
	}

	ListFilter struct {
		GameType dto.GameType
		Statuses []dto.GameStatus
		Limit    int
		Offset   int
	}

	Service interface {
		Invite(ctx context.Context, inviterID, opponentID uuid.UUID, gameType dto.GameType) (*dto.GameRoom, error)
		Accept(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		Decline(ctx context.Context, roomID, userID uuid.UUID) error
		Cancel(ctx context.Context, roomID, userID uuid.UUID) error
		Get(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.GameRoom, error)
		List(ctx context.Context, userID uuid.UUID, filter ListFilter) (*dto.GameRoomListResponse, error)
		ListLive(ctx context.Context, gameType dto.GameType, limit, offset int) (*dto.GameRoomListResponse, error)
		ListFinished(ctx context.Context, gameType dto.GameType, limit, offset int) (*dto.GameRoomListResponse, error)
		CountLive(ctx context.Context) (int, error)
		SubmitAction(ctx context.Context, roomID, userID uuid.UUID, action json.RawMessage) (*dto.GameRoom, error)
		Resign(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		Scoreboard(ctx context.Context, gameType dto.GameType) (*dto.GameScoreboardResponse, error)
		PostSpectatorChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error)
		GetSpectatorChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error)
		PostPlayerChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error)
		GetPlayerChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error)
		HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID)
		HandleClientLeave(userID, roomID uuid.UUID)
		CancelIdleGames(ctx context.Context) (int, error)
	}

	roomState struct {
		players        map[uuid.UUID]int
		spectators     map[uuid.UUID]int
		timers         map[uuid.UUID]*time.Timer
		disconnectedAt map[uuid.UUID]time.Time
		chat           []dto.SpectatorMessage
		playerChat     []dto.SpectatorMessage
	}

	service struct {
		repo          repository.GameRoomRepository
		userRepo      repository.UserRepository
		blockSvc      repository.BlockRepository
		notifSvc      Notifier
		hub           *ws.Hub
		contentFilter *contentfilter.Manager
		handlers      map[dto.GameType]GameHandler

		mu    sync.Mutex
		rooms map[uuid.UUID]*roomState
	}
)

func NewService(
	repo repository.GameRoomRepository,
	userRepo repository.UserRepository,
	blockRepo repository.BlockRepository,
	notifSvc Notifier,
	hub *ws.Hub,
	contentFilter *contentfilter.Manager,
	handlers []GameHandler,
) Service {
	m := make(map[dto.GameType]GameHandler, len(handlers))
	for _, h := range handlers {
		m[h.GameType()] = h
	}
	return &service{
		repo:          repo,
		userRepo:      userRepo,
		blockSvc:      blockRepo,
		notifSvc:      notifSvc,
		hub:           hub,
		contentFilter: contentFilter,
		handlers:      m,
		rooms:         make(map[uuid.UUID]*roomState),
	}
}

func (s *service) stateFor(roomID uuid.UUID) *roomState {
	st, ok := s.rooms[roomID]
	if !ok {
		st = &roomState{
			players:        make(map[uuid.UUID]int),
			spectators:     make(map[uuid.UUID]int),
			timers:         make(map[uuid.UUID]*time.Timer),
			disconnectedAt: make(map[uuid.UUID]time.Time),
		}
		s.rooms[roomID] = st
	}
	return st
}

func (s *service) Invite(ctx context.Context, inviterID, opponentID uuid.UUID, gameType dto.GameType) (*dto.GameRoom, error) {
	if inviterID == opponentID {
		return nil, ErrSelfInvite
	}
	if _, ok := s.handlers[gameType]; !ok {
		return nil, ErrUnknownGameType
	}

	opponent, err := s.userRepo.GetByID(ctx, opponentID)
	if err != nil || opponent == nil {
		return nil, ErrOpponentInactive
	}
	blocked, err := s.blockSvc.IsBlockedEither(ctx, inviterID, opponentID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrOpponentBlocked
	}

	inviter, err := s.userRepo.GetByID(ctx, inviterID)
	if err != nil || inviter == nil {
		return nil, fmt.Errorf("inviter not found")
	}

	roomID := uuid.New()
	if err := s.repo.CreateRoom(ctx, roomID, string(gameType), "{}", inviterID); err != nil {
		return nil, err
	}
	if err := s.repo.AddPlayer(ctx, roomID, inviterID, 0, true); err != nil {
		return nil, err
	}
	if err := s.repo.AddPlayer(ctx, roomID, opponentID, 1, false); err != nil {
		return nil, err
	}

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if s.notifSvc != nil {
		_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
			RecipientID:   opponentID,
			Type:          dto.NotifGameInvite,
			ReferenceID:   roomID,
			ReferenceType: string(gameType),
			ActorID:       inviterID,
			Message:       inviter.DisplayName + " invited you to a " + string(gameType) + " game",
		})
	}

	return room, nil
}

func (s *service) Accept(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status != string(dto.GameStatusPending) {
		return nil, ErrRoomNotPending
	}

	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return nil, ErrNotParticipant
	}
	if slot != 1 {
		return nil, ErrNotInvitee
	}

	handler, ok := s.handlers[dto.GameType(row.GameType)]
	if !ok {
		return nil, ErrUnknownGameType
	}

	if err := s.repo.SetPlayerJoined(ctx, roomID, userID); err != nil {
		return nil, err
	}

	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}

	stateJSON, firstTurnSlot, err := handler.InitialState(roomID, players)
	if err != nil {
		return nil, err
	}

	var firstTurnUser *uuid.UUID
	for i := range players {
		if players[i].Slot == firstTurnSlot {
			uid := players[i].UserID
			firstTurnUser = &uid
			break
		}
	}

	if err := s.repo.SetState(ctx, roomID, stateJSON, firstTurnUser); err != nil {
		return nil, err
	}
	if err := s.repo.SetStatus(ctx, roomID, string(dto.GameStatusActive)); err != nil {
		return nil, err
	}

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	s.broadcast(room, "game_room_started", nil)
	s.notifyTurn(ctx, room)
	s.broadcastLiveGamesCount(ctx)

	return room, nil
}

func (s *service) Cancel(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNotFound
	}
	if row.Status != string(dto.GameStatusPending) {
		return ErrRoomNotPending
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return ErrNotParticipant
	}
	if slot != 0 {
		return ErrNotInviter
	}
	if err := s.repo.SetStatus(ctx, roomID, string(dto.GameStatusDeclined)); err != nil {
		return err
	}
	room, _ := s.loadRoom(ctx, roomID)
	if room != nil {
		s.broadcast(room, "game_room_cancelled", map[string]any{"by_user_id": userID.String()})
	}
	return nil
}

func (s *service) Decline(ctx context.Context, roomID, userID uuid.UUID) error {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrNotFound
	}
	if row.Status != string(dto.GameStatusPending) {
		return ErrRoomNotPending
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return ErrNotParticipant
	}
	if slot != 1 {
		return ErrNotInvitee
	}
	if err := s.repo.SetStatus(ctx, roomID, string(dto.GameStatusDeclined)); err != nil {
		return err
	}

	room, _ := s.loadRoom(ctx, roomID)
	if room != nil {
		s.broadcast(room, "game_room_declined", map[string]any{"by_user_id": userID.String()})
	}
	return nil
}

func (s *service) Get(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.GameRoom, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status == string(dto.GameStatusPending) || row.Status == string(dto.GameStatusDeclined) {
		if viewerID == uuid.Nil {
			return nil, ErrNotParticipant
		}
		isParticipant, err := s.repo.IsParticipant(ctx, roomID, viewerID)
		if err != nil {
			return nil, err
		}
		if !isParticipant {
			return nil, ErrNotParticipant
		}
	}
	return s.loadRoom(ctx, roomID)
}

func (s *service) List(ctx context.Context, userID uuid.UUID, filter ListFilter) (*dto.GameRoomListResponse, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, total, err := s.repo.ListForUser(ctx, userID, string(filter.GameType), filter.Statuses, limit, filter.Offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.GameRoom, 0, len(rows))
	for _, row := range rows {
		r, err := s.hydrateRoom(ctx, &row)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return &dto.GameRoomListResponse{Rooms: out, Total: total}, nil
}

func (s *service) ListLive(ctx context.Context, gameType dto.GameType, limit, offset int) (*dto.GameRoomListResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, total, err := s.repo.ListLive(ctx, string(gameType), limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.GameRoom, 0, len(rows))
	for _, row := range rows {
		r, err := s.hydrateRoom(ctx, &row)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return &dto.GameRoomListResponse{Rooms: out, Total: total}, nil
}

func (s *service) CountLive(ctx context.Context) (int, error) {
	return s.repo.CountLive(ctx)
}

func (s *service) broadcastLiveGamesCount(ctx context.Context) {
	count, err := s.repo.CountLive(ctx)
	if err != nil {
		logger.Log.Warn().Err(err).Msg("count live games for broadcast")
		return
	}
	s.hub.Broadcast(ws.Message{
		Type: "live_games_count",
		Data: map[string]any{"count": count},
	})
}

func (s *service) ListFinished(ctx context.Context, gameType dto.GameType, limit, offset int) (*dto.GameRoomListResponse, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, total, err := s.repo.ListFinished(ctx, string(gameType), limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]dto.GameRoom, 0, len(rows))
	for _, row := range rows {
		r, err := s.hydrateRoom(ctx, &row)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return &dto.GameRoomListResponse{Rooms: out, Total: total}, nil
}

func (s *service) SubmitAction(ctx context.Context, roomID, userID uuid.UUID, action json.RawMessage) (*dto.GameRoom, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status != string(dto.GameStatusActive) {
		return nil, ErrRoomNotActive
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return nil, ErrNotParticipant
	}
	if row.TurnUserID == nil || *row.TurnUserID != userID {
		return nil, ErrNotYourTurn
	}
	handler, ok := s.handlers[dto.GameType(row.GameType)]
	if !ok {
		return nil, ErrUnknownGameType
	}

	s.clearForfeit(roomID, userID)

	result, err := handler.ValidateAction(row.StateJSON, slot, action)
	if err != nil {
		return nil, err
	}

	ply, err := s.repo.NextPly(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AppendMove(ctx, roomID, ply, userID, string(action)); err != nil {
		return nil, err
	}

	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}

	if result.Finished {
		winner := winnerUserID(result.WinnerSlot, players)
		if err := s.repo.FinishRoom(ctx, roomID, string(dto.GameStatusFinished), winner, result.Result, result.NewStateJSON); err != nil {
			return nil, err
		}
		room, err := s.loadRoom(ctx, roomID)
		if err != nil {
			return nil, err
		}
		s.broadcast(room, "game_room_finished", nil)
		s.notifyFinished(ctx, room, userID)
		s.broadcastLiveGamesCount(ctx)
		return room, nil
	}

	var nextTurn *uuid.UUID
	if result.NextTurnSlot != nil {
		for i := range players {
			if players[i].Slot == *result.NextTurnSlot {
				uid := players[i].UserID
				nextTurn = &uid
				break
			}
		}
	}
	if err := s.repo.SetState(ctx, roomID, result.NewStateJSON, nextTurn); err != nil {
		return nil, err
	}

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	s.broadcast(room, "game_room_action", map[string]any{
		"ply":     ply,
		"by_slot": slot,
		"action":  action,
	})
	s.notifyTurn(ctx, room)
	return room, nil
}

func (s *service) Resign(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status != string(dto.GameStatusActive) {
		return nil, ErrRoomNotActive
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return nil, ErrNotParticipant
	}
	s.clearForfeit(roomID, userID)
	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	var winner *uuid.UUID
	for i := range players {
		if players[i].Slot != slot {
			uid := players[i].UserID
			winner = &uid
			break
		}
	}
	result := "resign"
	if err := s.repo.FinishRoom(ctx, roomID, string(dto.GameStatusFinished), winner, result, row.StateJSON); err != nil {
		return nil, err
	}
	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	s.broadcast(room, "game_room_finished", map[string]any{"resigned_by": userID.String()})
	s.notifyFinished(ctx, room, userID)
	s.broadcastLiveGamesCount(ctx)
	return room, nil
}

func (s *service) Scoreboard(ctx context.Context, gameType dto.GameType) (*dto.GameScoreboardResponse, error) {
	if _, ok := s.handlers[gameType]; !ok {
		return nil, ErrUnknownGameType
	}
	rows, err := s.repo.Scoreboard(ctx, string(gameType))
	if err != nil {
		return nil, err
	}
	out := make([]dto.GameScoreboardRow, 0, len(rows))
	for _, r := range rows {
		u, err := s.userRepo.GetByID(ctx, r.UserID)
		if err != nil || u == nil {
			continue
		}
		games := r.Wins + r.Losses + r.Draws
		var winRate float64
		if games > 0 {
			winRate = float64(r.Wins) / float64(games)
		}
		out = append(out, dto.GameScoreboardRow{
			User:        *u.ToResponse(),
			Wins:        r.Wins,
			Losses:      r.Losses,
			Draws:       r.Draws,
			GamesPlayed: games,
			WinRate:     winRate,
		})
	}
	return &dto.GameScoreboardResponse{GameType: gameType, Rows: out}, nil
}

func (s *service) PostSpectatorChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrEmptyChat
	}
	if len(body) > maxChatBodyLen {
		body = body[:maxChatBodyLen]
	}
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status == string(dto.GameStatusPending) || row.Status == string(dto.GameStatusDeclined) {
		return nil, ErrRoomNotActive
	}
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if isParticipant {
		return nil, ErrPlayersCantChat
	}
	if s.contentFilter != nil {
		if err := s.contentFilter.Check(ctx, body); err != nil {
			return nil, err
		}
	}
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrOpponentInactive
	}
	msg := dto.SpectatorMessage{
		ID:        uuid.NewString(),
		UserID:    userID,
		User:      *u.ToResponse(),
		Body:      body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	st := s.stateFor(roomID)
	st.chat = append(st.chat, msg)
	if len(st.chat) > maxChatMessages {
		st.chat = st.chat[len(st.chat)-maxChatMessages:]
	}
	s.mu.Unlock()

	s.hub.BroadcastToTopic("spectator-chat:"+roomID.String(), ws.Message{
		Type: "spectator_chat_message",
		Data: map[string]any{
			"room_id": roomID.String(),
			"message": msg,
		},
	})
	return &msg, nil
}

func (s *service) GetSpectatorChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error) {
	if viewerID != uuid.Nil {
		isParticipant, err := s.repo.IsParticipant(ctx, roomID, viewerID)
		if err != nil {
			return nil, err
		}
		if isParticipant {
			return nil, ErrPlayersCantChat
		}
	}
	s.mu.Lock()
	st, ok := s.rooms[roomID]
	var msgs []dto.SpectatorMessage
	if ok {
		msgs = make([]dto.SpectatorMessage, len(st.chat))
		copy(msgs, st.chat)
	}
	s.mu.Unlock()
	return &dto.SpectatorChatResponse{Messages: msgs}, nil
}

func (s *service) PostPlayerChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrEmptyChat
	}
	if len(body) > maxChatBodyLen {
		body = body[:maxChatBodyLen]
	}
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	if row.Status == string(dto.GameStatusPending) || row.Status == string(dto.GameStatusDeclined) {
		return nil, ErrRoomNotActive
	}
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}
	if s.contentFilter != nil {
		if err := s.contentFilter.Check(ctx, body); err != nil {
			return nil, err
		}
	}
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, ErrOpponentInactive
	}
	msg := dto.SpectatorMessage{
		ID:        uuid.NewString(),
		UserID:    userID,
		User:      *u.ToResponse(),
		Body:      body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	st := s.stateFor(roomID)
	st.playerChat = append(st.playerChat, msg)
	if len(st.playerChat) > maxChatMessages {
		st.playerChat = st.playerChat[len(st.playerChat)-maxChatMessages:]
	}
	s.mu.Unlock()

	players, err := s.loadPlayers(ctx, roomID)
	if err == nil {
		for _, p := range players {
			s.hub.SendToUser(p.UserID, ws.Message{
				Type: "player_chat_message",
				Data: map[string]any{
					"room_id": roomID.String(),
					"message": msg,
				},
			})
		}
	}
	return &msg, nil
}

func (s *service) GetPlayerChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error) {
	if viewerID == uuid.Nil {
		return nil, ErrNotParticipant
	}
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, viewerID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotParticipant
	}
	s.mu.Lock()
	st, ok := s.rooms[roomID]
	var msgs []dto.SpectatorMessage
	if ok {
		msgs = make([]dto.SpectatorMessage, len(st.playerChat))
		copy(msgs, st.playerChat)
	}
	s.mu.Unlock()
	return &dto.SpectatorChatResponse{Messages: msgs}, nil
}

func (s *service) HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID) {
	if userID == uuid.Nil {
		return
	}
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil || row == nil {
		return
	}
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return
	}

	gameTopic := "game-room:" + roomID.String()
	s.hub.JoinTopic(gameTopic, userID)

	if isParticipant {
		s.mu.Lock()
		st := s.stateFor(roomID)
		timerCleared := false
		if t := st.timers[userID]; t != nil {
			t.Stop()
			delete(st.timers, userID)
			timerCleared = true
		}
		delete(st.disconnectedAt, userID)
		st.players[userID]++
		s.mu.Unlock()
		_ = s.repo.TouchPlayerSeen(ctx, roomID, userID)
		if timerCleared {
			s.hub.SendToUser(userID, ws.Message{
				Type: "game_forfeit_cleared",
				Data: map[string]any{"room_id": roomID.String()},
			})
		}
	} else {
		if row.Status == string(dto.GameStatusPending) || row.Status == string(dto.GameStatusDeclined) {
			s.hub.LeaveTopic(gameTopic, userID)
			return
		}
		s.hub.JoinTopic("spectator-chat:"+roomID.String(), userID)
		s.mu.Lock()
		st := s.stateFor(roomID)
		st.spectators[userID]++
		s.mu.Unlock()
	}

	s.broadcastPresence(roomID, userID, true, isParticipant)
}

func (s *service) HandleClientLeave(userID, roomID uuid.UUID) {
	if userID == uuid.Nil {
		return
	}
	gameTopic := "game-room:" + roomID.String()

	s.mu.Lock()
	st, ok := s.rooms[roomID]
	if !ok {
		s.mu.Unlock()
		s.hub.LeaveTopic(gameTopic, userID)
		s.hub.LeaveTopic("spectator-chat:"+roomID.String(), userID)
		return
	}

	isPlayer := false
	var warningAt time.Time
	var timerStarted bool
	if st.players[userID] > 0 {
		isPlayer = true
		st.players[userID]--
		if st.players[userID] <= 0 {
			delete(st.players, userID)
			warningAt = time.Now()
			st.disconnectedAt[userID] = warningAt
			timerStarted = true
			st.timers[userID] = time.AfterFunc(disconnectGracePeriod, func() {
				s.graceExpired(userID, roomID)
			})
		}
	} else if st.spectators[userID] > 0 {
		st.spectators[userID]--
		if st.spectators[userID] <= 0 {
			delete(st.spectators, userID)
		}
	}
	s.mu.Unlock()

	if timerStarted {
		row, err := s.repo.GetRoom(context.Background(), roomID)
		if err == nil && row != nil && row.Status == string(dto.GameStatusActive) {
			s.hub.SendToUser(userID, ws.Message{
				Type: "game_forfeit_warning",
				Data: map[string]any{
					"room_id":         roomID.String(),
					"game_type":       row.GameType,
					"disconnected_at": warningAt.UTC().Format(time.RFC3339),
					"grace_seconds":   int(disconnectGracePeriod.Seconds()),
				},
			})
		} else {
			s.mu.Lock()
			if st, ok := s.rooms[roomID]; ok {
				if t := st.timers[userID]; t != nil {
					t.Stop()
					delete(st.timers, userID)
				}
				delete(st.disconnectedAt, userID)
			}
			s.mu.Unlock()
		}
	}

	s.hub.LeaveTopic(gameTopic, userID)
	if !isPlayer {
		s.hub.LeaveTopic("spectator-chat:"+roomID.String(), userID)
	}
	s.broadcastPresence(roomID, userID, false, isPlayer)
}

func (s *service) CancelIdleGames(ctx context.Context) (int, error) {
	idleSince := time.Now().Add(-idleGameTimeout)
	rows, err := s.repo.ListIdleActive(ctx, idleSince)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		if err := s.repo.FinishRoom(ctx, row.ID, string(dto.GameStatusAbandoned), nil, "timeout", row.StateJSON); err != nil {
			logger.Log.Warn().Err(err).Str("room_id", row.ID.String()).Msg("cancel idle game")
			continue
		}
		room, err := s.loadRoom(ctx, row.ID)
		if err == nil {
			s.broadcast(room, "game_room_finished", map[string]any{"reason": "timeout"})
		}
		count++
	}
	if count > 0 {
		s.broadcastLiveGamesCount(ctx)
	}
	return count, nil
}

func (s *service) clearForfeit(roomID, userID uuid.UUID) {
	s.mu.Lock()
	st, ok := s.rooms[roomID]
	if !ok {
		s.mu.Unlock()
		return
	}
	timer, hadTimer := st.timers[userID]
	if hadTimer {
		timer.Stop()
		delete(st.timers, userID)
	}
	_, wasOffline := st.disconnectedAt[userID]
	delete(st.disconnectedAt, userID)
	connected := st.players[userID] > 0
	s.mu.Unlock()

	if hadTimer {
		s.hub.SendToUser(userID, ws.Message{
			Type: "game_forfeit_cleared",
			Data: map[string]any{"room_id": roomID.String()},
		})
	}
	if wasOffline {
		s.broadcastPresence(roomID, userID, connected, true)
	}
}

func (s *service) graceExpired(userID, roomID uuid.UUID) {
	s.mu.Lock()
	if st, ok := s.rooms[roomID]; ok {
		delete(st.timers, userID)
		delete(st.disconnectedAt, userID)
	}
	s.mu.Unlock()

	ctx := context.Background()
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil || row == nil || row.Status != string(dto.GameStatusActive) {
		return
	}
	handler, ok := s.handlers[dto.GameType(row.GameType)]
	if !ok {
		return
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return
	}
	res := handler.OnGraceExpired(row.StateJSON, slot)
	if !res.Finished {
		return
	}
	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return
	}
	winner := winnerUserID(res.WinnerSlot, players)
	if err := s.repo.FinishRoom(ctx, roomID, string(dto.GameStatusAbandoned), winner, res.Result, row.StateJSON); err != nil {
		logger.Log.Warn().Err(err).Msg("finish room after grace expired")
		return
	}
	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return
	}
	s.broadcast(room, "game_room_finished", map[string]any{"abandoned_by": userID.String()})
	s.notifyFinished(ctx, room, uuid.Nil)
	s.broadcastLiveGamesCount(ctx)
}

func (s *service) watcherCount(roomID uuid.UUID) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.rooms[roomID]
	if !ok {
		return 0
	}
	return len(st.spectators)
}

func (s *service) broadcastPresence(roomID, userID uuid.UUID, connected, asPlayer bool) {
	data := map[string]any{
		"room_id":       roomID.String(),
		"user_id":       userID.String(),
		"connected":     connected,
		"as_player":     asPlayer,
		"watcher_count": s.watcherCount(roomID),
	}
	if !connected && asPlayer {
		s.mu.Lock()
		if st, ok := s.rooms[roomID]; ok {
			if t, offline := st.disconnectedAt[userID]; offline {
				data["disconnected_at"] = t.UTC().Format(time.RFC3339)
				data["grace_seconds"] = int(disconnectGracePeriod.Seconds())
			}
		}
		s.mu.Unlock()
	}
	s.hub.BroadcastToTopic("game-room:"+roomID.String(), ws.Message{
		Type: "game_room_presence",
		Data: data,
	})
}

func (s *service) loadRoom(ctx context.Context, roomID uuid.UUID) (*dto.GameRoom, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrNotFound
	}
	return s.hydrateRoom(ctx, row)
}

func (s *service) hydrateRoom(ctx context.Context, row *repository.GameRoomRow) (*dto.GameRoom, error) {
	players, err := s.loadPlayers(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	watchers := 0
	if st, ok := s.rooms[row.ID]; ok {
		for i := range players {
			if st.players[players[i].UserID] > 0 {
				players[i].Connected = true
			} else if t, offline := st.disconnectedAt[players[i].UserID]; offline {
				ts := t.UTC().Format(time.RFC3339)
				players[i].DisconnectedAt = &ts
			}
		}
		watchers = len(st.spectators)
	}
	s.mu.Unlock()

	var finishedAt *string
	if row.FinishedAt != nil {
		finishedAt = row.FinishedAt
	}

	var stats json.RawMessage
	if row.Status == string(dto.GameStatusActive) || row.Status == string(dto.GameStatusFinished) || row.Status == string(dto.GameStatusAbandoned) {
		if handler, ok := s.handlers[dto.GameType(row.GameType)]; ok {
			finished := ""
			if finishedAt != nil {
				finished = *finishedAt
			}
			if computed, err := handler.ComputeStats(row.StateJSON, row.Result, row.CreatedAt, finished); err == nil && computed != nil {
				if raw, err := json.Marshal(computed); err == nil {
					stats = raw
				}
			}
		}
	}

	return &dto.GameRoom{
		ID:           row.ID,
		GameType:     dto.GameType(row.GameType),
		Status:       dto.GameStatus(row.Status),
		State:        json.RawMessage(row.StateJSON),
		TurnUserID:   row.TurnUserID,
		WinnerID:     row.WinnerID,
		Result:       row.Result,
		CreatedBy:    row.CreatedBy,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		FinishedAt:   finishedAt,
		Players:      players,
		WatcherCount: watchers,
		Stats:        stats,
	}, nil
}

func (s *service) loadPlayers(ctx context.Context, roomID uuid.UUID) ([]dto.GameRoomPlayer, error) {
	rows, err := s.repo.GetPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	out := make([]dto.GameRoomPlayer, 0, len(rows))
	for _, row := range rows {
		u, err := s.userRepo.GetByID(ctx, row.UserID)
		if err != nil {
			return nil, err
		}
		player := dto.GameRoomPlayer{
			UserID: row.UserID,
			Slot:   row.Slot,
			Joined: row.Joined,
		}
		if u != nil {
			resp := userToResponse(u)
			player.User = *resp
			player.Username = resp.Username
			player.DisplayName = resp.DisplayName
			player.AvatarURL = resp.AvatarURL
			player.Role = string(resp.Role)
		}
		out = append(out, player)
	}
	return out, nil
}

func (s *service) broadcast(room *dto.GameRoom, eventType string, extra map[string]any) {
	payload := map[string]any{
		"room_id": room.ID.String(),
		"room":    room,
	}
	for k, v := range extra {
		payload[k] = v
	}
	s.hub.BroadcastToTopic("game-room:"+room.ID.String(), ws.Message{
		Type: eventType,
		Data: payload,
	})
	for _, p := range room.Players {
		s.hub.SendToUser(p.UserID, ws.Message{
			Type: eventType,
			Data: payload,
		})
	}
}

func (s *service) isViewing(roomID, userID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.rooms[roomID]
	if !ok {
		return false
	}
	return st.players[userID] > 0
}

func (s *service) notifyTurn(ctx context.Context, room *dto.GameRoom) {
	if room.TurnUserID == nil {
		return
	}
	if s.isViewing(room.ID, *room.TurnUserID) {
		s.hub.SendToUser(*room.TurnUserID, ws.Message{
			Type: "game_your_turn",
			Data: map[string]any{
				"room_id":   room.ID.String(),
				"game_type": string(room.GameType),
			},
		})
		return
	}
	if s.notifSvc == nil {
		return
	}
	var actorID uuid.UUID
	for _, p := range room.Players {
		if p.UserID != *room.TurnUserID {
			actorID = p.UserID
			break
		}
	}
	_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
		RecipientID:   *room.TurnUserID,
		Type:          dto.NotifGameYourTurn,
		ReferenceID:   room.ID,
		ReferenceType: string(room.GameType),
		ActorID:       actorID,
		Message:       "Your move in " + string(room.GameType),
	})
}

func (s *service) notifyFinished(ctx context.Context, room *dto.GameRoom, actorID uuid.UUID) {
	if s.notifSvc == nil {
		return
	}
	for _, p := range room.Players {
		if p.UserID == actorID {
			continue
		}
		_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
			RecipientID:   p.UserID,
			Type:          dto.NotifGameFinished,
			ReferenceID:   room.ID,
			ReferenceType: string(room.GameType),
			ActorID:       actorID,
			Message:       "Your " + string(room.GameType) + " game has ended",
		})
	}
}

func winnerUserID(slot *int, players []dto.GameRoomPlayer) *uuid.UUID {
	if slot == nil {
		return nil
	}
	for i := range players {
		if players[i].Slot == *slot {
			uid := players[i].UserID
			return &uid
		}
	}
	return nil
}

func userToResponse(u *model.User) *dto.UserResponse {
	return u.ToResponse()
}

var _ = errors.New
