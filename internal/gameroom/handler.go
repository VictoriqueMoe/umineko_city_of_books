package gameroom

import (
	"encoding/json"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	// ActionResult is returned by a GameHandler after validating an action.
	// If Finished is true the service will mark the room finished and set
	// WinnerSlot/Result. Otherwise NextTurnSlot determines whose turn is next.
	ActionResult struct {
		NewStateJSON string
		NextTurnSlot *int
		Finished     bool
		WinnerSlot   *int
		Result       string
	}

	// DisconnectResult lets a game handler optionally terminate the game
	// when a player's grace-period timer expires. Live games can forfeit;
	// correspondence games return an empty result (no change).
	DisconnectResult struct {
		Finished   bool
		WinnerSlot *int
		Result     string
	}

	GameHandler interface {
		GameType() dto.GameType
		InitialState(roomID uuid.UUID, players []dto.GameRoomPlayer) (stateJSON string, firstTurnSlot int, err error)
		ValidateAction(stateJSON string, actorSlot int, action json.RawMessage) (ActionResult, error)
		OnGraceExpired(stateJSON string, playerSlot int) DisconnectResult
		ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error)
	}
)
