package checkers

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
	slotRed   = 0
	slotBlack = 1

	boardSize = 8

	cellEmpty     byte = '.'
	cellRedMan    byte = 'r'
	cellRedKing   byte = 'R'
	cellBlackMan  byte = 'b'
	cellBlackKing byte = 'B'

	drawMoveLimit = 40
)

type (
	Handler struct{}

	state struct {
		Board             string `json:"board"`
		Turn              int    `json:"turn"`
		TotalMoves        int    `json:"total_moves"`
		RedCaptures       int    `json:"red_captures"`
		BlackCaptures     int    `json:"black_captures"`
		RedCrownings      int    `json:"red_crownings"`
		BlackCrownings    int    `json:"black_crownings"`
		MovesSinceCapture int    `json:"moves_since_capture"`
	}

	moveAction struct {
		From string   `json:"from"`
		Path []string `json:"path"`
	}

	Stats struct {
		TotalMoves      int    `json:"total_moves"`
		RedMoves        int    `json:"red_moves"`
		BlackMoves      int    `json:"black_moves"`
		RedCaptures     int    `json:"red_captures"`
		BlackCaptures   int    `json:"black_captures"`
		RedCrownings    int    `json:"red_crownings"`
		BlackCrownings  int    `json:"black_crownings"`
		ResultReason    string `json:"result_reason"`
		DurationSeconds int    `json:"duration_seconds"`
		FinalBoard      string `json:"final_board"`
		RedPiecesLeft   int    `json:"red_pieces_left"`
		BlackPiecesLeft int    `json:"black_pieces_left"`
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypeCheckers
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	b := make([]byte, boardSize*boardSize)
	for i := range b {
		b[i] = cellEmpty
	}
	for row := 0; row < 3; row++ {
		for col := 0; col < boardSize; col++ {
			if isDarkSquare(row, col) {
				b[row*boardSize+col] = cellRedMan
			}
		}
	}
	for row := 5; row < boardSize; row++ {
		for col := 0; col < boardSize; col++ {
			if isDarkSquare(row, col) {
				b[row*boardSize+col] = cellBlackMan
			}
		}
	}

	s := state{
		Board: string(b),
		Turn:  slotRed,
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return "", 0, err
	}
	return string(raw), slotRed, nil
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
	if mv.From == "" || len(mv.Path) == 0 {
		return gameroom.ActionResult{}, errors.New("missing from/path")
	}

	fromR, fromC, err := parseSquare(mv.From)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	piece := b[fromR][fromC]
	if piece == cellEmpty {
		return gameroom.ActionResult{}, errors.New("no piece at source")
	}
	if pieceOwner(piece) != actorSlot {
		return gameroom.ActionResult{}, errors.New("that piece is not yours")
	}

	path := make([][2]int, 0, len(mv.Path))
	for _, sq := range mv.Path {
		r, c, perr := parseSquare(sq)
		if perr != nil {
			return gameroom.ActionResult{}, perr
		}
		path = append(path, [2]int{r, c})
	}

	capturesAvailable := playerHasCapture(b, actorSlot)
	firstStep := path[0]
	isJump := absInt(firstStep[0]-fromR) == 2

	if capturesAvailable && !isJump {
		return gameroom.ActionResult{}, errors.New("capture is mandatory")
	}

	var captured int
	var crowned bool
	if isJump {
		captured, crowned, err = applyJumpSequence(&b, fromR, fromC, path, actorSlot)
		if err != nil {
			return gameroom.ActionResult{}, err
		}
	} else {
		if len(path) != 1 {
			return gameroom.ActionResult{}, errors.New("simple moves must be a single step")
		}
		crowned, err = applySimpleMove(&b, fromR, fromC, firstStep[0], firstStep[1], actorSlot)
		if err != nil {
			return gameroom.ActionResult{}, err
		}
	}

	s.Board = boardString(b)
	s.TotalMoves++
	if isJump {
		s.MovesSinceCapture = 0
		if actorSlot == slotRed {
			s.RedCaptures += captured
		} else {
			s.BlackCaptures += captured
		}
	} else {
		s.MovesSinceCapture++
	}
	if crowned {
		if actorSlot == slotRed {
			s.RedCrownings++
		} else {
			s.BlackCrownings++
		}
	}

	nextSlot := 1 - actorSlot
	s.Turn = nextSlot

	outcome, reason := evaluateOutcome(b, nextSlot, s.MovesSinceCapture)
	raw, err := json.Marshal(s)
	if err != nil {
		return gameroom.ActionResult{}, err
	}
	res := gameroom.ActionResult{NewStateJSON: string(raw)}
	if outcome.finished {
		res.Finished = true
		res.Result = reason
		if outcome.winnerSlot != nil {
			w := *outcome.winnerSlot
			res.WinnerSlot = &w
		}
		return res, nil
	}
	res.NextTurnSlot = &nextSlot
	return res, nil
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
	red, black := countPieces(s.Board)
	redMoves, blackMoves := splitMoves(s.TotalMoves)
	return Stats{
		TotalMoves:      s.TotalMoves,
		RedMoves:        redMoves,
		BlackMoves:      blackMoves,
		RedCaptures:     s.RedCaptures,
		BlackCaptures:   s.BlackCaptures,
		RedCrownings:    s.RedCrownings,
		BlackCrownings:  s.BlackCrownings,
		ResultReason:    classifyResult(result),
		DurationSeconds: durationSeconds(createdAt, finishedAt),
		FinalBoard:      s.Board,
		RedPiecesLeft:   red,
		BlackPiecesLeft: black,
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
	case "no_pieces":
		return "no_pieces"
	case "forty_move_rule":
		return "forty_move_rule"
	}
	return result
}

func splitMoves(total int) (int, int) {
	red := (total + 1) / 2
	black := total / 2
	return red, black
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
