package pong

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRng() *rand.Rand {
	const seed uint64 = 42

	return rand.New(rand.NewPCG(seed, seed^0xdeadbeef))
}

func newTestSim(t *testing.T) *sim {
	t.Helper()

	st := newInitialState(func(int) int { return 0 }, func() float64 { return 0.5 })

	raw, err := json.Marshal(st)
	require.NoError(t, err)

	s, err := newSim(uuid.New(), string(raw), testRng())
	require.NoError(t, err)

	return s
}

func rallySim(t *testing.T) *sim {
	t.Helper()

	s := newTestSim(t)
	s.st.Phase = phaseRally
	s.phaseTick = 0

	return s
}

func TestSim_SweptCollisionCatchesBallAtMaxSpeed(t *testing.T) {
	cases := []struct {
		name     string
		slot     int
		ballX    float64
		ballVX   float64
		wantHits [2]int
	}{
		{"left paddle", 0, faceX0 + ballRadius + 20, -ballSpeedMax, [2]int{1, 0}},
		{"right paddle", 1, faceX1 - ballRadius - 20, ballSpeedMax, [2]int{0, 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.speed = ballSpeedMax
			s.ballX = tc.ballX
			s.ballY = boardHeight / 2
			s.ballVX = tc.ballVX
			s.ballVY = 0
			s.paddleY[tc.slot] = boardHeight / 2
			s.target[tc.slot] = boardHeight / 2

			// when
			res := s.Tick(tickInterval)

			// then
			assert.False(t, res.Scored)
			assert.Equal(t, tc.wantHits, s.st.Hits)
			assert.Equal(t, 1, s.st.RallyHits)
			if tc.slot == 0 {
				assert.Positive(t, s.ballVX)
			} else {
				assert.Negative(t, s.ballVX)
			}
		})
	}
}

func TestSim_SweptCollisionCatchesBallCrossingWallAndFaceInOneTick(t *testing.T) {
	cases := []struct {
		name     string
		slot     int
		ballX    float64
		ballVX   float64
		ballY    float64
		ballVY   float64
		paddleY  float64
		wantHits [2]int
	}{
		{"left paddle at the top wall", 0, 80, -800, ballRadius, -1640, paddleHeight / 2.0, [2]int{1, 0}},
		{"right paddle at the bottom wall", 1, 1120, 800, boardHeight - ballRadius, 1640, boardHeight - paddleHeight/2.0, [2]int{0, 1}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.speed = ballSpeedMax
			s.ballX = tc.ballX
			s.ballY = tc.ballY
			s.ballVX = tc.ballVX
			s.ballVY = tc.ballVY
			s.paddleY[tc.slot] = tc.paddleY
			s.target[tc.slot] = tc.paddleY

			// when
			res := s.Tick(tickInterval)

			// then
			assert.False(t, res.Scored)
			assert.Equal(t, tc.wantHits, s.st.Hits)
			assert.Equal(t, 1, s.st.RallyHits)
			assert.GreaterOrEqual(t, s.ballY, float64(ballRadius))
			assert.LessOrEqual(t, s.ballY, float64(boardHeight-ballRadius))
		})
	}
}

func TestSim_WallCollisionReflectsAndStaysInCourt(t *testing.T) {
	cases := []struct {
		name    string
		ballY   float64
		ballVY  float64
		wantPos bool
	}{
		{"top wall", ballRadius + 2, -ballSpeedStart, true},
		{"bottom wall", boardHeight - ballRadius - 2, ballSpeedStart, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.ballX = boardWidth / 2
			s.ballY = tc.ballY
			s.ballVX = 0
			s.ballVY = tc.ballVY

			// when
			s.Tick(tickInterval)

			// then
			assert.NotZero(t, s.events&eventWall)
			assert.GreaterOrEqual(t, s.ballY, float64(ballRadius))
			assert.LessOrEqual(t, s.ballY, float64(boardHeight-ballRadius))
			if tc.wantPos {
				assert.Positive(t, s.ballVY)
			} else {
				assert.Negative(t, s.ballVY)
			}
		})
	}
}

func TestSim_BounceSetsAngleFromPaddleOffset(t *testing.T) {
	cases := []struct {
		name      string
		contactY  float64
		wantAngle float64
	}{
		{"centre is flat", boardHeight / 2, 0},
		{"half way up", boardHeight/2 - paddleHeight/4.0, -maxBounceAngle / 2},
		{"top edge", boardHeight/2 - paddleHeight/2.0, -maxBounceAngle},
		{"bottom edge", boardHeight/2 + paddleHeight/2.0, maxBounceAngle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.speed = ballSpeedStart
			s.paddleY[0] = boardHeight / 2
			s.paddleVY[0] = 0

			// when
			s.bounce(0, tc.contactY)

			// then
			assert.InDelta(t, tc.wantAngle, math.Atan2(s.ballVY, math.Abs(s.ballVX)), 1e-9)
			assert.InDelta(t, ballSpeedStart*ballSpeedGain, math.Hypot(s.ballVX, s.ballVY), 1e-9)
			assert.Positive(t, s.ballVX)
		})
	}
}

func TestSim_BounceEnglishIsClampedAndKeepsMagnitude(t *testing.T) {
	cases := []struct {
		name     string
		contactY float64
		paddleVY float64
		minAngle float64
		maxAngle float64
	}{
		{"english tilts a flat hit", boardHeight / 2, paddleMaxSpeed / 2, 0.01, maxBounceAngle - 0.01},
		{"english cannot exceed the clamp", boardHeight/2 + paddleHeight/2.0, paddleMaxSpeed, maxBounceAngle, maxBounceAngle},
		{"downward english cannot exceed the clamp", boardHeight/2 - paddleHeight/2.0, -paddleMaxSpeed, -maxBounceAngle, -maxBounceAngle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.speed = ballSpeedStart
			s.paddleY[0] = boardHeight / 2
			s.paddleVY[0] = tc.paddleVY

			// when
			s.bounce(0, tc.contactY)

			// then
			angle := math.Atan2(s.ballVY, math.Abs(s.ballVX))
			assert.GreaterOrEqual(t, angle, tc.minAngle-1e-9)
			assert.LessOrEqual(t, angle, tc.maxAngle+1e-9)
			assert.InDelta(t, ballSpeedStart*ballSpeedGain, math.Hypot(s.ballVX, s.ballVY), 1e-9)
		})
	}
}

func TestSim_SpeedRampReachesCapAndHolds(t *testing.T) {
	// given
	s := rallySim(t)
	s.speed = ballSpeedStart
	s.paddleY[0] = boardHeight / 2

	// when
	for range 12 {
		s.bounce(0, boardHeight/2)
	}
	after12 := s.speed

	s.bounce(0, boardHeight/2)
	after13 := s.speed

	for range 5 {
		s.bounce(0, boardHeight/2)
	}

	// then
	assert.Less(t, after12, float64(ballSpeedMax))
	assert.InDelta(t, float64(ballSpeedMax), after13, 1e-9)
	assert.InDelta(t, float64(ballSpeedMax), s.speed, 1e-9)
	assert.Equal(t, ballSpeedMax, s.st.TopSpeed)
}

func TestSim_ChasingPaddleDoesNotRecollide(t *testing.T) {
	cases := []struct {
		name       string
		target     float64
		wantHits   int
		wantBounce bool
	}{
		{
			name:       "a paddle still within reach hits once and never twice",
			target:     boardHeight/2 + 40,
			wantHits:   1,
			wantBounce: true,
		},
		{
			name:       "a paddle moved out of reach misses, because it tracks the pointer exactly",
			target:     boardHeight/2 + 200,
			wantHits:   0,
			wantBounce: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.speed = ballSpeedStart
			s.ballX = faceX0 + ballRadius + 5
			s.ballY = boardHeight / 2
			s.ballVX = -ballSpeedStart
			s.ballVY = 0
			s.paddleY[0] = boardHeight / 2
			s.target[0] = tt.target

			// when
			s.Tick(tickInterval)
			hitsAfterFirst := s.st.Hits[0]
			s.target[0] = s.ballY
			s.Tick(tickInterval)

			// then
			assert.Equal(t, tt.wantHits, hitsAfterFirst)
			assert.Equal(t, tt.wantHits, s.st.Hits[0])
			if tt.wantBounce {
				assert.GreaterOrEqual(t, s.ballX, float64(faceX0+ballRadius))
			}
		})
	}
}

func TestSim_PaddleTracksTheTargetExactly(t *testing.T) {
	// given
	s := rallySim(t)
	s.paddleY[0] = boardHeight / 2
	s.target[0] = boardHeight / 2

	// when
	s.target[0] = paddleHeight / 2
	s.Tick(tickInterval)

	// then
	assert.InDelta(t, paddleHeight/2.0, s.paddleY[0], 1e-9)
	assert.InDelta(t, -paddleMaxSpeed, s.paddleVY[0], 1e-9)
}

func TestSim_ConcededPointCreditsTheOtherSlot(t *testing.T) {
	// given
	s := rallySim(t)
	s.ballX = 2
	s.ballY = boardHeight / 2
	s.ballVX = -ballSpeedStart
	s.ballVY = 0
	s.st.RallyHits = 4

	// when
	res := s.Tick(tickInterval)

	// then
	assert.True(t, res.Scored)
	assert.False(t, res.Finished)
	assert.Equal(t, [2]int{0, 1}, s.st.Scores)
	assert.Equal(t, [2]int{0, 4}, s.st.LongestRally)
	assert.Equal(t, 0, s.st.RallyHits)
	assert.Equal(t, 0, s.st.ServeToSlot)
	assert.Equal(t, phaseServe, s.st.Phase)
	assert.NotZero(t, s.events&eventScore)
}

func TestSim_WinConditionNeedsMarginOrHardCap(t *testing.T) {
	cases := []struct {
		name       string
		scores     [2]int
		winner     int
		wantFinish bool
		wantScores [2]int
	}{
		{"target reached with margin", [2]int{6, 0}, 0, true, [2]int{7, 0}},
		{"seven six is a deuce", [2]int{6, 6}, 0, false, [2]int{7, 6}},
		{"hard cap ends a deuce", [2]int{10, 10}, 1, true, [2]int{10, 11}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.st.Scores = tc.scores

			// when
			res := s.scorePoint(tc.winner)

			// then
			assert.Equal(t, tc.wantScores, s.st.Scores)
			assert.Equal(t, tc.wantFinish, res.Finished)
			if !tc.wantFinish {
				assert.True(t, res.Scored)
				assert.Equal(t, phaseServe, s.st.Phase)
				return
			}
			require.NotNil(t, res.WinnerSlot)
			assert.Equal(t, tc.winner, *res.WinnerSlot)
			assert.Equal(t, resultWin, res.Result)
			assert.Equal(t, phaseFinished, s.st.Phase)
		})
	}
}

func TestSim_TimeLimitAlwaysProducesAWinner(t *testing.T) {
	cases := []struct {
		name       string
		scores     [2]int
		hits       [2]int
		wantWinner int
	}{
		{"higher score wins", [2]int{3, 5}, [2]int{0, 0}, 1},
		{"level score breaks on hits", [2]int{4, 4}, [2]int{9, 2}, 0},
		{"level on everything falls to slot zero", [2]int{2, 2}, [2]int{5, 5}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			s.st.Scores = tc.scores
			s.st.Hits = tc.hits
			s.st.Ticks = maxMatchTicks - 1

			// when
			res := s.Tick(tickInterval)

			// then
			require.True(t, res.Finished)
			require.NotNil(t, res.WinnerSlot)
			assert.Equal(t, tc.wantWinner, *res.WinnerSlot)
			assert.Equal(t, resultTimeLimit, res.Result)
			assert.Equal(t, phaseFinished, s.st.Phase)
		})
	}
}

func TestSim_ApplyInputClampsTheTarget(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		want    float64
	}{
		{"above the court", `{"y":-5000,"seq":1}`, paddleHeight / 2.0},
		{"below the court", `{"y":999999,"seq":1}`, boardHeight - paddleHeight/2.0},
		{"inside the court", `{"y":300,"seq":1}`, 300},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)

			// when
			s.ApplyInput(0, json.RawMessage(tc.payload))

			// then
			assert.InDelta(t, tc.want, s.target[0], 1e-9)
		})
	}
}

func TestSim_ApplyInputIgnoresBadSlotsAndPayloads(t *testing.T) {
	cases := []struct {
		name    string
		slot    int
		payload string
	}{
		{"slot below range", -1, `{"y":300,"seq":1}`},
		{"slot above range", 2, `{"y":300,"seq":1}`},
		{"malformed payload", 0, `not json`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			s := rallySim(t)
			before := s.target

			// when
			s.ApplyInput(tc.slot, json.RawMessage(tc.payload))

			// then
			assert.Equal(t, before, s.target)
		})
	}
}

func TestSim_ApplyInputKeepsOnlyTheLatestTarget(t *testing.T) {
	// given
	s := rallySim(t)

	// when
	s.ApplyInput(1, json.RawMessage(`{"y":200,"seq":8}`))
	s.ApplyInput(1, json.RawMessage(`{"y":500,"seq":9}`))

	// then
	assert.InDelta(t, 500, s.target[1], 1e-9)
	assert.Equal(t, 9, s.ackSeq[1])
}

func TestSim_ApplyInputNeverRejectsOnSeq(t *testing.T) {
	// given
	s := rallySim(t)
	s.ApplyInput(0, json.RawMessage(`{"y":200,"seq":900}`))

	// when
	s.ApplyInput(0, json.RawMessage(`{"y":460,"seq":1}`))

	// then
	assert.InDelta(t, 460, s.target[0], 1e-9)
	assert.Equal(t, 1, s.ackSeq[0])
}

func TestSim_SetConnectedFreezesTheTarget(t *testing.T) {
	// given
	s := rallySim(t)
	s.paddleY[1] = 210
	s.target[1] = 700

	// when
	s.SetConnected(1, false)

	// then
	assert.InDelta(t, 210, s.target[1], 1e-9)
	assert.Equal(t, [2]bool{true, false}, s.connected)
}

func TestSim_SnapshotClearsEventsAndReportsTheClock(t *testing.T) {
	// given
	s := rallySim(t)
	s.st.Ticks = 120
	s.events = eventWall | eventScore

	// when
	first, ok := s.Snapshot().(frame)
	require.True(t, ok)
	second, ok := s.Snapshot().(frame)
	require.True(t, ok)

	// then
	assert.Equal(t, 2000, first.T)
	assert.Equal(t, eventWall|eventScore, first.Events)
	assert.Equal(t, 0, second.Events)
	assert.Equal(t, s.roomID.String(), first.RoomID)
}

func TestSim_StateJSONRoundsTheLiveFloats(t *testing.T) {
	// given
	s := rallySim(t)
	s.ballX = 123.6
	s.ballY = 400.4
	s.ballVX = -899.7
	s.ballVY = 12.2
	s.speed = 954.4
	s.paddleY = [2]float64{100.5, 700.4}
	s.phaseTick = 60

	// when
	raw, err := s.StateJSON()
	require.NoError(t, err)

	// then
	st, err := loadState(raw)
	require.NoError(t, err)
	assert.Equal(t, 124, st.BallX)
	assert.Equal(t, 400, st.BallY)
	assert.Equal(t, -900, st.BallVX)
	assert.Equal(t, 12, st.BallVY)
	assert.Equal(t, 954, st.BallSpeed)
	assert.Equal(t, [2]int{101, 700}, st.PaddleY)
	assert.Equal(t, 1000, st.PhaseRemainMS)
}

func TestSim_CountdownServesAfterTheDelay(t *testing.T) {
	// given
	s := newTestSim(t)
	require.Equal(t, phaseCountdown, s.st.Phase)
	require.Equal(t, countdownMS*tickHz/1000, s.phaseTick)

	// when
	for range countdownMS * tickHz / 1000 {
		s.Tick(tickInterval)
	}

	// then
	assert.Equal(t, phaseRally, s.st.Phase)
	assert.NotZero(t, s.events&eventServe)
	assert.InDelta(t, float64(s.st.ServeVX), s.ballVX, 1e-9)
	assert.InDelta(t, float64(s.st.ServeVY), s.ballVY, 1e-9)
	assert.InDelta(t, float64(ballSpeedStart), s.speed, 1e-9)
}
