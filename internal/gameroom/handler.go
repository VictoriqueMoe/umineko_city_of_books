package gameroom

import (
	"encoding/json"
	"time"

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

	TickResult struct {
		Scored     bool
		Finished   bool
		WinnerSlot *int
		Result     string
	}

	GameSim interface {
		Tick(step time.Duration) TickResult
		ApplyInput(slot int, payload json.RawMessage)
		SetConnected(slot int, connected bool)
		Snapshot() any
		StateJSON() (string, error)
	}

	TickingHandler interface {
		TickInterval() time.Duration
		SnapshotEvery() int
		SnapshotType() string
		NewSim(roomID uuid.UUID, stateJSON string) (GameSim, error)
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
