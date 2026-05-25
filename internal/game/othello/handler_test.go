package othello

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
	assert.Equal(t, slotBlack, firstTurn)
	var s state
	require.NoError(t, json.Unmarshal([]byte(stateJSON), &s))
	assert.Equal(t, 64, len(s.Board))
	assert.Equal(t, slotBlack, s.Turn)
	black, white := countDiscs(s.Board)
	assert.Equal(t, 2, black)
	assert.Equal(t, 2, white)
	assert.Equal(t, cellWhite, s.Board[3*boardSize+3])
	assert.Equal(t, cellBlack, s.Board[3*boardSize+4])
	assert.Equal(t, cellBlack, s.Board[4*boardSize+3])
	assert.Equal(t, cellWhite, s.Board[4*boardSize+4])
}

func TestHandler_ValidateAction_LegalOpening(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: black plays d3 to flip d4
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"d3"}`))

	// then
	require.NoError(t, err)
	assert.False(t, res.Finished)
	require.NotNil(t, res.NextTurnSlot)
	assert.Equal(t, slotWhite, *res.NextTurnSlot)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, cellBlack, ns.Board[2*boardSize+3])
	assert.Equal(t, cellBlack, ns.Board[3*boardSize+3])
	assert.Equal(t, 1, ns.BlackMoves)
	assert.Equal(t, 1, ns.BlackFlips)
	require.NotNil(t, ns.LastMove)
	assert.Equal(t, "d3", ns.LastMove.Square)
	assert.Equal(t, slotBlack, ns.LastMove.Slot)
	assert.Equal(t, []string{"d4"}, ns.LastMove.Flipped)
}

func TestHandler_ValidateAction_WrongTurn(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: white tries to move when it is black's turn
	_, err = h.ValidateAction(stateJSON, slotWhite, json.RawMessage(`{"square":"e6"}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not your turn")
}

func TestHandler_ValidateAction_OccupiedSquare(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: try to play on the occupied centre square
	_, err = h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"d4"}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "occupied")
}

func TestHandler_ValidateAction_NoFlanking(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: a1 flanks nothing from the opening position
	_, err = h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"a1"}`))

	// then
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must flip")
}

func TestHandler_ValidateAction_MultiDirectionFlip(t *testing.T) {
	// given: black places at d3 to flip both c3 (west) and d4 (south) in two directions
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 2, 1, cellBlack) // b3 - west bookend
	b = setCell(b, 2, 2, cellWhite) // c3
	b = setCell(b, 3, 3, cellWhite) // d4
	b = setCell(b, 4, 3, cellBlack) // d5 - south bookend
	stateJSON := stateWith(b, slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"d3"}`))

	// then
	require.NoError(t, err)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, cellBlack, ns.Board[2*boardSize+2]) // c3 flipped
	assert.Equal(t, cellBlack, ns.Board[3*boardSize+3]) // d4 flipped
	assert.Equal(t, 2, ns.BlackFlips)
	require.NotNil(t, ns.LastMove)
	assert.ElementsMatch(t, []string{"c3", "d4"}, ns.LastMove.Flipped)
}

func TestHandler_ValidateAction_ServerAutoPass(t *testing.T) {
	// given: after black plays h3, white has no legal moves but black still does (e5 flips f6 via g7 bookend)
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 0, 7, cellBlack) // h1
	b = setCell(b, 1, 7, cellWhite) // h2
	b = setCell(b, 3, 7, cellBlack) // h4
	b = setCell(b, 5, 5, cellWhite) // f6
	b = setCell(b, 6, 6, cellBlack) // g7 - bookend so black can flank f6 from e5
	b = setCell(b, 7, 7, cellBlack) // h8 - blocks white from flanking g7
	stateJSON := stateWith(b, slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"h3"}`))

	// then
	require.NoError(t, err)
	assert.False(t, res.Finished)
	require.NotNil(t, res.NextTurnSlot)
	assert.Equal(t, slotBlack, *res.NextTurnSlot)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	assert.Equal(t, slotBlack, ns.Turn)
	assert.Equal(t, 1, ns.WhitePasses)
	assert.Equal(t, 0, ns.BlackPasses)
}

func TestHandler_ValidateAction_GameEndsOnDoubleNoMoves(t *testing.T) {
	// given: same setup as auto-pass minus the g7 bookend, so black is also stuck after h3
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 0, 7, cellBlack) // h1
	b = setCell(b, 1, 7, cellWhite) // h2
	b = setCell(b, 3, 7, cellBlack) // h4
	b = setCell(b, 5, 5, cellWhite) // f6 - unflankable for either side
	stateJSON := stateWith(b, slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"h3"}`))

	// then
	require.NoError(t, err)
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotBlack, *res.WinnerSlot)
	assert.Equal(t, "no_moves", res.Result)
}

func TestHandler_ValidateAction_GameEndsBoardFull(t *testing.T) {
	// given: 63 discs with h8 as the only empty cell; black plays h8 and the board fills
	h := NewHandler()
	b := []byte(strings.Repeat(string(cellWhite), boardSize*boardSize))
	b[0*boardSize+0] = cellBlack // a1 - bookend for the NW anti-diagonal
	b[0*boardSize+7] = cellBlack // h1 - bookend for the column h line
	b[7*boardSize+7] = cellEmpty // h8 - the play spot
	stateJSON := stateWith(string(b), slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"h8"}`))

	// then
	require.NoError(t, err)
	assert.True(t, res.Finished)
	assert.Equal(t, "most_discs", res.Result)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotWhite, *res.WinnerSlot)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	black, white := countDiscs(ns.Board)
	assert.Equal(t, 64, black+white)
	assert.Greater(t, white, black)
}

func TestHandler_ValidateAction_DrawOnEqualDiscs(t *testing.T) {
	// given: a near-full board where black's only legal play (a1 flips a2 once) leaves 32-32
	h := NewHandler()
	rows := []string{
		".WWWWWWW",
		"WBWWWWWW",
		"BWWWWWWW",
		"BBBBBBBB",
		"BBBBBBBB",
		"BBBBBBBB",
		"BBBBWWWW",
		"WWWWWWWW",
	}
	stateJSON := stateWith(strings.Join(rows, ""), slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"a1"}`))

	// then
	require.NoError(t, err)
	assert.True(t, res.Finished)
	assert.Nil(t, res.WinnerSlot)
	assert.Equal(t, "draw", res.Result)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	black, white := countDiscs(ns.Board)
	assert.Equal(t, 32, black)
	assert.Equal(t, 32, white)
}

func TestHandler_ValidateAction_CornerCapture(t *testing.T) {
	// given: black plays the a1 corner, flipping b1 with the c1 bookend
	h := NewHandler()
	b := emptyBoardString()
	b = setCell(b, 0, 1, cellWhite) // b1
	b = setCell(b, 0, 2, cellBlack) // c1
	stateJSON := stateWith(b, slotBlack)

	// when
	res, err := h.ValidateAction(stateJSON, slotBlack, json.RawMessage(`{"square":"a1"}`))

	// then
	require.NoError(t, err)
	var ns state
	require.NoError(t, json.Unmarshal([]byte(res.NewStateJSON), &ns))
	boardArr, perr := parseBoard(ns.Board)
	require.NoError(t, perr)
	black, white := cornerCounts(boardArr)
	assert.Equal(t, 1, black)
	assert.Equal(t, 0, white)
}

func TestHandler_OnGraceExpired(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when: black disconnects
	res := h.OnGraceExpired(stateJSON, slotBlack)

	// then
	assert.True(t, res.Finished)
	require.NotNil(t, res.WinnerSlot)
	assert.Equal(t, slotWhite, *res.WinnerSlot)
	assert.Equal(t, "abandoned", res.Result)
}

func TestHandler_ComputeStats(t *testing.T) {
	// given
	h := NewHandler()
	stateJSON, _, err := h.InitialState(uuid.New(), nil)
	require.NoError(t, err)

	// when
	stats, err := h.ComputeStats(stateJSON, "most_discs", "2024-01-01T00:00:00Z", "2024-01-01T00:01:00Z")

	// then
	require.NoError(t, err)
	s := stats.(Stats)
	assert.Equal(t, 2, s.BlackDiscs)
	assert.Equal(t, 2, s.WhiteDiscs)
	assert.Equal(t, 0, s.BlackCorners)
	assert.Equal(t, 0, s.WhiteCorners)
	assert.Equal(t, 0, s.TotalMoves)
	assert.Equal(t, 60, s.DurationSeconds)
	assert.Equal(t, "most_discs", s.ResultReason)
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
