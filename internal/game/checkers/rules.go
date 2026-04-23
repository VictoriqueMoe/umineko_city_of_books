package checkers

import (
	"errors"
	"fmt"
)

type board [boardSize][boardSize]byte

type outcomeResult struct {
	finished   bool
	winnerSlot *int
}

func isDarkSquare(row, col int) bool {
	return (row+col)%2 == 0
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
			case cellEmpty, cellRedMan, cellRedKing, cellBlackMan, cellBlackKing:
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

func parseSquare(s string) (int, int, error) {
	if len(s) != 2 {
		return 0, 0, fmt.Errorf("bad square %q", s)
	}
	col := int(s[0] - 'a')
	row := int(s[1] - '1')
	if col < 0 || col >= boardSize || row < 0 || row >= boardSize {
		return 0, 0, fmt.Errorf("square out of range: %q", s)
	}
	if !isDarkSquare(row, col) {
		return 0, 0, fmt.Errorf("light square not allowed: %q", s)
	}
	return row, col, nil
}

func pieceOwner(p byte) int {
	switch p {
	case cellRedMan, cellRedKing:
		return slotRed
	case cellBlackMan, cellBlackKing:
		return slotBlack
	}
	return -1
}

func isKing(p byte) bool {
	return p == cellRedKing || p == cellBlackKing
}

func moveDirs(piece byte) [][2]int {
	if isKing(piece) {
		return [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	}
	if piece == cellRedMan {
		return [][2]int{{1, -1}, {1, 1}}
	}
	if piece == cellBlackMan {
		return [][2]int{{-1, -1}, {-1, 1}}
	}
	return nil
}

func inBounds(r, c int) bool {
	return r >= 0 && r < boardSize && c >= 0 && c < boardSize
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func pieceHasJump(b board, r, c int) bool {
	piece := b[r][c]
	if piece == cellEmpty {
		return false
	}
	owner := pieceOwner(piece)
	for _, d := range moveDirs(piece) {
		midR, midC := r+d[0], c+d[1]
		landR, landC := r+2*d[0], c+2*d[1]
		if !inBounds(landR, landC) {
			continue
		}
		mid := b[midR][midC]
		if mid == cellEmpty {
			continue
		}
		if pieceOwner(mid) == owner {
			continue
		}
		if b[landR][landC] != cellEmpty {
			continue
		}
		return true
	}
	return false
}

func playerHasCapture(b board, slot int) bool {
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			piece := b[r][c]
			if piece == cellEmpty || pieceOwner(piece) != slot {
				continue
			}
			if pieceHasJump(b, r, c) {
				return true
			}
		}
	}
	return false
}

func playerHasAnyMove(b board, slot int) bool {
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			piece := b[r][c]
			if piece == cellEmpty || pieceOwner(piece) != slot {
				continue
			}
			if pieceHasJump(b, r, c) {
				return true
			}
			for _, d := range moveDirs(piece) {
				nr, nc := r+d[0], c+d[1]
				if !inBounds(nr, nc) {
					continue
				}
				if b[nr][nc] == cellEmpty {
					return true
				}
			}
		}
	}
	return false
}

func shouldCrown(piece byte, row int) bool {
	if piece == cellRedMan && row == boardSize-1 {
		return true
	}
	if piece == cellBlackMan && row == 0 {
		return true
	}
	return false
}

func promoted(piece byte) byte {
	switch piece {
	case cellRedMan:
		return cellRedKing
	case cellBlackMan:
		return cellBlackKing
	}
	return piece
}

func applySimpleMove(b *board, fromR, fromC, toR, toC, slot int) (bool, error) {
	piece := b[fromR][fromC]
	if piece == cellEmpty || pieceOwner(piece) != slot {
		return false, errors.New("invalid piece for simple move")
	}
	if !inBounds(toR, toC) {
		return false, errors.New("destination out of bounds")
	}
	if b[toR][toC] != cellEmpty {
		return false, errors.New("destination not empty")
	}
	dr, dc := toR-fromR, toC-fromC
	if absInt(dr) != 1 || absInt(dc) != 1 {
		return false, errors.New("simple move must be one diagonal step")
	}
	allowed := false
	for _, d := range moveDirs(piece) {
		if d[0] == dr && d[1] == dc {
			allowed = true
			break
		}
	}
	if !allowed {
		return false, errors.New("piece cannot move in that direction")
	}
	b[fromR][fromC] = cellEmpty
	if shouldCrown(piece, toR) {
		b[toR][toC] = promoted(piece)
		return true, nil
	}
	b[toR][toC] = piece
	return false, nil
}

func applyJumpSequence(b *board, fromR, fromC int, path [][2]int, slot int) (int, bool, error) {
	piece := b[fromR][fromC]
	if piece == cellEmpty || pieceOwner(piece) != slot {
		return 0, false, errors.New("invalid piece for jump")
	}
	curR, curC := fromR, fromC
	curPiece := piece
	b[fromR][fromC] = cellEmpty
	captured := 0
	crownedThisTurn := false

	for i, step := range path {
		nr, nc := step[0], step[1]
		dr, dc := nr-curR, nc-curC
		if absInt(dr) != 2 || absInt(dc) != 2 {
			return 0, false, fmt.Errorf("jump step %d must be two diagonal squares", i+1)
		}
		dirOK := false
		for _, d := range moveDirs(curPiece) {
			if d[0]*2 == dr && d[1]*2 == dc {
				dirOK = true
				break
			}
		}
		if !dirOK {
			return 0, false, fmt.Errorf("piece cannot jump in that direction at step %d", i+1)
		}
		midR, midC := curR+dr/2, curC+dc/2
		if !inBounds(nr, nc) {
			return 0, false, fmt.Errorf("jump %d out of bounds", i+1)
		}
		if b[nr][nc] != cellEmpty {
			return 0, false, fmt.Errorf("landing square not empty at step %d", i+1)
		}
		mid := b[midR][midC]
		if mid == cellEmpty || pieceOwner(mid) == slot {
			return 0, false, fmt.Errorf("no opponent piece to jump at step %d", i+1)
		}
		b[midR][midC] = cellEmpty
		captured++
		curR, curC = nr, nc
		if shouldCrown(curPiece, curR) {
			curPiece = promoted(curPiece)
			crownedThisTurn = true
			b[curR][curC] = curPiece
			if i != len(path)-1 {
				return 0, false, errors.New("cannot continue jumping after crowning")
			}
		} else {
			b[curR][curC] = curPiece
		}
	}

	if !crownedThisTurn && pieceHasJump(*b, curR, curC) {
		return 0, false, errors.New("must continue jumping")
	}

	return captured, crownedThisTurn, nil
}

func evaluateOutcome(b board, nextSlot, movesSinceCapture int) (outcomeResult, string) {
	red, black := countPiecesBoard(b)
	if red == 0 {
		winner := slotBlack
		return outcomeResult{finished: true, winnerSlot: &winner}, "no_pieces"
	}
	if black == 0 {
		winner := slotRed
		return outcomeResult{finished: true, winnerSlot: &winner}, "no_pieces"
	}
	if !playerHasAnyMove(b, nextSlot) {
		winner := 1 - nextSlot
		return outcomeResult{finished: true, winnerSlot: &winner}, "no_moves"
	}
	if movesSinceCapture >= drawMoveLimit {
		return outcomeResult{finished: true}, "forty_move_rule"
	}
	return outcomeResult{}, ""
}

func countPiecesBoard(b board) (int, int) {
	red, black := 0, 0
	for r := 0; r < boardSize; r++ {
		for c := 0; c < boardSize; c++ {
			switch b[r][c] {
			case cellRedMan, cellRedKing:
				red++
			case cellBlackMan, cellBlackKing:
				black++
			}
		}
	}
	return red, black
}

func countPieces(s string) (int, int) {
	red, black := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case cellRedMan, cellRedKing:
			red++
		case cellBlackMan, cellBlackKing:
			black++
		}
	}
	return red, black
}
