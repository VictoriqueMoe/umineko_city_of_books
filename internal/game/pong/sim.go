package pong

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	"umineko_city_of_books/internal/gameroom"

	"github.com/google/uuid"
)

type (
	sim struct {
		mu        sync.Mutex
		roomID    uuid.UUID
		rng       *rand.Rand
		st        State
		ballX     float64
		ballY     float64
		ballVX    float64
		ballVY    float64
		speed     float64
		paddleY   [2]float64
		paddleVY  [2]float64
		target    [2]float64
		ackSeq    [2]int
		connected [2]bool
		events    int
		phaseTick int
	}

	frame struct {
		RoomID    string  `json:"room_id"`
		T         int     `json:"t"`
		Phase     string  `json:"phase"`
		BallX     int     `json:"ball_x"`
		BallY     int     `json:"ball_y"`
		BallVX    int     `json:"ball_vx"`
		BallVY    int     `json:"ball_vy"`
		PaddleY   [2]int  `json:"paddle_y"`
		Scores    [2]int  `json:"scores"`
		Ack       [2]int  `json:"ack"`
		Connected [2]bool `json:"connected"`
		ServeInMS int     `json:"serve_in_ms"`
		Events    int     `json:"events"`
	}
)

func newSim(roomID uuid.UUID, stateJSON string, rng *rand.Rand) (*sim, error) {
	var st State
	if err := json.Unmarshal([]byte(stateJSON), &st); err != nil {
		return nil, fmt.Errorf("load state: %w", err)
	}

	s := &sim{
		roomID:    roomID,
		rng:       rng,
		st:        st,
		connected: [2]bool{true, true},
	}

	if st.Phase != phaseCountdown && st.Phase != phaseFinished {
		s.st.Phase = phaseServe
		s.st.BallX = boardWidth / 2
		s.st.BallY = boardHeight / 2
		s.st.BallVX = 0
		s.st.BallVY = 0
		s.st.BallSpeed = ballSpeedStart
		s.phaseTick = resumeCountdownMS * tickHz / 1000
	} else {
		s.phaseTick = st.PhaseRemainMS * tickHz / 1000
	}

	s.ballX = float64(s.st.BallX)
	s.ballY = float64(s.st.BallY)
	s.ballVX = float64(s.st.BallVX)
	s.ballVY = float64(s.st.BallVY)

	s.speed = float64(s.st.BallSpeed)
	if s.speed <= 0 {
		s.speed = ballSpeedStart
	}

	for i := range 2 {
		s.paddleY[i] = float64(s.st.PaddleY[i])
		s.target[i] = float64(s.st.PaddleY[i])
	}

	return s, nil
}

func (s *sim) Tick(step time.Duration) gameroom.TickResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.st.Phase == phaseFinished {
		return gameroom.TickResult{}
	}

	dt := step.Seconds()

	s.st.Ticks++
	if s.st.Ticks >= maxMatchTicks {
		return s.finishOnTimeLimit()
	}

	s.movePaddles(dt)

	if s.st.Phase == phaseCountdown || s.st.Phase == phaseServe {
		s.countDownToServe()
		return gameroom.TickResult{}
	}

	prevX, prevY := s.ballX, s.ballY
	s.ballX += s.ballVX * dt
	s.ballY += s.ballVY * dt

	s.sweepPaddles(prevX, prevY, dt)
	s.bounceWalls()

	return s.testGoal()
}

func (s *sim) movePaddles(dt float64) {
	for i := range 2 {
		next := clamp(s.target[i], paddleHeight/2.0, boardHeight-paddleHeight/2.0)
		delta := next - s.paddleY[i]
		s.paddleY[i] = next
		s.paddleVY[i] = clamp(delta/dt, -paddleMaxSpeed, paddleMaxSpeed)
	}
}

func (s *sim) countDownToServe() {
	s.phaseTick--
	if s.phaseTick > 0 {
		return
	}

	s.phaseTick = 0
	s.ballVX = float64(s.st.ServeVX)
	s.ballVY = float64(s.st.ServeVY)
	s.speed = ballSpeedStart
	s.st.Phase = phaseRally
	s.events |= eventServe
}

func (s *sim) sweepPaddles(prevX, prevY, dt float64) {
	if s.ballVX < 0 {
		s.sweepLeft(prevX, prevY, dt)
		return
	}

	if s.ballVX > 0 {
		s.sweepRight(prevX, prevY, dt)
	}
}

func (s *sim) sweepLeft(prevX, prevY, dt float64) {
	lead0 := prevX - ballRadius
	lead1 := s.ballX - ballRadius
	if lead0 <= faceX0 || lead1 > faceX0 {
		return
	}

	t := (lead0 - faceX0) / (lead0 - lead1)
	contactY := foldIntoCourt(prevY + (s.ballY-prevY)*t)
	if math.Abs(contactY-s.paddleY[0]) > paddleHeight/2.0+ballRadius {
		return
	}

	s.bounce(0, contactY)

	s.ballX = faceX0 + ballRadius
	s.ballY = contactY

	rem := 1 - t
	s.ballX += s.ballVX * dt * rem
	s.ballY += s.ballVY * dt * rem

	s.st.Hits[0]++
	s.st.RallyHits++
	s.events |= eventHitP0
}

func (s *sim) sweepRight(prevX, prevY, dt float64) {
	lead0 := prevX + ballRadius
	lead1 := s.ballX + ballRadius
	if lead0 >= faceX1 || lead1 < faceX1 {
		return
	}

	t := (faceX1 - lead0) / (lead1 - lead0)
	contactY := foldIntoCourt(prevY + (s.ballY-prevY)*t)
	if math.Abs(contactY-s.paddleY[1]) > paddleHeight/2.0+ballRadius {
		return
	}

	s.bounce(1, contactY)

	s.ballX = faceX1 - ballRadius
	s.ballY = contactY

	rem := 1 - t
	s.ballX += s.ballVX * dt * rem
	s.ballY += s.ballVY * dt * rem

	s.st.Hits[1]++
	s.st.RallyHits++
	s.events |= eventHitP1
}

func (s *sim) bounce(slot int, contactY float64) {
	offset := clamp((contactY-s.paddleY[slot])/(paddleHeight/2.0), -1, 1)
	s.speed = math.Min(s.speed*ballSpeedGain, ballSpeedMax)

	dir := 1.0
	if slot == 1 {
		dir = -1.0
	}

	angle := offset * maxBounceAngle
	vy := s.speed*math.Sin(angle) + s.paddleVY[slot]*paddleEnglish
	vx := s.speed * math.Cos(angle) * dir

	angle = math.Atan2(vy, math.Abs(vx))
	angle = clamp(angle, -maxBounceAngle, maxBounceAngle)

	s.ballVX = dir * s.speed * math.Cos(angle)
	s.ballVY = s.speed * math.Sin(angle)

	if int(s.speed) > s.st.TopSpeed {
		s.st.TopSpeed = int(s.speed)
	}
}

func (s *sim) bounceWalls() {
	if s.ballY < ballRadius {
		s.ballY = 2*ballRadius - s.ballY
		s.ballVY = math.Abs(s.ballVY)
		s.events |= eventWall
	}

	if s.ballY > boardHeight-ballRadius {
		s.ballY = 2*(boardHeight-ballRadius) - s.ballY
		s.ballVY = -math.Abs(s.ballVY)
		s.events |= eventWall
	}
}

func (s *sim) testGoal() gameroom.TickResult {
	if s.ballX+ballRadius < 0 {
		return s.scorePoint(1)
	}

	if s.ballX-ballRadius > boardWidth {
		return s.scorePoint(0)
	}

	return gameroom.TickResult{}
}

func (s *sim) scorePoint(winner int) gameroom.TickResult {
	s.st.Scores[winner]++
	if s.st.RallyHits > s.st.LongestRally[winner] {
		s.st.LongestRally[winner] = s.st.RallyHits
	}
	s.st.RallyHits = 0
	s.events |= eventScore

	s.st.ServeToSlot = 1 - winner
	s.st.ServeVX, s.st.ServeVY = serveVelocity(s.st.ServeToSlot, s.rng.IntN, s.rng.Float64)

	s.ballX = boardWidth / 2.0
	s.ballY = boardHeight / 2.0
	s.ballVX = 0
	s.ballVY = 0
	s.speed = ballSpeedStart
	s.st.Phase = phaseServe
	s.phaseTick = serveDelayMS * tickHz / 1000

	hi, lo := s.st.Scores[0], s.st.Scores[1]
	if lo > hi {
		hi, lo = lo, hi
	}

	if (hi >= pointsToWin && hi-lo >= winBy) || hi >= hardCapScore {
		return s.finish(winner, resultWin)
	}

	return gameroom.TickResult{Scored: true}
}

func (s *sim) finishOnTimeLimit() gameroom.TickResult {
	winner := 0

	switch {
	case s.st.Scores[1] > s.st.Scores[0]:
		winner = 1
	case s.st.Scores[1] == s.st.Scores[0] && s.st.Hits[1] > s.st.Hits[0]:
		winner = 1
	}

	return s.finish(winner, resultTimeLimit)
}

func (s *sim) finish(winnerSlot int, reason string) gameroom.TickResult {
	s.st.Phase = phaseFinished
	s.st.WinnerSlot = new(winnerSlot)
	s.st.Reason = reason
	s.st.FinishedAt = time.Now().UTC().Format(time.RFC3339)

	return gameroom.TickResult{
		Finished:   true,
		WinnerSlot: new(winnerSlot),
		Result:     reason,
	}
}

func (s *sim) ApplyInput(slot int, payload json.RawMessage) {
	if slot < 0 || slot > 1 {
		return
	}

	var in input
	if err := json.Unmarshal(payload, &in); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.target[slot] = clamp(float64(in.Y), paddleHeight/2.0, boardHeight-paddleHeight/2.0)
	s.ackSeq[slot] = in.Seq
}

func (s *sim) SetConnected(slot int, connected bool) {
	if slot < 0 || slot > 1 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.target[slot] = s.paddleY[slot]
	s.connected[slot] = connected
}

func (s *sim) Snapshot() any {
	s.mu.Lock()
	defer s.mu.Unlock()

	f := frame{
		RoomID:    s.roomID.String(),
		T:         s.st.Ticks * 1000 / tickHz,
		Phase:     s.st.Phase,
		BallX:     roundInt(s.ballX),
		BallY:     roundInt(s.ballY),
		BallVX:    roundInt(s.ballVX),
		BallVY:    roundInt(s.ballVY),
		PaddleY:   [2]int{roundInt(s.paddleY[0]), roundInt(s.paddleY[1])},
		Scores:    s.st.Scores,
		Ack:       s.ackSeq,
		Connected: s.connected,
		ServeInMS: s.phaseTick * 1000 / tickHz,
		Events:    s.events,
	}
	s.events = 0

	return f
}

func (s *sim) StateJSON() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.st
	st.BallX = roundInt(s.ballX)
	st.BallY = roundInt(s.ballY)
	st.BallVX = roundInt(s.ballVX)
	st.BallVY = roundInt(s.ballVY)
	st.BallSpeed = roundInt(s.speed)
	st.PaddleY = [2]int{roundInt(s.paddleY[0]), roundInt(s.paddleY[1])}
	st.PhaseRemainMS = s.phaseTick * 1000 / tickHz

	raw, err := json.Marshal(st)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

func foldIntoCourt(y float64) float64 {
	if y < ballRadius {
		return 2*ballRadius - y
	}

	if y > boardHeight-ballRadius {
		return 2*(boardHeight-ballRadius) - y
	}

	return y
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

func roundInt(v float64) int {
	return int(math.Round(v))
}
