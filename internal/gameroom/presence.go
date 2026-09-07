package gameroom

import (
	"context"
	"time"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) HandleClientJoin(ctx context.Context, userID, roomID uuid.UUID) {
	if userID == uuid.Nil {
		return
	}
	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil || row == nil {
		return
	}
	isParticipant, err := s.repo.IsParticipant(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: userID})
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
		_ = s.repo.TouchPlayerSeen(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: userID})
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

	if isParticipant && row.Status == string(dto.GameStatusActive) {
		s.resumeTicker(ctx, roomID)
	}

	if isParticipant {
		s.tickerSetConnected(roomID, userID, true)
	}

	s.sendTickerSnapshot(roomID, userID)
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
		s.tickerSetConnected(roomID, userID, false)

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
		cancelled, err := s.repo.CancelIdleRoom(ctx, spec.GameRoomIdleCancel{RoomID: row.ID, IdleSince: idleSince})
		if err != nil {
			logger.Ctx(ctx).Warn().Err(err).Str("room_id", row.ID.String()).Msg("cancel idle game")
			continue
		}
		if !cancelled {
			continue
		}

		s.stopTicker(row.ID, false)

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
	slot, err := s.repo.GetPlayerSlot(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: userID})
	if err != nil {
		return
	}
	res := handler.OnGraceExpired(row.StateJSON, slot)
	if !res.Finished {
		return
	}
	s.mu.Lock()
	if st, ok := s.rooms[roomID]; ok {
		delete(st.disconnectedAt, userID)
	}
	s.mu.Unlock()
	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return
	}
	winner := winnerUserID(res.WinnerSlot, players)

	if err := s.repo.FinishRoom(ctx, spec.GameRoomFinish{
		RoomID:    roomID,
		Status:    string(dto.GameStatusAbandoned),
		WinnerID:  winner,
		Result:    res.Result,
		StateJSON: row.StateJSON,
	}); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Msg("finish room after grace expired")
		return
	}

	s.stopTicker(roomID, false)

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return
	}
	s.broadcast(room, "game_room_finished", map[string]any{"abandoned_by": userID.String()})
	s.notifyFinished(ctx, room, uuid.Nil)
	s.broadcastLiveGamesCount(ctx)
}
