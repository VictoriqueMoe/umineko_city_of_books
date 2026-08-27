package gameroom

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/ws"

	"github.com/google/uuid"
)

const (
	tickerPersistInterval = 60 * time.Second
	tickerWriteTimeout    = 10 * time.Second
	tickerFlushTimeout    = 5 * time.Second
)

type (
	tickHandle struct {
		sim       GameSim
		msgType   string
		topicID   uuid.UUID
		slots     map[uuid.UUID]int
		players   [2]uuid.UUID
		cancel    context.CancelFunc
		stopOnce  sync.Once
		flush     atomic.Bool
		lastFrame atomic.Value
	}
)

func (s *service) tickerFor(roomID uuid.UUID) *tickHandle {
	s.tickMu.RLock()
	defer s.tickMu.RUnlock()

	return s.ticks[roomID]
}

func (s *service) startTicker(roomID uuid.UUID, th TickingHandler, stateJSON string, players []dto.GameRoomPlayer, connected [2]bool) {
	if s.tickerFor(roomID) != nil {
		return
	}

	sim, err := th.NewSim(roomID, stateJSON)
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("build game sim")
		return
	}

	for slot, live := range connected {
		sim.SetConnected(slot, live)
	}

	h := &tickHandle{
		sim:     sim,
		msgType: th.SnapshotType(),
		topicID: ws.TopicUUID("game-room:" + roomID.String()),
		slots:   make(map[uuid.UUID]int, len(players)),
	}
	for _, p := range players {
		if p.Slot < 0 || p.Slot > 1 {
			continue
		}
		h.slots[p.UserID] = p.Slot
		h.players[p.Slot] = p.UserID
	}

	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel

	s.tickMu.Lock()
	if _, running := s.ticks[roomID]; running {
		s.tickMu.Unlock()
		cancel()
		return
	}
	s.ticks[roomID] = h
	s.tickMu.Unlock()

	s.tickWG.Add(1)
	go s.runTicker(ctx, roomID, h, th)
}

func (s *service) resumeTicker(ctx context.Context, roomID uuid.UUID) {
	if s.tickerFor(roomID) != nil {
		return
	}

	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil || row == nil || row.Status != string(dto.GameStatusActive) {
		return
	}

	handler, known := s.handlers[dto.GameType(row.GameType)]
	if !known {
		return
	}

	th, tickable := handler.(TickingHandler)
	if !tickable {
		return
	}

	players, err := s.loadPlayers(ctx, roomID)
	if err != nil {
		return
	}

	var connected [2]bool

	s.mu.Lock()
	st := s.stateFor(roomID)
	for _, p := range players {
		if p.Slot < 0 || p.Slot > 1 {
			continue
		}
		connected[p.Slot] = st.players[p.UserID] > 0
	}
	s.mu.Unlock()

	s.startTicker(roomID, th, row.StateJSON, players, connected)
}

func (s *service) stopTicker(roomID uuid.UUID, flush bool) {
	s.tickMu.Lock()
	h := s.ticks[roomID]
	delete(s.ticks, roomID)
	s.tickMu.Unlock()

	if h == nil {
		return
	}

	h.flush.Store(flush)
	h.stopOnce.Do(h.cancel)
}

func (s *service) runTicker(ctx context.Context, roomID uuid.UUID, h *tickHandle, th TickingHandler) {
	defer s.tickWG.Done()

	ticker := time.NewTicker(th.TickInterval())
	defer ticker.Stop()

	step := th.TickInterval()
	every := th.SnapshotEvery()
	persistEvery := int(tickerPersistInterval / step)
	n, sinceReconcile := 0, 0

	for {
		select {
		case <-ctx.Done():
			s.flushTicker(roomID, h)
			return
		case <-ticker.C:
			res := h.sim.Tick(step)
			n++
			sinceReconcile++

			if every > 0 && n%every == 0 {
				snap := h.sim.Snapshot()
				h.lastFrame.Store(snap)
				s.hub.BroadcastFrameToRoom(h.topicID, ws.Message{Type: h.msgType, Data: snap})
			}

			if res.Finished {
				s.finishFromTicker(roomID, h, res)
				return
			}

			if res.Scored {
				s.persistPoint(roomID, h, res)
			}

			if sinceReconcile < persistEvery {
				continue
			}

			sinceReconcile = 0
			if !s.reconcileTicker(roomID, h) {
				return
			}
		}
	}
}

func (s *service) submitMuFor(roomID uuid.UUID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()

	return &s.stateFor(roomID).submitMu
}

func (s *service) persistPoint(roomID uuid.UUID, h *tickHandle, res TickResult) {
	stateJSON, err := h.sim.StateJSON()
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker point state")
		return
	}

	submitMu := s.submitMuFor(roomID)
	submitMu.Lock()
	defer submitMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), tickerWriteTimeout)
	defer cancel()

	if err := s.repo.SetState(ctx, roomID, stateJSON, nil); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker point write")
		return
	}

	room, err := s.loadRoom(ctx, roomID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker point reload")
		return
	}

	bySlot := -1
	if res.WinnerSlot != nil {
		bySlot = *res.WinnerSlot
	}

	s.broadcast(room, "game_room_action", map[string]any{"by_slot": bySlot, "reason": "point"})
}

func (s *service) reconcileTicker(roomID uuid.UUID, h *tickHandle) bool {
	ctx, cancel := context.WithTimeout(context.Background(), tickerWriteTimeout)
	defer cancel()

	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker reconcile read")
		return true
	}

	if row == nil || row.Status != string(dto.GameStatusActive) {
		s.stopTicker(roomID, false)
		return false
	}

	stateJSON, err := h.sim.StateJSON()
	if err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker reconcile state")
		return true
	}

	if err := s.repo.SetState(ctx, roomID, stateJSON, nil); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker reconcile write")
	}

	return true
}

func (s *service) finishFromTicker(roomID uuid.UUID, h *tickHandle, res TickResult) {
	stateJSON, err := h.sim.StateJSON()
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker finish state")
		return
	}

	var winner *uuid.UUID
	if res.WinnerSlot != nil && *res.WinnerSlot >= 0 && *res.WinnerSlot < len(h.players) {
		if id := h.players[*res.WinnerSlot]; id != uuid.Nil {
			winner = new(id)
		}
	}

	submitMu := s.submitMuFor(roomID)
	submitMu.Lock()
	defer submitMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), tickerWriteTimeout)
	defer cancel()

	if _, err := s.finishAndBroadcast(ctx, roomID, winner, res.Result, stateJSON, nil, uuid.Nil); err != nil {
		if errors.Is(err, ErrRoomNotActive) {
			logger.Ctx(ctx).Info().Str("room_id", roomID.String()).Msg("ticker finish raced an earlier finish")
			return
		}
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker finish room")
	}
}

func (s *service) flushTicker(roomID uuid.UUID, h *tickHandle) {
	if !h.flush.Load() {
		return
	}

	stateJSON, err := h.sim.StateJSON()
	if err != nil {
		logger.Log.Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker flush state")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), tickerFlushTimeout)
	defer cancel()

	row, err := s.repo.GetRoom(ctx, roomID)
	if err != nil || row == nil || row.Status != string(dto.GameStatusActive) {
		return
	}

	if err := s.repo.SetState(ctx, roomID, stateJSON, nil); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("room_id", roomID.String()).Msg("ticker flush write")
	}
}

func (s *service) tickerSetConnected(roomID, userID uuid.UUID, connected bool) {
	h := s.tickerFor(roomID)
	if h == nil {
		return
	}

	slot, ok := h.slots[userID]
	if !ok {
		return
	}

	h.sim.SetConnected(slot, connected)
}

func (s *service) sendTickerSnapshot(roomID, userID uuid.UUID) {
	h := s.tickerFor(roomID)
	if h == nil {
		return
	}

	snap := h.lastFrame.Load()
	if snap == nil {
		return
	}

	s.hub.SendToUser(userID, ws.Message{Type: h.msgType, Data: snap})
}

func (s *service) HandleClientInput(userID, roomID uuid.UUID, payload json.RawMessage) {
	h := s.tickerFor(roomID)
	if h == nil {
		return
	}

	slot, ok := h.slots[userID]
	if !ok {
		return
	}

	h.sim.ApplyInput(slot, payload)
}

func (s *service) Shutdown(ctx context.Context) {
	s.tickMu.RLock()
	ids := make([]uuid.UUID, 0, len(s.ticks))
	for id := range s.ticks {
		ids = append(ids, id)
	}
	s.tickMu.RUnlock()

	for _, id := range ids {
		s.stopTicker(id, true)
	}

	done := make(chan struct{})
	go func() {
		s.tickWG.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		logger.Ctx(ctx).Warn().Msg("game ticker drain incomplete")
	}
}
