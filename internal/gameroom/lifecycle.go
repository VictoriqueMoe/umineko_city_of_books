package gameroom

import (
	"context"
	"fmt"
	"umineko_city_of_books/internal/bounds"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

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
	blocked, err := s.blockSvc.IsBlockedEither(ctx, spec.BlockPairSpec{UserA: inviterID, UserB: opponentID})
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

	created, err := s.repo.CreateInvite(ctx, spec.NewGameRoomInvite{
		GameType:         string(gameType),
		InitialStateJSON: "{}",
		InviterID:        inviterID,
		OpponentID:       opponentID,
	})
	if err != nil {
		return nil, err
	}

	room, err := s.hydrateRoom(ctx, created)
	if err != nil {
		return nil, err
	}

	if s.notifSvc != nil {
		_ = s.notifSvc.Notify(ctx, dto.NotifyParams{
			RecipientID:   opponentID,
			Type:          dto.NotifGameInvite,
			ReferenceID:   created.ID,
			ReferenceType: string(gameType),
			ActorID:       inviterID,
			Message:       inviter.DisplayName + " invited you to a " + string(gameType) + " game",
		})
	}

	return room, nil
}

func (s *service) Accept(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, slot, err := s.loadPendingRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if slot != 1 {
		return nil, ErrNotInvitee
	}

	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	var inviterID uuid.UUID
	for i := range players {
		if players[i].Slot == 0 {
			inviterID = players[i].UserID
			break
		}
	}

	s.mu.Lock()
	st := s.stateFor(roomID)
	accepterPresent := st.players[userID] > 0
	inviterPresent := inviterID != uuid.Nil && st.players[inviterID] > 0
	s.mu.Unlock()
	if !accepterPresent {
		return nil, ErrAccepterNotInRoom
	}
	if !inviterPresent {
		return nil, ErrInviterNotInRoom
	}

	handler, ok := s.handlers[dto.GameType(row.GameType)]
	if !ok {
		return nil, ErrUnknownGameType
	}

	stateJSON, firstTurnSlot, err := handler.InitialState(roomID, players)
	if err != nil {
		return nil, err
	}

	firstTurnUser := winnerUserID(&firstTurnSlot, players)

	if err := s.repo.Start(ctx, spec.GameRoomStart{
		RoomID:     roomID,
		UserID:     userID,
		StateJSON:  stateJSON,
		TurnUserID: firstTurnUser,
		Status:     string(dto.GameStatusActive),
	}); err != nil {
		return nil, err
	}

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	s.broadcast(room, "game_room_started", nil)
	s.notifyTurn(ctx, room)
	s.broadcastLiveGamesCount(ctx)

	if th, tickable := handler.(TickingHandler); tickable {
		s.startTicker(roomID, th, stateJSON, players, [2]bool{true, true})
	}

	return room, nil
}

func (s *service) Cancel(ctx context.Context, roomID, userID uuid.UUID) error {
	_, slot, err := s.loadPendingRoomForActor(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if slot != 0 {
		return ErrNotInviter
	}
	if err := s.repo.SetStatus(ctx, spec.GameRoomStatusUpdate{RoomID: roomID, Status: string(dto.GameStatusDeclined)}); err != nil {
		return err
	}
	room, _ := s.loadRoom(ctx, roomID)
	if room != nil {
		s.broadcast(room, "game_room_cancelled", map[string]any{"by_user_id": userID.String()})
	}
	return nil
}

func (s *service) Decline(ctx context.Context, roomID, userID uuid.UUID) error {
	_, slot, err := s.loadPendingRoomForActor(ctx, roomID, userID)
	if err != nil {
		return err
	}
	if slot != 1 {
		return ErrNotInvitee
	}
	if err := s.repo.SetStatus(ctx, spec.GameRoomStatusUpdate{RoomID: roomID, Status: string(dto.GameStatusDeclined)}); err != nil {
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
		isParticipant, err := s.repo.IsParticipant(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: viewerID})
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
	rows, total, err := s.repo.ListForUser(ctx, spec.GameRoomUserFilter{
		UserID:   userID,
		GameType: string(filter.GameType),
		Statuses: filter.Statuses,
		Limit:    filter.Page.Limit(),
		Offset:   filter.Page.Offset(),
	})
	if err != nil {
		return nil, err
	}
	return s.hydrateRoomList(ctx, rows, total)
}

func (s *service) ListLive(ctx context.Context, gameType dto.GameType, page bounds.Page) (*dto.GameRoomListResponse, error) {
	rows, total, err := s.repo.ListLive(ctx, spec.GameRoomListFilter{
		GameType: string(gameType),
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}
	return s.hydrateRoomList(ctx, rows, total)
}

func (s *service) CountLive(ctx context.Context) (int, error) {
	return s.repo.CountLive(ctx)
}

func (s *service) broadcastLiveGamesCount(ctx context.Context) {
	count, err := s.repo.CountLive(ctx)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("count live games for broadcast")
		return
	}
	s.hub.Broadcast(ws.Message{
		Type: "live_games_count",
		Data: map[string]any{"count": count},
	})
}

func (s *service) ListFinished(ctx context.Context, gameType dto.GameType, page bounds.Page) (*dto.GameRoomListResponse, error) {
	rows, total, err := s.repo.ListFinished(ctx, spec.GameRoomListFilter{
		GameType: string(gameType),
		Limit:    page.Limit(),
		Offset:   page.Offset(),
	})
	if err != nil {
		return nil, err
	}
	return s.hydrateRoomList(ctx, rows, total)
}
