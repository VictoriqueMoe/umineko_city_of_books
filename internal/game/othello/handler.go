package othello

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
)

const (
	slotBlack = 0
	slotWhite = 1

	boardSize = 8

	cellEmpty byte = '.'
	cellBlack byte = 'B'
	cellWhite byte = 'W'
)

type (
	Handler struct{}

	state struct {
		Board       string    `json:"board"`
		Turn        int       `json:"turn"`
		BlackMoves  int       `json:"black_moves"`
		WhiteMoves  int       `json:"white_moves"`
		BlackPasses int       `json:"black_passes"`
		WhitePasses int       `json:"white_passes"`
		BlackFlips  int       `json:"black_flips"`
		WhiteFlips  int       `json:"white_flips"`
		LastMove    *lastMove `json:"last_move,omitempty"`
	}

	lastMove struct {
		Square  string   `json:"square"`
		Slot    int      `json:"slot"`
		Flipped []string `json:"flipped"`
	}

	moveAction struct {
		Square string `json:"square"`
	}

	Stats struct {
		TotalMoves      int    `json:"total_moves"`
		BlackMoves      int    `json:"black_moves"`
		WhiteMoves      int    `json:"white_moves"`
		BlackPasses     int    `json:"black_passes"`
		WhitePasses     int    `json:"white_passes"`
		BlackDiscs      int    `json:"black_discs"`
		WhiteDiscs      int    `json:"white_discs"`
		BlackFlips      int    `json:"black_flips"`
		WhiteFlips      int    `json:"white_flips"`
		BlackCorners    int    `json:"black_corners"`
		WhiteCorners    int    `json:"white_corners"`
		ResultReason    string `json:"result_reason"`
		DurationSeconds int    `json:"duration_seconds"`
		FinalBoard      string `json:"final_board"`
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypeOthello
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	b := make([]byte, boardSize*boardSize)
	for i := range b {
		b[i] = cellEmpty
	}
	b[3*boardSize+3] = cellWhite
	b[3*boardSize+4] = cellBlack
	b[4*boardSize+3] = cellBlack
	b[4*boardSize+4] = cellWhite

	s := state{
		Board: string(b),
		Turn:  slotBlack,
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return "", 0, err
	}
	return string(raw), slotBlack, nil
}

func (h *Handler) ValidateAction(stateJSON string, actorSlot int, action json.RawMessage) (gameroom.ActionResult, error) {
	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("load state: %w", err)
	}
	if s.Turn != actorSlot {
		return gameroom.ActionResult{}, errors.New("not your turn")
	}

	b, err := parseBoard(s.Board)
	if err != nil {
		return gameroom.ActionResult{}, err
	}

	var mv moveAction
	if err := json.Unmarshal(action, &mv); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("parse action: %w", err)
	}
	if mv.Square == "" {
		return gameroom.ActionResult{}, errors.New("missing square")
	}

	row, col, err := parseSquare(mv.Square)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	if b[row][col] != cellEmpty {
		return gameroom.ActionResult{}, errors.New("square is occupied")
	}

	flipped, err := applyPlacement(&b, row, col, actorSlot)
	if err != nil {
		return gameroom.ActionResult{}, err
	}

	s.Board = boardString(b)
	flippedSquares := make([]string, len(flipped))
	for i, f := range flipped {
		flippedSquares[i] = formatSquare(f.row, f.col)
	}
	s.LastMove = &lastMove{
		Square:  mv.Square,
		Slot:    actorSlot,
		Flipped: flippedSquares,
	}
	if actorSlot == slotBlack {
		s.BlackMoves++
		s.BlackFlips += len(flipped)
	} else {
		s.WhiteMoves++
		s.WhiteFlips += len(flipped)
	}

	outcome, reason := evaluateOutcome(b)
	if outcome.finished {
		s.Turn = -1
		raw, merr := json.Marshal(s)
		if merr != nil {
			return gameroom.ActionResult{}, merr
		}
		res := gameroom.ActionResult{
			NewStateJSON: string(raw),
			Finished:     true,
			Result:       reason,
		}
		if outcome.winnerSlot != nil {
			w := *outcome.winnerSlot
			res.WinnerSlot = &w
		}
		return res, nil
	}

	nextSlot := 1 - actorSlot
	if !playerHasAnyLegalMove(b, nextSlot) {
		if nextSlot == slotBlack {
			s.BlackPasses++
		} else {
			s.WhitePasses++
		}
		nextSlot = actorSlot
	}
	s.Turn = nextSlot

	raw, err := json.Marshal(s)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	return gameroom.ActionResult{
		NewStateJSON: string(raw),
		NextTurnSlot: &nextSlot,
	}, nil
}

func (h *Handler) OnGraceExpired(_ string, disconnectedSlot int) gameroom.DisconnectResult {
	winnerSlot := 1 - disconnectedSlot
	return gameroom.DisconnectResult{
		Finished:   true,
		WinnerSlot: &winnerSlot,
		Result:     "abandoned",
	}
}

func (h *Handler) ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error) {
	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	black, white := countDiscs(s.Board)
	b, perr := parseBoard(s.Board)
	var blackCorners, whiteCorners int
	if perr == nil {
		blackCorners, whiteCorners = cornerCounts(b)
	}
	return Stats{
		TotalMoves:      s.BlackMoves + s.WhiteMoves,
		BlackMoves:      s.BlackMoves,
		WhiteMoves:      s.WhiteMoves,
		BlackPasses:     s.BlackPasses,
		WhitePasses:     s.WhitePasses,
		BlackDiscs:      black,
		WhiteDiscs:      white,
		BlackFlips:      s.BlackFlips,
		WhiteFlips:      s.WhiteFlips,
		BlackCorners:    blackCorners,
		WhiteCorners:    whiteCorners,
		ResultReason:    classifyResult(result),
		DurationSeconds: durationSeconds(createdAt, finishedAt),
		FinalBoard:      s.Board,
	}, nil
}

func classifyResult(result string) string {
	switch result {
	case "abandoned":
		return "abandoned"
	case "timeout":
		return "timeout"
	case "resign", "resigned":
		return "resignation"
	case "no_moves":
		return "no_moves"
	case "most_discs":
		return "most_discs"
	case "draw":
		return "draw"
	}
	return result
}

func durationSeconds(createdAt, finishedAt string) int {
	if createdAt == "" {
		return 0
	}
	start, err := parseDBTime(createdAt)
	if err != nil {
		return 0
	}
	end := time.Now().UTC()
	if finishedAt != "" {
		parsed, perr := parseDBTime(finishedAt)
		if perr != nil {
			return 0
		}
		end = parsed
	}
	d := end.Sub(start)
	if d < 0 {
		return 0
	}
	return int(d.Seconds())
}

func parseDBTime(s string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05Z"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised time format: %s", s)
}
