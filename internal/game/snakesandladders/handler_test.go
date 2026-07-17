package snakesandladders

import (
	"encoding/json"
	"testing"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rollAction(t *testing.T) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(action{Type: actionRoll})
	require.NoError(t, err)
	return raw
}

func TestHandler_GameTypeAndDraw(t *testing.T) {
	// given
	h := NewHandler()

	// then
	assert.Equal(t, dto.GameTypeSnakesLadders, h.GameType())
	assert.False(t, h.SupportsDraw())
}

func TestInitialState_StartsAtZeroForSlotZero(t *testing.T) {
	// given
	h := NewHandler()

	// when
	stateJSON, firstSlot, err := h.InitialState(uuid.New(), nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, 0, firstSlot)

	var s state
	require.NoError(t, json.Unmarshal([]byte(stateJSON), &s))
	assert.Equal(t, [2]int{0, 0}, s.Positions)
	assert.Equal(t, 0, s.Turn)
}

func TestValidateAction_RejectsOutOfTurn(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	_, err = h.ValidateAction(stateJSON, 1, rollAction(t))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")
}

func TestValidateAction_RejectsUnknownAction(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	_, err = h.ValidateAction(stateJSON, 0, json.RawMessage(`{"type":"teleport"}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestValidateAction_RollMovesAndPassesTurn(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	res, err := h.ValidateAction(stateJSON, 0, rollAction(t))

	// then
	require.NoError(t, err)
	assert.False(t, res.Finished)
	require.NotNil(t, res.NextTurnSlot)
	assert.Equal(t, 1, *res.NextTurnSlot)

	var s state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &s))
	assert.Equal(t, 1, s.Turn)
	assert.Equal(t, 1, s.Rolls)
	require.NotNil(t, s.Last)
	assert.GreaterOrEqual(t, s.Last.Roll, 1)
	assert.LessOrEqual(t, s.Last.Roll, diceSides)
	assert.Equal(t, s.Positions[0], s.Last.To)
}

func TestValidateAction_OvershootKeepsPlayerPut(t *testing.T) {
	// given: from 99 a roll of 1 wins, any other roll overshoots 100 and stays put
	h := NewHandler()
	s := state{Positions: [2]int{99, 0}, Turn: 0}
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when
	res, err := h.ValidateAction(string(raw), 0, rollAction(t))

	// then
	require.NoError(t, err)

	var next state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &next))
	require.NotNil(t, next.Last)
	if next.Last.Roll == 1 {
		assert.True(t, res.Finished)
		assert.Equal(t, 100, next.Positions[0])
	} else {
		assert.False(t, res.Finished)
		assert.Equal(t, 99, next.Positions[0])
		assert.Equal(t, 99, next.Last.Stepped)
	}
}

func TestResolveMove(t *testing.T) {
	tests := []struct {
		name          string
		from          int
		roll          int
		wantStepped   int
		wantTo        int
		wantViaLadder bool
		wantViaSnake  bool
	}{
		{name: "plain step", from: 0, roll: 3, wantStepped: 3, wantTo: 3},
		{name: "ladder bottom climbs to top", from: 79, roll: 1, wantStepped: 80, wantTo: 100, wantViaLadder: true},
		{name: "snake head slides to tail", from: 15, roll: 1, wantStepped: 16, wantTo: 6, wantViaSnake: true},
		{name: "exact finish", from: 96, roll: 4, wantStepped: 100, wantTo: 100},
		{name: "overshoot stays put", from: 99, roll: 4, wantStepped: 99, wantTo: 99},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// when
			stepped, to, viaLadder, viaSnake := resolveMove(tc.from, tc.roll)

			// then
			assert.Equal(t, tc.wantStepped, stepped)
			assert.Equal(t, tc.wantTo, to)
			assert.Equal(t, tc.wantViaLadder, viaLadder)
			assert.Equal(t, tc.wantViaSnake, viaSnake)
		})
	}
}

func TestValidateAction_LadderClimbWinsAndCountsClimb(t *testing.T) {
	// given: from 79, every roll of 1 ladders 80 -> 100 and wins
	h := NewHandler()
	s := state{Positions: [2]int{79, 0}, Turn: 0}
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when we roll until we land the deciding 1
	for range 500 {
		res, verr := h.ValidateAction(string(raw), 0, rollAction(t))
		require.NoError(t, verr)

		var next state
		require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &next))
		if next.Last.Roll != 1 {
			continue
		}

		// then
		assert.True(t, res.Finished)
		assert.Equal(t, resultWin, res.Result)
		require.NotNil(t, res.WinnerSlot)
		assert.Equal(t, 0, *res.WinnerSlot)
		assert.Equal(t, 100, next.Positions[0])
		assert.Equal(t, 1, next.LaddersClimbed[0])
		return
	}
	t.Fatal("expected to observe a roll of 1 within 500 attempts")
}

func TestOnGraceExpired_OtherPlayerWins(t *testing.T) {
	// given
	h := NewHandler()

	// when
	res := h.OnGraceExpired("", 0)

	// then
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, 1, *res.WinnerSlot)
	assert.Equal(t, resultForfeit, res.Result)
}

func TestComputeStats_SplitsRolls(t *testing.T) {
	// given
	h := NewHandler()
	s := state{Positions: [2]int{100, 64}, Rolls: 7, LaddersClimbed: [2]int{2, 1}, SnakesHit: [2]int{1, 3}}
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when
	out, err := h.ComputeStats(string(raw), resultWin, "", "")

	// then
	require.NoError(t, err)
	stats, ok := out.(Stats)
	require.True(t, ok)
	assert.Equal(t, 7, stats.TotalRolls)
	assert.Equal(t, 4, stats.RollsP0)
	assert.Equal(t, 3, stats.RollsP1)
	assert.Equal(t, 100, stats.FinalP0)
	assert.Equal(t, resultWin, stats.ResultReason)
}
