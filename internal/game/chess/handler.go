package chess

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	chesslib "github.com/corentings/chess/v2"
	"github.com/google/uuid"
)

const (
	slotWhite = 0
	slotBlack = 1
)

type (
	Handler struct{}

	state struct {
		FEN string `json:"fen"`
		PGN string `json:"pgn"`
	}

	moveAction struct {
		From      string `json:"from"`
		To        string `json:"to"`
		Promotion string `json:"promotion"`
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypeChess
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	game := chesslib.NewGame()
	s := state{
		FEN: game.Position().String(),
		PGN: game.String(),
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return "", 0, err
	}
	return string(raw), slotWhite, nil
}

func (h *Handler) ValidateAction(stateJSON string, actorSlot int, action json.RawMessage) (gameroom.ActionResult, error) {
	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("load state: %w", err)
	}
	game, err := loadGame(s.PGN)
	if err != nil {
		return gameroom.ActionResult{}, err
	}

	sideToMove := game.Position().Turn()
	if sideToMove == chesslib.White && actorSlot != slotWhite {
		return gameroom.ActionResult{}, errors.New("not your turn: white to move")
	}
	if sideToMove == chesslib.Black && actorSlot != slotBlack {
		return gameroom.ActionResult{}, errors.New("not your turn: black to move")
	}

	var mv moveAction
	if err := json.Unmarshal(action, &mv); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("parse action: %w", err)
	}
	if mv.From == "" || mv.To == "" {
		return gameroom.ActionResult{}, errors.New("missing from/to")
	}

	uci := mv.From + mv.To + mv.Promotion
	move, err := chesslib.UCINotation{}.Decode(game.Position(), uci)
	if err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("illegal move: %w", err)
	}
	if err := game.Move(move, nil); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("illegal move: %w", err)
	}

	newState := state{
		FEN: game.Position().String(),
		PGN: game.String(),
	}
	raw, err := json.Marshal(newState)
	if err != nil {
		return gameroom.ActionResult{}, err
	}

	result := gameroom.ActionResult{NewStateJSON: string(raw)}
	if outcome := game.Outcome(); outcome != chesslib.NoOutcome {
		result.Finished = true
		result.Result = string(outcome)
		switch outcome {
		case chesslib.WhiteWon:
			slot := slotWhite
			result.WinnerSlot = &slot
		case chesslib.BlackWon:
			slot := slotBlack
			result.WinnerSlot = &slot
		}
		return result, nil
	}
	nextSlot := slotWhite
	if game.Position().Turn() == chesslib.Black {
		nextSlot = slotBlack
	}
	result.NextTurnSlot = &nextSlot
	return result, nil
}

type Stats struct {
	TotalPly        int    `json:"total_ply"`
	WhiteMoves      int    `json:"white_moves"`
	BlackMoves      int    `json:"black_moves"`
	WhiteCaptures   int    `json:"white_captures"`
	BlackCaptures   int    `json:"black_captures"`
	WhiteChecks     int    `json:"white_checks"`
	BlackChecks     int    `json:"black_checks"`
	ResultReason    string `json:"result_reason"`
	DurationSeconds int    `json:"duration_seconds"`
	FinalFEN        string `json:"final_fen"`
}

func (h *Handler) ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error) {
	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}
	game, err := loadGame(s.PGN)
	if err != nil {
		return nil, err
	}

	stats := Stats{
		ResultReason: classifyResult(result, game),
		FinalFEN:     s.FEN,
	}
	for i, move := range game.Moves() {
		stats.TotalPly++
		whiteMove := i%2 == 0
		if whiteMove {
			stats.WhiteMoves++
		} else {
			stats.BlackMoves++
		}
		if move.HasTag(chesslib.Capture) || move.HasTag(chesslib.EnPassant) {
			if whiteMove {
				stats.WhiteCaptures++
			} else {
				stats.BlackCaptures++
			}
		}
		if move.HasTag(chesslib.Check) {
			if whiteMove {
				stats.WhiteChecks++
			} else {
				stats.BlackChecks++
			}
		}
	}

	stats.DurationSeconds = durationSeconds(createdAt, finishedAt)
	return stats, nil
}

func classifyResult(result string, game *chesslib.Game) string {
	if result == "abandoned" {
		return "abandoned"
	}
	if result == "timeout" {
		return "timeout"
	}
	if result == "resign" || result == "resigned" {
		return "resignation"
	}
	switch game.Outcome() {
	case chesslib.WhiteWon, chesslib.BlackWon:
		if game.Method() == chesslib.Checkmate {
			return "checkmate"
		}
		return "win"
	case chesslib.Draw:
		switch game.Method() {
		case chesslib.Stalemate:
			return "stalemate"
		case chesslib.InsufficientMaterial:
			return "insufficient_material"
		case chesslib.FiftyMoveRule:
			return "fifty_move_rule"
		case chesslib.ThreefoldRepetition, chesslib.FivefoldRepetition:
			return "repetition"
		case chesslib.DrawOffer:
			return "draw_agreed"
		}
		return "draw"
	}
	return ""
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
		parsed, err := parseDBTime(finishedAt)
		if err != nil {
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

func (h *Handler) OnGraceExpired(_ string, disconnectedSlot int) gameroom.DisconnectResult {
	winnerSlot := slotWhite
	if disconnectedSlot == slotWhite {
		winnerSlot = slotBlack
	}
	return gameroom.DisconnectResult{
		Finished:   true,
		WinnerSlot: &winnerSlot,
		Result:     "abandoned",
	}
}

func loadGame(pgn string) (*chesslib.Game, error) {
	trimmed := strings.TrimSpace(pgn)
	if trimmed == "" || trimmed == "*" {
		return chesslib.NewGame(), nil
	}
	fn, err := chesslib.PGN(strings.NewReader(pgn + "\n"))
	if err != nil {
		return nil, fmt.Errorf("parse pgn: %w", err)
	}
	return chesslib.NewGame(fn), nil
}
