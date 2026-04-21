package chess

import (
	"encoding/json"
	"testing"

	"umineko_city_of_books/internal/dto"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_InitialState(t *testing.T) {
	// given
	h := NewHandler()
	players := []dto.GameRoomPlayer{
		{UserID: uuid.New(), Slot: 0},
		{UserID: uuid.New(), Slot: 1},
	}

	// when
	stateJSON, firstTurn, err := h.InitialState(uuid.New(), players)

	// then
	require.NoError(t, err)
	assert.Equal(t, slotWhite, firstTurn)
	var s state
	require.NoError(t, json.Unmarshal([]byte(stateJSON), &s))
	assert.Equal(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", s.FEN)
}

func TestHandler_ValidateAction_LegalMove(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)
	action := json.RawMessage(`{"from":"e2","to":"e4"}`)

	// when
	result, err := h.ValidateAction(stateJSON, slotWhite, action)

	// then
	require.NoError(t, err)
	assert.False(t, result.Finished)
	require.NotNil(t, result.NextTurnSlot)
	assert.Equal(t, slotBlack, *result.NextTurnSlot)
}

func TestHandler_ValidateAction_WrongTurn(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)
	action := json.RawMessage(`{"from":"e7","to":"e5"}`)

	// when
	_, err = h.ValidateAction(stateJSON, slotBlack, action)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")
}

func TestHandler_ValidateAction_IllegalMove(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)
	action := json.RawMessage(`{"from":"e2","to":"e5"}`)

	// when
	_, err = h.ValidateAction(stateJSON, slotWhite, action)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "illegal move")
}

func TestHandler_ValidateAction_Checkmate(t *testing.T) {
	// given: fool's mate setup - black to deliver mate with Qxf2
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	moves := []struct {
		slot   int
		action string
	}{
		{slotWhite, `{"from":"f2","to":"f3"}`},
		{slotBlack, `{"from":"e7","to":"e5"}`},
		{slotWhite, `{"from":"g2","to":"g4"}`},
	}
	for _, m := range moves {
		res, err := h.ValidateAction(stateJSON, m.slot, json.RawMessage(m.action))
		require.NoError(t, err)
		stateJSON = res.NewStateJSON
	}

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"from":"d8","to":"h4"}`))

	// then
	require.NoError(t, err)
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotBlack, *res.WinnerSlot)
}
