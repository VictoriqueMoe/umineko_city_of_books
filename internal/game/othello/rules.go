package othello

import (
	"errors"
	"fmt"
)

type (
	board [boardSize][boardSize]byte

	coord struct {
		row int
		col int
	}

	outcomeResult struct {
		finished   bool
		winnerSlot *int
	}
)

var directions = [8][2]int{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1}, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

func parseBoard(s string) (board, error) {
	var b board
	if len(s) != boardSize*boardSize {
		return b, fmt.Errorf("invalid board length: %d", len(s))
	}
	for row := 0; row < boardSize; row++ {
		for col := 0; col < boardSize; col++ {
			c := s[row*boardSize+col]
			switch c {
			case cellEmpty, cellBlack, cellWhite:
				b[row][col] = c
			default:
				return b, fmt.Errorf("invalid board cell: %c", c)
			}
		}
	}
	return b, nil
}

func boardString(b board) string {
	out := make([]byte, 0, boardSize*boardSize)
	for row := 0; row < boardSize; row++ {
		for col := 0; col < boardSize; col++ {
			out = append(out, b[row][col])
		}
	}
	return string(out)
}

func formatSquare(row, col int) string {
	return fmt.Sprintf("%c%c", byte('a'+col), byte('1'+row))
}

func parseSquare(s string) (int, int, error) {
	if len(s) != 2 {
		return 0, 0, fmt.Errorf("bad square %q", s)
	}
	col := int(s[0] - 'a')
	row := int(s[1] - '1')
	if col < 0 || col >= boardSize || row < 0 || row >= boardSize {
		return 0, 0, fmt.Errorf("square out of range: %q", s)
	}
	return row, col, nil
}

func inBounds(r, c int) bool {
	return r >= 0 && r < boardSize && c >= 0 && c < boardSize
}

func discFor(slot int) byte {
	if slot == slotBlack {
		return cellBlack
	}
	return cellWhite
}

func slotForCell(c byte) int {
	switch c {
	case cellBlack:
		return slotBlack
	case cellWhite:
		return slotWhite
	}
	return -1
}

func flipsForPlacement(b board, row, col, slot int) []coord {
	if !inBounds(row, col) || b[row][col] != cellEmpty {
		return nil
	}
	own := discFor(slot)
	var opp byte
	if own == cellBlack {
		opp = cellWhite
	} else {
		opp = cellBlack
	}

	var flipped []coord
	for _, d := range directions {
		var line []coord
		r, c := row+d[0], col+d[1]
		for inBounds(r, c) && b[r][c] == opp {
			line = append(line, coord{r, c})
			r += d[0]
			c += d[1]
		}
		if len(line) == 0 || !inBounds(r, c) || b[r][c] != own {
			continue
		}
		flipped = append(flipped, line...)
	}
	return flipped
}

func playerHasAnyLegalMove(b board, slot int) bool {
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			if b[r][c] != cellEmpty {
				continue
			}
			if len(flipsForPlacement(b, r, c, slot)) > 0 {
				return true
			}
		}
	}
	return false
}

func applyPlacement(b *board, row, col, slot int) ([]coord, error) {
	flipped := flipsForPlacement(*b, row, col, slot)
	if len(flipped) == 0 {
		return nil, errors.New("placement must flip at least one disc")
	}
	disc := discFor(slot)
	b[row][col] = disc
	for _, f := range flipped {
		b[f.row][f.col] = disc
	}
	return flipped, nil
}

func countDiscsBoard(b board) (int, int) {
	black, white := 0, 0
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			switch b[r][c] {
			case cellBlack:
				black++
			case cellWhite:
				white++
			}
		}
	}
	return black, white
}

func countDiscs(s string) (int, int) {
	black, white := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case cellBlack:
			black++
		case cellWhite:
			white++
		}
	}
	return black, white
}

func cornerCounts(b board) (int, int) {
	black, white := 0, 0
	corners := [4][2]int{{0, 0}, {0, boardSize - 1}, {boardSize - 1, 0}, {boardSize - 1, boardSize - 1}}
	for _, c := range corners {
		switch b[c[0]][c[1]] {
		case cellBlack:
			black++
		case cellWhite:
			white++
		}
	}
	return black, white
}

func evaluateOutcome(b board) (outcomeResult, string) {
	black, white := countDiscsBoard(b)
	boardFull := (black + white) == boardSize*boardSize
	blackHasMove := !boardFull && playerHasAnyLegalMove(b, slotBlack)
	whiteHasMove := !boardFull && playerHasAnyLegalMove(b, slotWhite)

	if !boardFull && (blackHasMove || whiteHasMove) {
		return outcomeResult{}, ""
	}

	reason := "most_discs"
	if !boardFull {
		reason = "no_moves"
	}
	if black == white {
		return outcomeResult{finished: true}, "draw"
	}
	if black > white {
		w := slotBlack
		return outcomeResult{finished: true, winnerSlot: &w}, reason
	}
	w := slotWhite
	return outcomeResult{finished: true, winnerSlot: &w}, reason
}
