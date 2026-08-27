package pong

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
)

const (
	boardWidth   = 1200
	boardHeight  = 800
	ballRadius   = 10
	paddleWidth  = 18
	paddleHeight = 130
	paddleInset  = 40

	faceX0 = paddleInset + paddleWidth
	faceX1 = boardWidth - paddleInset - paddleWidth

	paddleMaxSpeed = 1100
	ballSpeedStart = 900
	ballSpeedGain  = 1.06
	ballSpeedMax   = 1900

	maxBounceAngle = math.Pi / 3
	paddleEnglish  = 0.28
	serveAngleMin  = 15 * math.Pi / 180
	serveAngleMax  = 30 * math.Pi / 180

	pointsToWin  = 7
	winBy        = 2
	hardCapScore = 11

	tickHz       = 60
	tickInterval = time.Second / tickHz

	snapshotEvery     = 3
	countdownMS       = 3000
	serveDelayMS      = 1200
	resumeCountdownMS = 3000
	maxMatchTicks     = 54000

	phaseCountdown = "countdown"
	phaseServe     = "serve"
	phaseRally     = "rally"
	phaseFinished  = "finished"

	resultWin       = "win"
	resultForfeit   = "forfeit"
	resultAbandoned = "abandoned"
	resultTimeLimit = "time_limit"

	eventHitP0 = 1
	eventHitP1 = 2
	eventWall  = 4
	eventScore = 8
	eventServe = 16

	snapshotType = "game_pong_frame"
)

type (
	Handler struct {
		gameroom.BaseHandler
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypePong
}

func (h *Handler) Mode() gameroom.Mode {
	return gameroom.ModeConcurrent
}

func (h *Handler) SupportsDraw() bool {
	return false
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	s := newInitialState(rand.IntN, rand.Float64)

	raw, err := json.Marshal(s)
	if err != nil {
		return "", -1, err
	}

	return string(raw), -1, nil
}

func (h *Handler) ValidateAction(_ string, _ int, _ json.RawMessage) (gameroom.ActionResult, error) {
	return gameroom.ActionResult{}, errors.New("pong takes no actions")
}

func (h *Handler) OnGraceExpired(stateJSON string, slot int) gameroom.DisconnectResult {
	s, err := loadState(stateJSON)
	if err != nil {
		return gameroom.DisconnectResult{}
	}

	switch s.Phase {
	case phaseCountdown:
		return gameroom.DisconnectResult{
			Finished: true,
			Result:   resultAbandoned,
		}
	case phaseServe, phaseRally:
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

	reason := s.Reason
	if reason == "" {
		reason = result
	}

	return Stats{
		PointsP0:        s.Scores[0],
		PointsP1:        s.Scores[1],
		HitsP0:          s.Hits[0],
		HitsP1:          s.Hits[1],
		LongestRallyP0:  s.LongestRally[0],
		LongestRallyP1:  s.LongestRally[1],
		TopSpeed:        s.TopSpeed,
		ResultReason:    reason,
		DurationSeconds: gameroom.DurationSeconds(createdAt, finishedAt),
	}, nil
}

func (h *Handler) TickInterval() time.Duration {
	return tickInterval
}

func (h *Handler) SnapshotEvery() int {
	return snapshotEvery
}

func (h *Handler) SnapshotType() string {
	return snapshotType
}

func (h *Handler) NewSim(roomID uuid.UUID, stateJSON string) (gameroom.GameSim, error) {
	seed := binary.BigEndian.Uint64(roomID[8:])

	s, err := newSim(roomID, stateJSON, rand.New(rand.NewPCG(seed, seed^0xdeadbeef)))
	if err != nil {
		return nil, err
	}

	return s, nil
}

func loadState(stateJSON string) (*State, error) {
	var s State
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	return &s, nil
}
