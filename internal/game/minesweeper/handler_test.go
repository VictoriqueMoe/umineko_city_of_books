package minesweeper

import (
	"encoding/json"
	"testing"
	"time"

	"umineko_city_of_books/internal/dto"
	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unmarshalState(t *testing.T, raw string) *State {
	t.Helper()
	var s State
	require.NoError(t, json.Unmarshal([]byte(raw), &s))
	return &s
}

func actionJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}

func TestHandler_InitialState(t *testing.T) {
	// given
	h := NewHandler()

	// when
	stateJSON, firstTurn, err := h.InitialState(uuid.New(), nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, -1, firstTurn)
	s := unmarshalState(t, stateJSON)
	assert.Equal(t, phaseCharSelect, s.Phase)
	assert.Equal(t, defaultWidth, s.Width)
	assert.Equal(t, defaultHeight, s.Height)
	assert.Equal(t, defaultMineCount, s.MineCount)
	assert.Equal(t, "", s.Characters[0])
	assert.Equal(t, "", s.Characters[1])
	assert.False(t, s.MinesPlaced)
	assert.Nil(t, s.Mines)
}

func TestHandler_Mode_And_Draw(t *testing.T) {
	// given
	h := NewHandler()

	// when / then
	assert.Equal(t, gameroom.ModeConcurrent, h.Mode())
	assert.False(t, h.SupportsDraw())
	assert.Equal(t, dto.GameTypeMinesweeper, h.GameType())
}

func TestHandler_SelectCharacter(t *testing.T) {
	cases := []struct {
		name        string
		startPhase  string
		picks       [2]string
		actor       int
		character   string
		wantErr     string
		wantPhase   string
		wantStarted bool
	}{
		{
			name:      "first pick stays in char_select",
			picks:     [2]string{"", ""},
			actor:     0,
			character: characterBernkastel,
			wantPhase: phaseCharSelect,
		},
		{
			name:        "second pick transitions to playing",
			picks:       [2]string{characterBernkastel, ""},
			actor:       1,
			character:   characterErika,
			wantPhase:   phasePlaying,
			wantStarted: true,
		},
		{
			name:      "re-selection within char_select is allowed",
			picks:     [2]string{characterBernkastel, ""},
			actor:     0,
			character: characterLambdadelta,
			wantPhase: phaseCharSelect,
		},
		{
			name:      "unknown character rejected",
			picks:     [2]string{"", ""},
			actor:     0,
			character: "ronove",
			wantErr:   "unknown character",
		},
		{
			name:       "selecting while playing is rejected",
			startPhase: phasePlaying,
			picks:      [2]string{characterBernkastel, characterErika},
			actor:      0,
			character:  characterDlanor,
			wantErr:    "character selection is closed",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := NewHandler()
			stateJSON, _, err := h.InitialState(uuid.New(), nil)
			require.NoError(t, err)
			s := unmarshalState(t, stateJSON)
			s.Characters = tc.picks
			if tc.startPhase != "" {
				s.Phase = tc.startPhase
			}
			raw, err := json.Marshal(s)
			require.NoError(t, err)

			// when
			result, err := h.ValidateAction(string(raw), tc.actor, actionJSON(t, map[string]any{
				"type":      actionSelectCharacter,
				"character": tc.character,
			}))

			// then
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			after := unmarshalState(t, result.NewStateJSON)
			assert.Equal(t, tc.wantPhase, after.Phase)
			if tc.wantStarted {
				assert.NotEmpty(t, after.StartedAt)
			}
		})
	}
}

func startedState(t *testing.T, h *Handler) string {
	t.Helper()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)
	s := unmarshalState(t, stateJSON)
	s.Phase = phasePlaying
	s.Characters = [2]string{characterBernkastel, characterErika}
	s.StartedAt = time.Now().UTC().Format(time.RFC3339)
	raw, err := json.Marshal(s)
	require.NoError(t, err)
	return string(raw)
}

func TestHandler_Reveal_FirstClickFlow(t *testing.T) {
	// given a freshly-started game where neither player has clicked
	h := NewHandler()
	stateJSON := startedState(t, h)

	// when slot 0 makes the first reveal
	result, err := h.ValidateAction(stateJSON, 0, actionJSON(t, map[string]any{"type": actionReveal, "x": 3, "y": 4}))
	require.NoError(t, err)

	// then mines are NOT placed yet — waiting for slot 1
	s := unmarshalState(t, result.NewStateJSON)
	assert.False(t, s.MinesPlaced)
	assert.False(t, result.Finished)
	require.NotNil(t, s.PendingClicks[0])
	assert.Equal(t, [2]int{3, 4}, *s.PendingClicks[0])
	assert.Nil(t, s.PendingClicks[1])
	assert.Equal(t, 0, s.RevealedCount[0])

	// when slot 1 also makes their first reveal
	result, err = h.ValidateAction(result.NewStateJSON, 1, actionJSON(t, map[string]any{"type": actionReveal, "x": 10, "y": 10}))
	require.NoError(t, err)

	// then mines are placed and both players have flood-fill applied
	s = unmarshalState(t, result.NewStateJSON)
	assert.True(t, s.MinesPlaced)
	assert.NotNil(t, s.Mines)
	assert.Equal(t, defaultWidth*defaultHeight, len(s.Mines))
	mineCount := 0
	for i := 0; i < len(s.Mines); i++ {
		if s.Mines[i] {
			mineCount++
		}
	}
	assert.Equal(t, defaultMineCount, mineCount)
	assert.True(t, s.Revealed[0][s.idx(3, 4)])
	assert.True(t, s.Revealed[1][s.idx(10, 10)])
	assert.False(t, s.Mines[s.idx(3, 4)])
	assert.False(t, s.Mines[s.idx(10, 10)])
	assert.Greater(t, s.RevealedCount[0], 0)
	assert.Greater(t, s.RevealedCount[1], 0)
}

func TestHandler_Reveal_MineHit_OpponentWins(t *testing.T) {
	// given mines have already been placed and slot 0 is about to step on one
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	total := s.Width * s.Height
	s.Mines = make([]bool, total)
	s.MinesPlaced = true
	mineX, mineY := 5, 5
	s.Mines[s.idx(mineX, mineY)] = true
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when slot 0 reveals the mine cell
	result, err := h.ValidateAction(string(raw), 0, actionJSON(t, map[string]any{"type": actionReveal, "x": mineX, "y": mineY}))

	// then the game ends and slot 1 wins
	require.NoError(t, err)
	assert.True(t, result.Finished)
	require.NotNil(t, result.WinnerSlot)
	assert.Equal(t, 1, *result.WinnerSlot)
	assert.Equal(t, resultMineHit, result.Result)
	after := unmarshalState(t, result.NewStateJSON)
	require.NotNil(t, after.HitMineX)
	require.NotNil(t, after.HitMineY)
	assert.Equal(t, mineX, *after.HitMineX)
	assert.Equal(t, mineY, *after.HitMineY)
}

func TestHandler_Reveal_BoardComplete_ActorWins(t *testing.T) {
	// given a state where slot 0 only has one safe cell left
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	total := s.Width * s.Height
	s.Mines = make([]bool, total)
	s.MinesPlaced = true
	for i := 0; i < s.MineCount; i++ {
		s.Mines[i] = true
	}
	for i := s.MineCount; i < total-1; i++ {
		x := i % s.Width
		y := i / s.Width
		s.Revealed[0][s.idx(x, y)] = true
		s.RevealedCount[0]++
	}
	lastIdx := total - 1
	lastX := lastIdx % s.Width
	lastY := lastIdx / s.Width
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when slot 0 reveals the final safe cell
	result, err := h.ValidateAction(string(raw), 0, actionJSON(t, map[string]any{"type": actionReveal, "x": lastX, "y": lastY}))

	// then slot 0 wins by completion
	require.NoError(t, err)
	assert.True(t, result.Finished)
	require.NotNil(t, result.WinnerSlot)
	assert.Equal(t, 0, *result.WinnerSlot)
	assert.Equal(t, resultComplete, result.Result)
}

func TestHandler_Reveal_NoOpOnAlreadyRevealed(t *testing.T) {
	// given slot 0 has already revealed cell (3,3)
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.Mines = make([]bool, s.Width*s.Height)
	s.MinesPlaced = true
	s.Revealed[0][s.idx(3, 3)] = true
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when slot 0 tries to reveal it again
	result, err := h.ValidateAction(string(raw), 0, actionJSON(t, map[string]any{"type": actionReveal, "x": 3, "y": 3}))

	// then no error, no finish, state mostly unchanged
	require.NoError(t, err)
	assert.False(t, result.Finished)
}

func TestHandler_Flag_Toggle(t *testing.T) {
	// given mines placed and cell (2,2) is hidden
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.Mines = make([]bool, s.Width*s.Height)
	s.MinesPlaced = true
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when slot 0 flags (2,2)
	result, err := h.ValidateAction(string(raw), 0, actionJSON(t, map[string]any{"type": actionFlag, "x": 2, "y": 2}))
	require.NoError(t, err)
	after := unmarshalState(t, result.NewStateJSON)
	assert.True(t, after.Flagged[0][s.idx(2, 2)])

	// when slot 0 toggles the flag off
	result, err = h.ValidateAction(result.NewStateJSON, 0, actionJSON(t, map[string]any{"type": actionFlag, "x": 2, "y": 2}))
	require.NoError(t, err)
	after = unmarshalState(t, result.NewStateJSON)
	assert.False(t, after.Flagged[0][s.idx(2, 2)])
}

func TestHandler_Flag_RejectsRevealedCell(t *testing.T) {
	// given cell (1,1) is already revealed for slot 0
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.Mines = make([]bool, s.Width*s.Height)
	s.MinesPlaced = true
	s.Revealed[0][s.idx(1, 1)] = true
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when slot 0 tries to flag it
	_, err = h.ValidateAction(string(raw), 0, actionJSON(t, map[string]any{"type": actionFlag, "x": 1, "y": 1}))

	// then error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revealed cell")
}

func TestHandler_ValidateAction_UnknownAction(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON := startedState(t, h)

	// when
	_, err := h.ValidateAction(stateJSON, 0, json.RawMessage(`{"type":"explode"}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown action")
}

func TestHandler_ProjectState_StripsSecretsDuringPlay(t *testing.T) {
	// given a state with mines placed and pending clicks recorded
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.Mines = make([]bool, s.Width*s.Height)
	s.MinesPlaced = true
	s.Mines[s.idx(0, 0)] = true
	pc := [2]int{1, 1}
	s.PendingClicks[0] = &pc
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when projecting during active play
	projected, err := h.ProjectState(string(raw), false)
	require.NoError(t, err)

	// then mines are stripped but pending clicks remain so the actor can see their own click
	after := unmarshalState(t, projected)
	assert.Nil(t, after.Mines)
	require.NotNil(t, after.PendingClicks[0])
	assert.Equal(t, [2]int{1, 1}, *after.PendingClicks[0])
	assert.True(t, after.MinesPlaced)
}

func TestHandler_ProjectState_ExposesSecretsWhenFinished(t *testing.T) {
	// given a finished state with mines placed
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.Phase = phaseFinished
	s.Mines = make([]bool, s.Width*s.Height)
	s.MinesPlaced = true
	s.Mines[s.idx(0, 0)] = true
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	// when projecting after finish
	projected, err := h.ProjectState(string(raw), true)
	require.NoError(t, err)

	// then mines are exposed
	after := unmarshalState(t, projected)
	require.NotNil(t, after.Mines)
	assert.True(t, after.Mines[after.idx(0, 0)])
}

func TestHandler_OnGraceExpired(t *testing.T) {
	cases := []struct {
		name       string
		phase      string
		slot       int
		wantFinish bool
		wantResult string
		wantWinner *int
	}{
		{
			name:       "char_select abandons without winner",
			phase:      phaseCharSelect,
			slot:       0,
			wantFinish: true,
			wantResult: resultAbandoned,
			wantWinner: nil,
		},
		{
			name:       "playing slot 0 forfeits to slot 1",
			phase:      phasePlaying,
			slot:       0,
			wantFinish: true,
			wantResult: resultForfeit,
			wantWinner: new(1),
		},
		{
			name:       "playing slot 1 forfeits to slot 0",
			phase:      phasePlaying,
			slot:       1,
			wantFinish: true,
			wantResult: resultForfeit,
			wantWinner: new(0),
		},
		{
			name:       "finished phase is no-op",
			phase:      phaseFinished,
			slot:       0,
			wantFinish: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given
			h := NewHandler()
			stateJSON, _, err := h.InitialState(uuid.New(), nil)
			require.NoError(t, err)
			s := unmarshalState(t, stateJSON)
			s.Phase = tc.phase
			raw, err := json.Marshal(s)
			require.NoError(t, err)

			// when
			result := h.OnGraceExpired(string(raw), tc.slot)

			// then
			assert.Equal(t, tc.wantFinish, result.Finished)
			assert.Equal(t, tc.wantResult, result.Result)
			if tc.wantWinner == nil {
				assert.Nil(t, result.WinnerSlot)
			} else {
				require.NotNil(t, result.WinnerSlot)
				assert.Equal(t, *tc.wantWinner, *result.WinnerSlot)
			}
		})
	}
}

func TestHandler_ComputeStats(t *testing.T) {
	// given a finished state with some reveals and flags
	h := NewHandler()
	stateJSON := startedState(t, h)
	s := unmarshalState(t, stateJSON)
	s.RevealedCount = [2]int{40, 25}
	s.Phase = phaseFinished
	s.Reason = resultComplete
	flagsP0 := []int{1, 2, 3}
	for _, idx := range flagsP0 {
		s.Flagged[0][idx] = true
	}
	raw, err := json.Marshal(s)
	require.NoError(t, err)

	createdAt := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	finishedAt := time.Now().UTC().Format(time.RFC3339)

	// when
	out, err := h.ComputeStats(string(raw), resultComplete, createdAt, finishedAt)
	require.NoError(t, err)

	// then
	stats, ok := out.(Stats)
	require.True(t, ok)
	assert.Equal(t, 40, stats.RevealedP0)
	assert.Equal(t, 25, stats.RevealedP1)
	assert.Equal(t, 3, stats.FlagsP0)
	assert.Equal(t, 0, stats.FlagsP1)
	assert.Equal(t, resultComplete, stats.Reason)
	assert.GreaterOrEqual(t, stats.DurationSeconds, 110)
}
