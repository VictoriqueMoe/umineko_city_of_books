package pong

import (
	"encoding/json"
	"testing"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stateJSONOf(t *testing.T, s State) string {
	t.Helper()

	raw, err := json.Marshal(s)
	require.NoError(t, err)

	return string(raw)
}

func TestHandler_Identity(t *testing.T) {
	// given
	h := NewHandler()

	// when
	gameType := h.GameType()

	// then
	assert.Equal(t, dto.GameTypePong, gameType)
	assert.Equal(t, gameroom.ModeConcurrent, h.Mode())
	assert.False(t, h.SupportsDraw())
	assert.Equal(t, tickInterval, h.TickInterval())
	assert.Equal(t, snapshotEvery, h.SnapshotEvery())
	assert.Equal(t, snapshotType, h.SnapshotType())
}

func TestHandler_InitialStateStartsInCountdownWithNoTurn(t *testing.T) {
	// given
	h := NewHandler()

	// when
	raw, firstSlot, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// then
	assert.Equal(t, -1, firstSlot)

	s, err := loadState(raw)
	require.NoError(t, err)
	assert.Equal(t, phaseCountdown, s.Phase)
	assert.Equal(t, countdownMS, s.PhaseRemainMS)
	assert.Equal(t, boardWidth, s.Width)
	assert.Equal(t, boardHeight, s.Height)
	assert.Equal(t, ballRadius, s.BallRadius)
	assert.Equal(t, paddleWidth, s.PaddleWidth)
	assert.Equal(t, paddleHeight, s.PaddleHeight)
	assert.Equal(t, paddleInset, s.PaddleInset)
	assert.Equal(t, pointsToWin, s.PointsToWin)
	assert.Equal(t, [2]int{boardHeight / 2, boardHeight / 2}, s.PaddleY)
	assert.Equal(t, ballSpeedStart, s.BallSpeed)
	assert.NotEmpty(t, s.StartedAt)
	assert.NotZero(t, s.ServeVX)
	assert.Contains(t, []int{0, 1}, s.ServeToSlot)
}

func TestHandler_ValidateActionAlwaysFails(t *testing.T) {
	cases := []struct {
		name   string
		slot   int
		action string
	}{
		{"a paddle move", 0, `{"y":300}`},
		{"an empty object", 1, `{}`},
		{"rubbish", 0, `not json`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := NewHandler()
			state := stateJSONOf(t, State{Phase: phaseRally})

			// when
			res, err := h.ValidateAction(state, tc.slot, json.RawMessage(tc.action))

			// then
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no actions")
			assert.Equal(t, gameroom.ActionResult{}, res)
		})
	}
}

func TestHandler_OnGraceExpired(t *testing.T) {
	cases := []struct {
		name       string
		phase      string
		slot       int
		wantFinish bool
		wantWinner *int
		wantResult string
	}{
		{"countdown is abandoned", phaseCountdown, 0, true, nil, resultAbandoned},
		{"serve is a forfeit", phaseServe, 0, true, new(1), resultForfeit},
		{"rally is a forfeit", phaseRally, 1, true, new(0), resultForfeit},
		{"finished does nothing", phaseFinished, 0, false, nil, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := NewHandler()
			state := stateJSONOf(t, State{Phase: tc.phase})

			// when
			res := h.OnGraceExpired(state, tc.slot)

			// then
			assert.Equal(t, tc.wantFinish, res.Finished)
			assert.Equal(t, tc.wantResult, res.Result)
			if tc.wantWinner == nil {
				assert.Nil(t, res.WinnerSlot)
				return
			}
			require.NotNil(t, res.WinnerSlot)
			assert.Equal(t, *tc.wantWinner, *res.WinnerSlot)
		})
	}
}

func TestHandler_ComputeStats(t *testing.T) {
	cases := []struct {
		name       string
		reason     string
		result     string
		wantReason string
	}{
		{"state reason wins", resultWin, "forfeit", resultWin},
		{"result column is the fallback", "", "timeout", "timeout"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := NewHandler()
			state := stateJSONOf(t, State{
				Phase:        phaseFinished,
				Scores:       [2]int{7, 4},
				Hits:         [2]int{31, 27},
				LongestRally: [2]int{9, 6},
				TopSpeed:     1608,
				Reason:       tc.reason,
			})

			// when
			raw, err := h.ComputeStats(state, tc.result, "2026-08-26T10:00:00Z", "2026-08-26T10:04:30Z")
			require.NoError(t, err)

			// then
			stats, ok := raw.(Stats)
			require.True(t, ok)
			assert.Equal(t, 7, stats.PointsP0)
			assert.Equal(t, 4, stats.PointsP1)
			assert.Equal(t, 31, stats.HitsP0)
			assert.Equal(t, 27, stats.HitsP1)
			assert.Equal(t, 9, stats.LongestRallyP0)
			assert.Equal(t, 6, stats.LongestRallyP1)
			assert.Equal(t, 1608, stats.TopSpeed)
			assert.Equal(t, tc.wantReason, stats.ResultReason)
			assert.Equal(t, 270, stats.DurationSeconds)
		})
	}
}

func TestHandler_ComputeStatsRejectsBrokenState(t *testing.T) {
	// given
	h := NewHandler()

	// when
	_, err := h.ComputeStats("not json", "win", "", "")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load state")
}

func TestHandler_NewSimResumesAMidRallyStateOnAFreshServe(t *testing.T) {
	// given
	h := NewHandler()
	state := stateJSONOf(t, State{
		Phase:        phaseRally,
		Scores:       [2]int{4, 3},
		Hits:         [2]int{18, 15},
		LongestRally: [2]int{7, 5},
		TopSpeed:     1420,
		Ticks:        9000,
		BallX:        640,
		BallY:        220,
		BallVX:       -1200,
		BallVY:       400,
		BallSpeed:    1420,
		PaddleY:      [2]int{300, 520},
		ServeToSlot:  1,
		ServeVX:      870,
		ServeVY:      230,
	})

	// when
	gs, err := h.NewSim(uuid.New(), state)
	require.NoError(t, err)

	// then
	s, ok := gs.(*sim)
	require.True(t, ok)
	assert.Equal(t, phaseServe, s.st.Phase)
	assert.Equal(t, resumeCountdownMS*tickHz/1000, s.phaseTick)
	assert.Equal(t, [2]int{4, 3}, s.st.Scores)
	assert.Equal(t, [2]int{18, 15}, s.st.Hits)
	assert.Equal(t, 9000, s.st.Ticks)
	assert.InDelta(t, boardWidth/2.0, s.ballX, 1e-9)
	assert.InDelta(t, boardHeight/2.0, s.ballY, 1e-9)
	assert.InDelta(t, 0, s.ballVX, 1e-9)
	assert.InDelta(t, 0, s.ballVY, 1e-9)
	assert.Equal(t, 1, s.st.ServeToSlot)
	assert.Equal(t, 870, s.st.ServeVX)
	assert.Equal(t, 230, s.st.ServeVY)
	assert.InDelta(t, 300, s.paddleY[0], 1e-9)
	assert.InDelta(t, 520, s.target[1], 1e-9)
	assert.Equal(t, [2]bool{true, true}, s.connected)
}

func TestHandler_NewSimLeavesACountdownAlone(t *testing.T) {
	// given
	h := NewHandler()
	raw, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	gs, err := h.NewSim(uuid.New(), raw)
	require.NoError(t, err)

	// then
	s, ok := gs.(*sim)
	require.True(t, ok)
	assert.Equal(t, phaseCountdown, s.st.Phase)
	assert.Equal(t, countdownMS*tickHz/1000, s.phaseTick)
}

func TestHandler_NewSimRejectsBrokenState(t *testing.T) {
	// given
	h := NewHandler()

	// when
	_, err := h.NewSim(uuid.New(), "not json")

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load state")
}

func TestHandler_SatisfiesTheGameroomInterfaces(t *testing.T) {
	// given
	h := NewHandler()

	// when
	var gh gameroom.GameHandler = h
	var th gameroom.TickingHandler = h

	// then
	assert.Equal(t, dto.GameTypePong, gh.GameType())
	assert.Equal(t, tickInterval, th.TickInterval())
}
