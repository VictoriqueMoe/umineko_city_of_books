package gameroom

import (
	"context"
	"encoding/json"
	"time"
	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

func (s *service) SubmitAction(ctx context.Context, roomID, userID uuid.UUID, action json.RawMessage) (*dto.GameRoom, error) {
	s.mu.Lock()
	submitMu := &s.stateFor(roomID).submitMu
	s.mu.Unlock()
	submitMu.Lock()
	defer submitMu.Unlock()

	row, slot, err := s.loadActiveRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	handler, ok := s.handlers[dto.GameType(row.GameType)]
	if !ok {
		return nil, ErrUnknownGameType
	}
	if handler.Mode() == ModeTurnBased {
		if row.TurnUserID == nil || *row.TurnUserID != userID {
			return nil, ErrNotYourTurn
		}
	}

	s.clearForfeit(roomID, userID)
	s.clearDrawOffer(roomID)

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
		return s.finishAndBroadcast(ctx, roomID, winner, result.Result, result.NewStateJSON, nil, userID)
	}

	nextTurn := winnerUserID(result.NextTurnSlot, players)
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

func (s *service) OfferDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, slot, err := s.loadActiveRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if handler, ok := s.handlers[dto.GameType(row.GameType)]; ok && !handler.SupportsDraw() {
		return nil, ErrDrawNotSupported
	}

	s.mu.Lock()
	st := s.stateFor(roomID)
	if st.draw != nil {
		s.mu.Unlock()
		return nil, ErrDrawOfferPending
	}
	st.draw = &drawOffer{fromSlot: slot, offeredAt: time.Now()}
	s.mu.Unlock()

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	s.broadcast(room, "game_draw_offered", map[string]any{"by_user_id": userID.String()})
	return room, nil
}

// consumeOpponentDrawOffer validates that a non-self draw offer is pending for userID (at slot)
// and clears it. Returns ErrNoDrawOffer or ErrCannotAcceptOwnDraw if the state is wrong.
func (s *service) consumeOpponentDrawOffer(roomID uuid.UUID, slot int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.stateFor(roomID)
	if st.draw == nil {
		return ErrNoDrawOffer
	}
	if st.draw.fromSlot == slot {
		return ErrCannotAcceptOwnDraw
	}
	st.draw = nil
	return nil
}

func (s *service) AcceptDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, slot, err := s.loadActiveRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.consumeOpponentDrawOffer(roomID, slot); err != nil {
		return nil, err
	}
	return s.finishAndBroadcast(ctx, roomID, nil, "draw_agreed", row.StateJSON, map[string]any{"draw_agreed_by": userID.String()}, userID)
}

func (s *service) DeclineDraw(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	_, slot, err := s.loadActiveRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	if err := s.consumeOpponentDrawOffer(roomID, slot); err != nil {
		return nil, err
	}
	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}
	s.broadcast(room, "game_draw_declined", map[string]any{"by_user_id": userID.String()})
	return room, nil
}

func (s *service) clearDrawOffer(roomID uuid.UUID) {
	s.mu.Lock()
	st, ok := s.rooms[roomID]
	if ok {
		st.draw = nil
	}
	s.mu.Unlock()
}

func (s *service) Resign(ctx context.Context, roomID, userID uuid.UUID) (*dto.GameRoom, error) {
	row, slot, err := s.loadActiveRoomForActor(ctx, roomID, userID)
	if err != nil {
		return nil, err
	}
	s.clearForfeit(roomID, userID)
	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return nil, err
	}
	var winner *uuid.UUID
	for i := range players {
		if players[i].Slot != slot {
			winner = new(players[i].UserID)
			break
		}
	}
	return s.finishAndBroadcast(ctx, roomID, winner, "resign", row.StateJSON, map[string]any{"resigned_by": userID.String()}, userID)
}
