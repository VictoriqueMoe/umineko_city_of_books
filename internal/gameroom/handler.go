package gameroom

import (
	"encoding/json"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
)

type (
	Mode int

	ActionResult struct {
		NewStateJSON string
		NextTurnSlot *int
		Finished     bool
		WinnerSlot   *int
		Result       string
	}

	DisconnectResult struct {
		Finished   bool
		WinnerSlot *int
		Result     string
	}

	GameHandler interface {
		GameType() dto.GameType
		Mode() Mode
		InitialState(roomID uuid.UUID, players []dto.GameRoomPlayer) (stateJSON string, firstTurnSlot int, err error)
		ValidateAction(stateJSON string, actorSlot int, action json.RawMessage) (ActionResult, error)
		OnGraceExpired(stateJSON string, playerSlot int) DisconnectResult
		ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error)
		ProjectState(stateJSON string, finished bool) (string, error)
		SupportsDraw() bool
	}

	BaseHandler struct{}
)

const (
	ModeTurnBased Mode = iota
	ModeConcurrent
)

func (BaseHandler) Mode() Mode {
	return ModeTurnBased
}

func (BaseHandler) ProjectState(stateJSON string, _ bool) (string, error) {
	return stateJSON, nil
}

func (BaseHandler) SupportsDraw() bool {
	return true
}
