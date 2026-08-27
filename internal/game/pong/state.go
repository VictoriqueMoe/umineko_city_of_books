package pong

import (
	"math"
	"time"
)

type (
	State struct {
		Phase         string `json:"phase"`
		Width         int    `json:"width"`
		Height        int    `json:"height"`
		BallRadius    int    `json:"ball_radius"`
		PaddleWidth   int    `json:"paddle_width"`
		PaddleHeight  int    `json:"paddle_height"`
		PaddleInset   int    `json:"paddle_inset"`
		PointsToWin   int    `json:"points_to_win"`
		BallX         int    `json:"ball_x"`
		BallY         int    `json:"ball_y"`
		BallVX        int    `json:"ball_vx"`
		BallVY        int    `json:"ball_vy"`
		BallSpeed     int    `json:"ball_speed"`
		PaddleY       [2]int `json:"paddle_y"`
		Scores        [2]int `json:"scores"`
		Hits          [2]int `json:"hits"`
		LongestRally  [2]int `json:"longest_rally"`
		RallyHits     int    `json:"rally_hits"`
		TopSpeed      int    `json:"top_speed"`
		ServeToSlot   int    `json:"serve_to_slot"`
		ServeVX       int    `json:"serve_vx"`
		ServeVY       int    `json:"serve_vy"`
		PhaseRemainMS int    `json:"phase_remain_ms"`
		Ticks         int    `json:"ticks"`
		StartedAt     string `json:"started_at,omitempty"`
		FinishedAt    string `json:"finished_at,omitempty"`
		WinnerSlot    *int   `json:"winner_slot,omitempty"`
		Reason        string `json:"reason,omitempty"`
	}

	input struct {
		Y   int `json:"y"`
		Seq int `json:"seq"`
	}

	Stats struct {
		PointsP0        int    `json:"points_p0"`
		PointsP1        int    `json:"points_p1"`
		HitsP0          int    `json:"hits_p0"`
		HitsP1          int    `json:"hits_p1"`
		LongestRallyP0  int    `json:"longest_rally_p0"`
		LongestRallyP1  int    `json:"longest_rally_p1"`
		TopSpeed        int    `json:"top_speed"`
		ResultReason    string `json:"result_reason"`
		DurationSeconds int    `json:"duration_seconds"`
	}
)

func newInitialState(rngIntN func(int) int, rngFloat func() float64) State {
	s := State{
		Phase:         phaseCountdown,
		Width:         boardWidth,
		Height:        boardHeight,
		BallRadius:    ballRadius,
		PaddleWidth:   paddleWidth,
		PaddleHeight:  paddleHeight,
		PaddleInset:   paddleInset,
		PointsToWin:   pointsToWin,
		BallX:         boardWidth / 2,
		BallY:         boardHeight / 2,
		BallSpeed:     ballSpeedStart,
		PaddleY:       [2]int{boardHeight / 2, boardHeight / 2},
		PhaseRemainMS: countdownMS,
		StartedAt:     time.Now().UTC().Format(time.RFC3339),
		ServeToSlot:   rngIntN(2),
	}
	s.ServeVX, s.ServeVY = serveVelocity(s.ServeToSlot, rngIntN, rngFloat)

	return s
}

func serveVelocity(serveToSlot int, rngIntN func(int) int, rngFloat func() float64) (int, int) {
	angle := serveAngleMin + rngFloat()*(serveAngleMax-serveAngleMin)

	sign := 1.0
	if rngIntN(2) == 0 {
		sign = -1.0
	}

	dir := 1.0
	if serveToSlot == 0 {
		dir = -1.0
	}

	vx := ballSpeedStart * math.Cos(angle) * dir
	vy := ballSpeedStart * math.Sin(angle) * sign

	return int(math.Round(vx)), int(math.Round(vy))
}
