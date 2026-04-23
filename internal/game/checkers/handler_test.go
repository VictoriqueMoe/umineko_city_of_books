package checkers

import (
	"encoding/json"
	"strings"
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
	assert.Equal(t, slotRed, firstTurn)
	var s state
	require.NoError(t, json.Unmarshal([]byte(stateJSON), &s))
	assert.Equal(t, 64, len(s.Board))
	red, black := countPieces(s.Board)
	assert.Equal(t, 12, red)
	assert.Equal(t, 12, black)
	assert.Equal(t, slotRed, s.Turn)
}

func TestHandler_ValidateAction_LegalSimpleMove(t *testing.T) {
	// given: c3 is a red man at start; diagonal forward to b4 or d4 is empty
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)
	action := json.RawMessage(`{"from":"c3","path":["b4"]}`)

	// when
	result, err := h.ValidateAction(stateJSON, slotRed, action)

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
	action := json.RawMessage(`{"from":"b6","path":["a5"]}`)

	// when
	_, err = h.ValidateAction(stateJSON, slotBlack, action)

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")
}

func TestHandler_ValidateAction_IllegalBackwardMan(t *testing.T) {
	// given: put a lone red man at c3 with empty surroundings
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 7, 0, cellBlackMan) // a8 so black still has a piece
	stateJSON := stateWith(b, slotRed)

	// when: try c3 -> b2 (backwards for a red man)
	_, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c3","path":["b2"]}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot move in that direction")
}

func TestHandler_ValidateAction_MandatoryCapture(t *testing.T) {
	// given: red has a jump available on c3 -> e5, plus an unrelated simple move from a1
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 3, 3, cellBlackMan) // d4
	b = setCell(b, 0, 0, cellRedMan)   // a1 - unrelated red with simple move to b2
	b = setCell(b, 7, 0, cellBlackMan) // a8 so black survives
	stateJSON := stateWith(b, slotRed)

	// when: try the simple move while a capture is available
	_, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"a1","path":["b2"]}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "capture is mandatory")
}

func TestHandler_ValidateAction_SingleJump(t *testing.T) {
	// given: red can jump black
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 3, 3, cellBlackMan) // d4
	b = setCell(b, 7, 0, cellBlackMan) // a8 so black survives
	stateJSON := stateWith(b, slotRed)

	// when: jump c3 over d4 to e5
	res, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c3","path":["e5"]}`))

	// then
	require.NoError(t, err)
	assert.False(t, res.Finished)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, 1, ns.RedCaptures)
	// d4 captured
	assert.Equal(t, byte(cellEmpty), ns.Board[3*boardSize+3])
	// e5 holds red man
	assert.Equal(t, byte(cellRedMan), ns.Board[4*boardSize+4])
}

func TestHandler_ValidateAction_MultiJump(t *testing.T) {
	// given: red on c3 can double-jump c3 -> e5 -> g7 over d4 and f6
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 3, 3, cellBlackMan) // d4
	b = setCell(b, 5, 5, cellBlackMan) // f6
	b = setCell(b, 7, 1, cellBlackMan) // b8 so black survives
	stateJSON := stateWith(b, slotRed)

	// when
	res, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c3","path":["e5","g7"]}`))

	// then
	require.NoError(t, err)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, 2, ns.RedCaptures)
}

func TestHandler_ValidateAction_MissingContinuation(t *testing.T) {
	// given: red on c3 must continue jumping past e5 (f6 capture is mandatory)
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 3, 3, cellBlackMan) // d4
	b = setCell(b, 5, 5, cellBlackMan) // f6
	b = setCell(b, 7, 1, cellBlackMan) // b8
	stateJSON := stateWith(b, slotRed)

	// when: only first jump submitted
	_, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c3","path":["e5"]}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must continue jumping")
}

func TestHandler_ValidateAction_Crowning(t *testing.T) {
	// given: red man on rank 7 moves forward to rank 8 -> crowned
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 6, 2, cellRedMan)   // c7
	b = setCell(b, 0, 1, cellBlackMan) // b1 so black survives
	stateJSON := stateWith(b, slotRed)

	// when
	res, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c7","path":["b8"]}`))

	// then
	require.NoError(t, err)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, byte(cellRedKing), ns.Board[7*boardSize+1])
	assert.Equal(t, 1, ns.RedCrownings)
}

func TestHandler_ValidateAction_WinByNoPieces(t *testing.T) {
	// given: red jumps to capture the last black piece
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 2, cellRedMan)   // c3
	b = setCell(b, 3, 3, cellBlackMan) // d4 (only black piece)
	stateJSON := stateWith(b, slotRed)

	// when
	res, err := h.ValidateAction(stateJSON, slotRed, json.RawMessage(`{"from":"c3","path":["e5"]}`))

	// then
	require.NoError(t, err)
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotRed, *res.WinnerSlot)
	assert.Equal(t, "no_pieces", res.Result)
}

func TestHandler_OnGraceExpired(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: red disconnects
	res := h.OnGraceExpired(stateJSON, slotRed)

	// then
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotBlack, *res.WinnerSlot)
	assert.Equal(t, "abandoned", res.Result)
}

func TestHandler_ComputeStats(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	stats, err := h.ComputeStats(stateJSON, "", "2024-01-01T00:00:00Z", "2024-01-01T00:01:00Z")

	// then
	require.NoError(t, err)
	s := stats.(Stats)
	assert.Equal(t, 12, s.RedPiecesLeft)
	assert.Equal(t, 12, s.BlackPiecesLeft)
	assert.Equal(t, 60, s.DurationSeconds)
}

func emptyBoardString() string {
	return strings.Repeat(string(cellEmpty), boardSize*boardSize)
}

func setCell(b string, row, col int, piece byte) string {
	bs := []byte(b)
	bs[row*boardSize+col] = piece
	return string(bs)
}

func stateWith(boardStr string, turn int) string {
	s := state{Board: boardStr, Turn: turn}
	raw, _ := json.Marshal(s)
	return string(raw)
}
