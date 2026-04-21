package chess

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
	chesslib "github.com/notnil/chess"
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
	if err := game.Move(move); err != nil {
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

func (h *Handler) OnGraceExpired(_ string, _ int) gameroom.DisconnectResult {
	return gameroom.DisconnectResult{}
}

func loadGame(pgn string) (*chesslib.Game, error) {
	if pgn == "" {
		return chesslib.NewGame(), nil
	}
	fn, err := chesslib.PGN(strings.NewReader(pgn + "\n"))
	if err != nil {
		return nil, fmt.Errorf("parse pgn: %w", err)
	}
	return chesslib.NewGame(fn), nil
}
