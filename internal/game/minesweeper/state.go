package minesweeper

import (
	"math/rand/v2"
)

type (
	State struct {
		Phase         string     `json:"phase"`
		Width         int        `json:"width"`
		Height        int        `json:"height"`
		MineCount     int        `json:"mine_count"`
		Characters    [2]string  `json:"characters"`
		StartedAt     string     `json:"started_at,omitempty"`
		FinishedAt    string     `json:"finished_at,omitempty"`
		Revealed      [2][]bool  `json:"revealed"`
		Flagged       [2][]bool  `json:"flagged"`
		RevealedCount [2]int     `json:"revealed_count"`
		Values        [2][]int8  `json:"values"`
		Mines         []bool     `json:"mines,omitempty"`
		MinesPlaced   bool       `json:"mines_placed"`
		PendingClicks [2]*[2]int `json:"pending_clicks"`
		WinnerSlot    *int       `json:"winner_slot,omitempty"`
		Reason        string     `json:"reason,omitempty"`
		HitMineX      *int       `json:"hit_mine_x,omitempty"`
		HitMineY      *int       `json:"hit_mine_y,omitempty"`
	}
)

func newInitialState(width, height, mineCount int) *State {
	total := width * height
	s := &State{
		Phase:     phaseCharSelect,
		Width:     width,
		Height:    height,
		MineCount: mineCount,
	}
	for i := range 2 {
		s.Revealed[i] = make([]bool, total)
		s.Flagged[i] = make([]bool, total)
		s.Values[i] = make([]int8, total)
	}
	return s
}

func (s *State) idx(x, y int) int {
	return y*s.Width + x
}

func (s *State) inBounds(x, y int) bool {
	return x >= 0 && x < s.Width && y >= 0 && y < s.Height
}

func (s *State) placeMines(rng *rand.Rand, safeZones [][2]int) {
	excluded := make(map[int]bool)
	for i := range safeZones {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx := safeZones[i][0] + dx
				ny := safeZones[i][1] + dy
				if s.inBounds(nx, ny) {
					excluded[s.idx(nx, ny)] = true
				}
			}
		}
	}
	total := s.Width * s.Height
	s.Mines = make([]bool, total)
	candidates := make([]int, 0, total)
	for i := range total {
		if !excluded[i] {
			candidates = append(candidates, i)
		}
	}
	rng.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	count := s.MineCount
	if count > len(candidates) {
		count = len(candidates)
	}
	for i := range count {
		s.Mines[candidates[i]] = true
	}
	s.MinesPlaced = true
}

func (s *State) adjacencyAt(x, y int) int8 {
	var count int8
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if s.inBounds(nx, ny) && s.Mines[s.idx(nx, ny)] {
				count++
			}
		}
	}
	return count
}

func (s *State) floodFill(slot, x, y int) int {
	revealed := 0
	stack := [][2]int{{x, y}}
	for len(stack) > 0 {
		pos := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !s.inBounds(pos[0], pos[1]) {
			continue
		}
		i := s.idx(pos[0], pos[1])
		if s.Revealed[slot][i] {
			continue
		}
		if s.Mines[i] {
			continue
		}
		s.Revealed[slot][i] = true
		revealed++
		val := s.adjacencyAt(pos[0], pos[1])
		s.Values[slot][i] = val
		if val == 0 {
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					stack = append(stack, [2]int{pos[0] + dx, pos[1] + dy})
				}
			}
		}
	}
	return revealed
}

func (s *State) totalSafeCells() int {
	return s.Width*s.Height - s.MineCount
}
