package snakesandladders

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
	finalCell = 100
	diceSides = 6

	actionRoll = "roll"

	resultWin     = "win"
	resultForfeit = "forfeit"
)

var (
	ladders = map[int]int{
		1:  38,
		4:  14,
		9:  31,
		21: 42,
		28: 84,
		36: 44,
		51: 67,
		71: 91,
		80: 100,
	}

	snakes = map[int]int{
		16: 6,
		47: 26,
		49: 11,
		56: 53,
		62: 19,
		64: 60,
		87: 24,
		93: 73,
		95: 75,
		98: 78,
	}
)

type (
	Handler struct {
		gameroom.BaseHandler
	}

	state struct {
		Positions      [2]int    `json:"positions"`
		Turn           int       `json:"turn"`
		Rolls          int       `json:"rolls"`
		LaddersClimbed [2]int    `json:"ladders_climbed"`
		SnakesHit      [2]int    `json:"snakes_hit"`
		Last           *lastRoll `json:"last,omitempty"`
	}

	lastRoll struct {
		Slot    int `json:"slot"`
		Roll    int `json:"roll"`
		From    int `json:"from"`
		Stepped int `json:"stepped"`
		To      int `json:"to"`
	}

	action struct {
		Type string `json:"type"`
	}

	Stats struct {
		TotalRolls      int    `json:"total_rolls"`
		RollsP0         int    `json:"rolls_p0"`
		RollsP1         int    `json:"rolls_p1"`
		LaddersP0       int    `json:"ladders_p0"`
		LaddersP1       int    `json:"ladders_p1"`
		SnakesP0        int    `json:"snakes_p0"`
		SnakesP1        int    `json:"snakes_p1"`
		FinalP0         int    `json:"final_p0"`
		FinalP1         int    `json:"final_p1"`
		ResultReason    string `json:"result_reason"`
		DurationSeconds int    `json:"duration_seconds"`
	}
)

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GameType() dto.GameType {
	return dto.GameTypeSnakesLadders
}

func (h *Handler) SupportsDraw() bool {
	return false
}

func (h *Handler) InitialState(_ uuid.UUID, _ []dto.GameRoomPlayer) (string, int, error) {
	s := state{
		Turn: 0,
	}

	raw, err := json.Marshal(s)
	if err != nil {
		return "", 0, err
	}

	return string(raw), 0, nil
}

func resolveMove(from, roll int) (stepped int, to int, viaLadder bool, viaSnake bool) {
	stepped = from + roll
	to = stepped

	if stepped > finalCell {
		return from, from, false, false
	}

	if dest, ok := ladders[stepped]; ok {
		return stepped, dest, true, false
	}

	if dest, ok := snakes[stepped]; ok {
		return stepped, dest, false, true
	}

	return stepped, to, false, false
}

func (h *Handler) ValidateAction(stateJSON string, actorSlot int, raw json.RawMessage) (gameroom.ActionResult, error) {
	if actorSlot < 0 || actorSlot > 1 {
		return gameroom.ActionResult{}, errors.New("invalid actor slot")
	}

	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("load state: %w", err)
	}
	if s.Turn != actorSlot {
		return gameroom.ActionResult{}, errors.New("not your turn")
	}

	var act action
	if err := json.Unmarshal(raw, &act); err != nil {
		return gameroom.ActionResult{}, fmt.Errorf("parse action: %w", err)
	}
	if act.Type != actionRoll {
		return gameroom.ActionResult{}, fmt.Errorf("unknown action: %s", act.Type)
	}

	roll := rand.IntN(diceSides) + 1
	from := s.Positions[actorSlot]
	stepped, to, viaLadder, viaSnake := resolveMove(from, roll)

	if viaLadder {
		s.LaddersClimbed[actorSlot]++
	}
	if viaSnake {
		s.SnakesHit[actorSlot]++
	}

	s.Positions[actorSlot] = to
	s.Rolls++
	s.Last = &lastRoll{Slot: actorSlot, Roll: roll, From: from, Stepped: stepped, To: to}

	nextSlot := 1 - actorSlot
	s.Turn = nextSlot

	raw, err := json.Marshal(s)
	if err != nil {
		return gameroom.ActionResult{}, err
	}

	res := gameroom.ActionResult{NewStateJSON: string(raw)}
	if to == finalCell {
		res.Finished = true
		res.Result = resultWin
		res.WinnerSlot = new(actorSlot)
		return res, nil
	}

	res.NextTurnSlot = &nextSlot
	return res, nil
}

func (h *Handler) OnGraceExpired(_ string, disconnectedSlot int) gameroom.DisconnectResult {
	return gameroom.DisconnectResult{
		Finished:   true,
		WinnerSlot: new(1 - disconnectedSlot),
		Result:     resultForfeit,
	}
}

func (h *Handler) ComputeStats(stateJSON, result, createdAt, finishedAt string) (any, error) {
	var s state
	if err := json.Unmarshal([]byte(stateJSON), &s); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	rollsP0 := (s.Rolls + 1) / 2
	rollsP1 := s.Rolls / 2

	return Stats{
		TotalRolls:      s.Rolls,
		RollsP0:         rollsP0,
		RollsP1:         rollsP1,
		LaddersP0:       s.LaddersClimbed[0],
		LaddersP1:       s.LaddersClimbed[1],
		SnakesP0:        s.SnakesHit[0],
		SnakesP1:        s.SnakesHit[1],
		FinalP0:         s.Positions[0],
		FinalP1:         s.Positions[1],
		ResultReason:    result,
		DurationSeconds: durationSeconds(createdAt, finishedAt),
	}, nil
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
