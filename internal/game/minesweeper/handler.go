package minesweeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
)

const (
	defaultWidth     = 16
	defaultHeight    = 16
	defaultMineCount = 40

	phaseCharSelect = "char_select"
	phasePlaying    = "playing"
	phaseFinished   = "finished"

	actionSelectCharacter = "select_character"
	actionReveal          = "reveal"
	actionFlag            = "flag"

	resultMineHit   = "mine_hit"
	resultComplete  = "completed"
	resultForfeit   = "forfeit"
	resultAbandoned = "abandoned"

	characterBernkastel  = "bernkastel"
	characterErika       = "erika"
	characterLambdadelta = "lambdadelta"
	characterDlanor      = "dlanor"
)

var (
	validCharacters = map[string]bool{
		characterBernkastel:  true,
		characterErika:       true,
		characterLambdadelta: true,
		characterDlanor:      true,
	}
)

type (
	Handler struct {
		gameroom.BaseHandler
	}

	action struct {
		Type      string `json:"type"`
		X         int    `json:"x"`
		Y         int    `json:"y"`
		Character string `json:"character"`
	}

	Stats struct {
		DurationSeconds int    `json:"duration_seconds"`
		RevealedP0      int    `json:"revealed_p0"`
		RevealedP1      int    `json:"revealed_p1"`
		FlagsP0         int    `json:"flags_p0"`
		FlagsP1         int    `json:"flags_p1"`
		Reason          string `json:"reason"`
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypeMinesweeper
}

func (h *Handler) Mode() gameroom.Mode {
	return gameroom.ModeConcurrent
}

func (h *Handler) SupportsDraw() bool {
	return false
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	s := newInitialState(defaultWidth, defaultHeight, defaultMineCount)
	raw, err := json.Marshal(s)
	if err != nil {
		return "", -1, err
	}
	return string(raw), -1, nil
}

func loadState(stateJSON string) (*State, error) {
	var s State
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	return &s, nil
}

func okResult(s *State) (gameroom.ActionResult, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	return gameroom.ActionResult{NewStateJSON: string(raw)}, nil
}

func finishResult(s *State, winnerSlot int, reason string) (gameroom.ActionResult, error) {
	s.Phase = phaseFinished
	s.WinnerSlot = &winnerSlot
	s.Reason = reason
	s.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.Marshal(s)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	return gameroom.ActionResult{
		NewStateJSON: string(raw),
		Finished:     true,
		WinnerSlot:   new(winnerSlot),
		Result:       reason,
	}, nil
}

func newRng() *rand.Rand {
	now := time.Now().UnixNano()
	return rand.New(rand.NewPCG(uint64(now), uint64(now>>1)))
}

func (h *Handler) ValidateAction(stateJSON string, actorSlot int, raw json.RawMessage) (gameroom.ActionResult, error) {
	if actorSlot < 0 || actorSlot > 1 {
		return gameroom.ActionResult{}, errors.New("invalid actor slot")
	}
	s, err := loadState(stateJSON)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	var act action
	if err := json.Unmarshal(raw, &act); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("parse action: %w", err)
	}
	switch act.Type {
	case actionSelectCharacter:
		return h.applySelectCharacter(s, actorSlot, act.Character)
	case actionReveal:
		return h.applyReveal(s, actorSlot, act.X, act.Y)
	case actionFlag:
		return h.applyFlag(s, actorSlot, act.X, act.Y)
	default:
		return gameroom.ActionResult{}, fmt.Errorf("unknown action: %s", act.Type)
	}
}

func (h *Handler) applySelectCharacter(s *State, slot int, character string) (gameroom.ActionResult, error) {
	if s.Phase != phaseCharSelect {
		return gameroom.ActionResult{}, errors.New("character selection is closed")
	}
	if !validCharacters[character] {
		return gameroom.ActionResult{}, fmt.Errorf("unknown character: %s", character)
	}
	s.Characters[slot] = character
	if s.Characters[0] != "" && s.Characters[1] != "" {
		s.Phase = phasePlaying
		s.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return okResult(s)
}

func (h *Handler) applyReveal(s *State, slot, x, y int) (gameroom.ActionResult, error) {
	if s.Phase != phasePlaying {
		return gameroom.ActionResult{}, errors.New("game is not in progress")
	}
	if !s.inBounds(x, y) {
		return gameroom.ActionResult{}, errors.New("out of bounds")
	}
	i := s.idx(x, y)
	if s.Flagged[slot][i] {
		return okResult(s)
	}
	if s.Revealed[slot][i] {
		return okResult(s)
	}

	if !s.MinesPlaced {
		if s.PendingClicks[slot] != nil {
			return okResult(s)
		}
		s.PendingClicks[slot] = &[2]int{x, y}
		other := 1 - slot
		if s.PendingClicks[other] == nil {
			return okResult(s)
		}
		zones := [][2]int{
			{s.PendingClicks[0][0], s.PendingClicks[0][1]},
			{s.PendingClicks[1][0], s.PendingClicks[1][1]},
		}
		s.placeMines(newRng(), zones)

		for p := range 2 {
			pc := s.PendingClicks[p]
			revealed := s.floodFill(p, pc[0], pc[1])
			s.RevealedCount[p] += revealed
			if s.RevealedCount[p] >= s.totalSafeCells() {
				return finishResult(s, p, resultComplete)
			}
		}
		return okResult(s)
	}

	if s.Mines[i] {
		s.HitMineX = new(x)
		s.HitMineY = new(y)
		winner := 1 - slot
		return finishResult(s, winner, resultMineHit)
	}

	revealed := s.floodFill(slot, x, y)
	s.RevealedCount[slot] += revealed
	if s.RevealedCount[slot] >= s.totalSafeCells() {
		return finishResult(s, slot, resultComplete)
	}
	return okResult(s)
}

func (h *Handler) applyFlag(s *State, slot, x, y int) (gameroom.ActionResult, error) {
	if s.Phase != phasePlaying {
		return gameroom.ActionResult{}, errors.New("game is not in progress")
	}
	if !s.inBounds(x, y) {
		return gameroom.ActionResult{}, errors.New("out of bounds")
	}
	i := s.idx(x, y)
	if s.Revealed[slot][i] {
		return gameroom.ActionResult{}, errors.New("cannot flag a revealed cell")
	}
	s.Flagged[slot][i] = !s.Flagged[slot][i]
	return okResult(s)
}

func (h *Handler) ProjectState(stateJSON string, finished bool) (string, error) {
	s, err := loadState(stateJSON)
	if err != nil {
		return stateJSON, err
	}
	if !finished {
		s.Mines = nil
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return stateJSON, err
	}
	return string(raw), nil
}

func (h *Handler) OnGraceExpired(stateJSON string, slot int) gameroom.DisconnectResult {
	s, err := loadState(stateJSON)
	if err != nil {
		return gameroom.DisconnectResult{}
	}
	switch s.Phase {
	case phaseCharSelect:
		return gameroom.DisconnectResult{
			Finished: true,
			Result:   resultAbandoned,
		}
	case phasePlaying:
		return gameroom.DisconnectResult{
			Finished:   true,
			WinnerSlot: new(1 - slot),
			Result:     resultForfeit,
		}
	default:
		return gameroom.DisconnectResult{}
	}
}

func (h *Handler) ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error) {
	s, err := loadState(stateJSON)
	if err != nil {
		return nil, err
	}
	var duration int
	if finishedAt != "" {
		start, errStart := time.Parse(time.RFC3339, createdAt)
		end, errEnd := time.Parse(time.RFC3339, finishedAt)
		if errStart == nil && errEnd == nil {
			duration = int(end.Sub(start).Seconds())
		}
	}
	var flags [2]int
	for p := range 2 {
		for i := range s.Flagged[p] {
			if s.Flagged[p][i] {
				flags[p]++
			}
		}
	}
	reason := s.Reason
	if reason == "" {
		reason = result
	}
	return Stats{
		DurationSeconds: duration,
		RevealedP0:      s.RevealedCount[0],
		RevealedP1:      s.RevealedCount[1],
		FlagsP0:         flags[0],
		FlagsP1:         flags[1],
		Reason:          reason,
	}, nil
}
