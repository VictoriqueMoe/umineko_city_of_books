package gameroom

import (
	"context"
	"encoding/json"
	"maps"
	"time"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/model"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

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

func (s *service) hydrateRoom(ctx context.Context, row *model.GameRoomRow) (*dto.GameRoom, error) {
	players, err := s.loadPlayers(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	watchers := 0
	var drawOfferFrom *uuid.UUID
	if st, ok := s.rooms[row.ID]; ok {
		for i := range players {
			if st.players[players[i].UserID] > 0 {
				players[i].Connected = true
			} else if t, offline := st.disconnectedAt[players[i].UserID]; offline {
				players[i].DisconnectedAt = new(t.UTC().Format(time.RFC3339))
			}
		}
		watchers = len(st.spectators)
		if st.draw != nil {
			drawOfferFrom = winnerUserID(&st.draw.fromSlot, players)
		}
	}
	s.mu.Unlock()

	finishedAt := row.FinishedAt

	var stats json.RawMessage
	projectedState := row.StateJSON
	finishedStatus := row.Status == string(dto.GameStatusFinished) || row.Status == string(dto.GameStatusAbandoned)
	if row.Status == string(dto.GameStatusActive) || finishedStatus {
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
			if proj, err := handler.ProjectState(row.StateJSON, finishedStatus); err == nil {
				projectedState = proj
			}
		}
	}

	return &dto.GameRoom{
		ID:                row.ID,
		GameType:          dto.GameType(row.GameType),
		Status:            dto.GameStatus(row.Status),
		State:             json.RawMessage(projectedState),
		TurnUserID:        row.TurnUserID,
		WinnerID:          row.WinnerID,
		Result:            row.Result,
		CreatedBy:         row.CreatedBy,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
		FinishedAt:        finishedAt,
		Players:           players,
		WatcherCount:      watchers,
		Stats:             stats,
		DrawOfferFromUser: drawOfferFrom,
	}, nil
}

func (s *service) usersByID(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*model.User, error) {
	users, err := s.userRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]*model.User, len(users))
	for i := range users {
		byID[users[i].ID] = &users[i]
	}

	return byID, nil
}

func (s *service) loadPlayers(ctx context.Context, roomID uuid.UUID) ([]dto.GameRoomPlayer, error) {
	rows, err := s.repo.GetPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].UserID
	}
	byID, err := s.usersByID(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make([]dto.GameRoomPlayer, 0, len(rows))
	for _, row := range rows {
		player := dto.GameRoomPlayer{
			UserID: row.UserID,
			Slot:   row.Slot,
			Joined: row.Joined,
		}
		if u := byID[row.UserID]; u != nil {
			resp := u.ToResponse()
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
	maps.Copy(payload, extra)
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
			return new(players[i].UserID)
		}
	}
	return nil
}
