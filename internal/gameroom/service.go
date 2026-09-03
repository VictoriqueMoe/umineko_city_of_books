package gameroom

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/contentfilter"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/repository"
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
		Page     bounds.Page
	}

	Service interface {
		Invite(ctx context.Context, inviterID, opponentID uuid.UUID, gameType dto.GameType) (*dto.GameRoom, error)
		Accept(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		Decline(ctx context.Context, roomID, userID uuid.UUID) error
		Cancel(ctx context.Context, roomID, userID uuid.UUID) error
		Get(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.GameRoom, error)
		List(ctx context.Context, userID uuid.UUID, filter ListFilter) (*dto.GameRoomListResponse, error)
		ListLive(ctx context.Context, gameType dto.GameType, page bounds.Page) (*dto.GameRoomListResponse, error)
		ListFinished(ctx context.Context, gameType dto.GameType, page bounds.Page) (*dto.GameRoomListResponse, error)
		CountLive(ctx context.Context) (int, error)
		SubmitAction(ctx context.Context, roomID, userID uuid.UUID, action json.RawMessage) (*dto.GameRoom, error)
		Resign(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		OfferDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		AcceptDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		DeclineDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error)
		Scoreboard(ctx context.Context, gameType dto.GameType) (*dto.GameScoreboardResponse, error)
		GetTopWinnerIDs(ctx context.Context, gameType dto.GameType) ([]string, error)
		PostSpectatorChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error)
		GetSpectatorChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error)
		PostPlayerChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error)
		GetPlayerChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error)
		HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID)
		HandleClientLeave(userID, roomID uuid.UUID)
		HandleClientInput(userID, roomID uuid.UUID, payload json.RawMessage)
		HandleClientPing(userID uuid.UUID, rttMS int)
		Shutdown(ctx context.Context)
		CancelIdleGames(ctx context.Context) (int, error)
	}

	drawOffer struct {
		fromSlot  int
		offeredAt time.Time
	}

	roomState struct {
		players        map[uuid.UUID]int
		spectators     map[uuid.UUID]int
		timers         map[uuid.UUID]*time.Timer
		disconnectedAt map[uuid.UUID]time.Time
		chat           []dto.SpectatorMessage
		playerChat     []dto.SpectatorMessage
		draw           *drawOffer
		submitMu       sync.Mutex
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

		tickMu sync.RWMutex
		ticks  map[uuid.UUID]*tickHandle
		tickWG sync.WaitGroup
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
		ticks:         make(map[uuid.UUID]*tickHandle),
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

func (s *service) loadRoomForActor(ctx context.Context, roomID, userID uuid.UUID, requireStatus dto.GameStatus, statusErr error) (*repository.GameRoomRow, int, error) {
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, 0, err
	}
	if row == nil {
		return nil, 0, ErrNotFound
	}
	if row.Status != string(requireStatus) {
		return nil, 0, statusErr
	}
	slot, err := s.repo.GetPlayerSlot(ctx, roomID, userID)
	if err != nil {
		return nil, 0, ErrNotParticipant
	}
	return row, slot, nil
}

func (s *service) loadActiveRoomForActor(ctx context.Context, roomID, userID uuid.UUID) (*repository.GameRoomRow, int, error) {
	return s.loadRoomForActor(ctx, roomID, userID, dto.GameStatusActive, ErrRoomNotActive)
}

func (s *service) loadPendingRoomForActor(ctx context.Context, roomID, userID uuid.UUID) (*repository.GameRoomRow, int, error) {
	return s.loadRoomForActor(ctx, roomID, userID, dto.GameStatusPending, ErrRoomNotPending)
}

func (s *service) hydrateRoomList(ctx context.Context, rows []repository.GameRoomRow, total int) (*dto.GameRoomListResponse, error) {
	out := make([]dto.GameRoom, 0, len(rows))
	for i := range rows {
		r, err := s.hydrateRoom(ctx, &rows[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return &dto.GameRoomListResponse{Rooms: out, Total: total}, nil
}

func (s *service) finishAndBroadcast(ctx context.Context, roomID uuid.UUID, winner *uuid.UUID, result, stateJSON string, extras map[string]any, actorID uuid.UUID) (*dto.GameRoom, error) {
	if err := s.repo.FinishRoom(ctx, roomID, string(dto.GameStatusFinished), winner, result, stateJSON); err != nil {
		if errors.Is(err, repository.ErrRoomNotActive) {
			return nil, ErrRoomNotActive
		}

		return nil, err
	}

	s.stopTicker(roomID, false)

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	s.broadcast(room, "game_room_finished", extras)
	s.notifyFinished(ctx, room, actorID)
	s.broadcastLiveGamesCount(ctx)
	go s.broadcastTopWinner(room.GameType)
	return room, nil
}

func (s *service) broadcastTopWinner(gameType dto.GameType) {
	var msgType string
	switch gameType {
	case dto.GameTypeChess:
		msgType = "top_chess_changed"
	case dto.GameTypeCheckers:
		msgType = "top_checkers_changed"
	case dto.GameTypeOthello:
		msgType = "top_othello_changed"
	case dto.GameTypeMinesweeper:
		msgType = "top_minesweeper_changed"
	default:
		return
	}

	bgCtx := context.Background()
	ids, err := s.repo.GetTopWinnerIDs(bgCtx, string(gameType))
	if err != nil {
		return
	}

	s.hub.Broadcast(ws.Message{
		Type: msgType,
		Data: map[string]any{
			"user_ids": ids,
		},
	})
}
