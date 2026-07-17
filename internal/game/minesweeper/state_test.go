package minesweeper

import (
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRngSeeded(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed, seed^0xdeadbeef))
}

func TestState_PlaceMines_HonoursSafeZones(t *testing.T) {
	cases := []struct {
		name      string
		width     int
		height    int
		mineCount int
		safe      [][2]int
	}{
		{name: "single safe zone in corner", width: 16, height: 16, mineCount: 40, safe: [][2]int{{0, 0}}},
		{name: "two safe zones", width: 16, height: 16, mineCount: 40, safe: [][2]int{{3, 4}, {10, 10}}},
		{name: "centre safe zone", width: 10, height: 10, mineCount: 15, safe: [][2]int{{5, 5}}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// given a fresh state
			s := newInitialState(tc.width, tc.height, tc.mineCount)

			// when placing mines avoiding the safe zones
			s.placeMines(newRngSeeded(42), tc.safe)

			// then no mine sits in the 3x3 safe zones and total mines match
			placed := 0
			for i := range s.Mines {
				if s.Mines[i] {
					placed++
				}
			}
			assert.Equal(t, tc.mineCount, placed)
			for _, sp := range tc.safe {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						nx, ny := sp[0]+dx, sp[1]+dy
						if !s.inBounds(nx, ny) {
							continue
						}
						assert.Falsef(t, s.Mines[s.idx(nx, ny)], "mine placed in safe zone at (%d,%d)", nx, ny)
					}
				}
			}
		})
	}
}

func TestState_FloodFill_RevealsZeroRegion(t *testing.T) {
	// given a 5x5 board with a single mine at (4,4) so most of the board is zero-value
	s := newInitialState(5, 5, 1)
	s.Mines = make([]bool, 25)
	s.Mines[s.idx(4, 4)] = true
	s.MinesPlaced = true

	// when slot 0 reveals (0,0)
	revealed := s.floodFill(0, 0, 0)

	// then the flood fills out across the safe region and stops at non-zero perimeter
	assert.Greater(t, revealed, 1)
	assert.True(t, s.Revealed[0][s.idx(0, 0)])
	assert.False(t, s.Revealed[0][s.idx(4, 4)])
}

func TestState_AdjacencyAt_CountsNeighbours(t *testing.T) {
	// given a 3x3 board with mines at (0,0) and (2,2)
	s := newInitialState(3, 3, 2)
	s.Mines = make([]bool, 9)
	s.Mines[s.idx(0, 0)] = true
	s.Mines[s.idx(2, 2)] = true

	// when computing adjacency for the centre cell
	count := s.adjacencyAt(1, 1)

	// then it counts both neighbours
	assert.Equal(t, int8(2), count)
}

func TestState_TotalSafeCells(t *testing.T) {
	// given
	s := newInitialState(16, 16, 40)

	// when / then
	require.Equal(t, 16*16-40, s.totalSafeCells())
}
