package gameroom

import (
	"context"
	"strings"
	"time"
	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/model/spec"
	"umineko_city_of_books/internal/text"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

func (s *service) Scoreboard(ctx context.Context, gameType dto.GameType) (*dto.GameScoreboardResponse, error) {
	if _, ok := s.handlers[gameType]; !ok {
		return nil, ErrUnknownGameType
	}
	rows, err := s.repo.Scoreboard(ctx, string(gameType))
	if err != nil {
		return nil, err
	}

	ids := make([]uuid.UUID, len(rows))
	for i := range rows {
		ids[i] = rows[i].UserID
	}
	byID, err := s.usersByID(ctx, ids)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("game_type", string(gameType)).Msg("load scoreboard users")
	}

	out := make([]dto.GameScoreboardRow, 0, len(rows))
	for _, r := range rows {
		u := byID[r.UserID]
		if u == nil {
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

func (s *service) GetTopWinnerIDs(ctx context.Context, gameType dto.GameType) ([]string, error) {
	return s.repo.GetTopWinnerIDs(ctx, string(gameType))
}

func (s *service) postChat(ctx context.Context, roomID, userID uuid.UUID, body string, player bool) (*dto.SpectatorMessage, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, ErrEmptyChat
	}
	body = text.ClampRunes(body, maxChatBodyLen)

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

	isParticipant, err := s.repo.IsParticipant(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: userID})
	if err != nil {
		return nil, err
	}
	if player && !isParticipant {
		return nil, ErrNotParticipant
	}
	if !player && isParticipant {
		return nil, ErrPlayersCantChat
	}

	if err := s.contentFilter.Check(ctx, body); err != nil {
		return nil, err
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
	if player {
		st.playerChat = append(st.playerChat, msg)
		if len(st.playerChat) > maxChatMessages {
			st.playerChat = st.playerChat[len(st.playerChat)-maxChatMessages:]
		}
	} else {
		st.chat = append(st.chat, msg)
		if len(st.chat) > maxChatMessages {
			st.chat = st.chat[len(st.chat)-maxChatMessages:]
		}
	}
	s.mu.Unlock()

	data := map[string]any{
		"room_id": roomID.String(),
		"message": msg,
	}
	if player {
		players, err := s.loadPlayers(ctx, roomID)
		if err == nil {
			for _, p := range players {
				s.hub.SendToUser(p.UserID, ws.Message{Type: "player_chat_message", Data: data})
			}
		}
	} else {
		s.hub.BroadcastToTopic("spectator-chat:"+roomID.String(), ws.Message{Type: "spectator_chat_message", Data: data})
	}

	return &msg, nil
}

func (s *service) PostSpectatorChat(ctx context.Context, roomID, userID uuid.UUID, body string) (*dto.SpectatorMessage, error) {
	return s.postChat(ctx, roomID, userID, body, false)
}

func (s *service) GetSpectatorChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error) {
	if viewerID != uuid.Nil {
		isParticipant, err := s.repo.IsParticipant(ctx, spec.GameRoomPlayerRef{RoomID: roomID, UserID: viewerID})
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
	return s.postChat(ctx, roomID, userID, body, true)
}

func (s *service) GetPlayerChat(ctx context.Context, roomID, viewerID uuid.UUID) (*dto.SpectatorChatResponse, error) {
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
