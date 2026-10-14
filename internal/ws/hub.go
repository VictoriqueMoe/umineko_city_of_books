package ws

import (
	"context"
	"encoding/json"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"umineko_city_of_books/internal/coalesce"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	sendBufferSize = 64
	writeTimeout   = 10 * time.Second
	pingInterval   = 30 * time.Second

	maxWatchedPosts        = 200
	maxBatchItems          = 64
	maxBatchBytes          = 64 * 1024
	defaultSidebarInterval = time.Second
)

type (
	Message struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}

	Client struct {
		UserID       uuid.UUID
		Conn         *websocket.Conn
		send         chan []byte
		closeCh      chan any
		closeOnce    sync.Once
		pingAt       atomic.Int64
		rttPeak      atomic.Int64
		batch        bool
		watchedPosts map[uuid.UUID]struct{}
	}

	viewerInfo struct {
		tabs  int
		state string
	}

	Hub struct {
		name         string
		tracer       trace.Tracer
		clients      map[uuid.UUID][]*Client
		rooms        map[uuid.UUID]map[uuid.UUID]bool
		viewers      map[uuid.UUID]map[uuid.UUID]*viewerInfo
		userRooms    map[uuid.UUID]map[uuid.UUID]struct{}
		userViewing  map[uuid.UUID]map[uuid.UUID]struct{}
		anon         map[*Client]any
		alwaysOnline map[uuid.UUID]struct{}
		postWatchers map[uuid.UUID]map[*Client]struct{}
		mu           sync.RWMutex
		sidebar      *coalesce.Keyed[string, time.Time]
	}
)

const (
	ViewerStateActive = "active"
	ViewerStateIdle   = "idle"
)

var (
	topicNamespace = uuid.MustParse("1b671a64-40d5-491e-99b0-da01ff1f3341")
)

func NewClient(userID uuid.UUID, conn *websocket.Conn) *Client {
	return &Client{
		UserID:  userID,
		Conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		closeCh: make(chan any),
	}
}

func (c *Client) RecordPong() {
	sent := c.pingAt.Swap(0)
	if sent == 0 {
		return
	}

	sample := time.Since(time.Unix(0, sent)).Milliseconds()
	if sample < 0 {
		return
	}

	peak := c.rttPeak.Load()
	if peak <= 0 || sample >= peak {
		c.rttPeak.Store(sample)
		return
	}

	c.rttPeak.Store(peak - (peak-sample)/5)
}

func (c *Client) MeasuredRTT() int64 {
	return c.rttPeak.Load()
}

// Start launches the writer goroutine. Call once per client.
func (c *Client) Start() {
	if c.Conn == nil {
		return
	}

	go pprof.Do(context.Background(), pprof.Labels("conn", "ws"), func(context.Context) {
		c.writeLoop()
	})
}

func (c *Client) writeLoop() {
	pingTicker := time.NewTicker(pingInterval)
	defer func() {
		pingTicker.Stop()
		_ = c.Conn.Close()
	}()
	for {
		select {
		case data, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.Conn.WriteMessage(websocket.TextMessage, c.framePayload(data)); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			c.pingAt.Store(time.Now().UnixNano())
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

func (c *Client) framePayload(first []byte) []byte {
	if !c.batch {
		return first
	}

	items := [][]byte{first}
	size := len(first)
	for len(items) < maxBatchItems && size < maxBatchBytes {
		next, ok := c.TryReceive()
		if !ok {
			break
		}

		items = append(items, next)
		size += len(next)
	}

	if len(items) == 1 {
		return first
	}

	frame := make([]byte, 0, size+len(items)+1)
	frame = append(frame, '[')
	for i, item := range items {
		if i > 0 {
			frame = append(frame, ',')
		}
		frame = append(frame, item...)
	}

	return append(frame, ']')
}

func (c *Client) TryReceive() ([]byte, bool) {
	select {
	case data, ok := <-c.send:
		return data, ok
	default:
		return nil, false
	}
}

func (c *Client) closed() bool {
	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

func (c *Client) enqueue(data []byte) bool {
	select {
	case c.send <- data:
		return true
	case <-c.closeCh:
		return false
	default:
		c.kill()
		return false
	}
}

func (c *Client) enqueueLossy(data []byte) bool {
	select {
	case c.send <- data:
		return true
	case <-c.closeCh:
		return false
	default:
		return false
	}
}

func (c *Client) enqueueFrame(data []byte) bool {
	if len(c.send) >= cap(c.send)/2 {
		return false
	}

	return c.enqueueLossy(data)
}

// kill signals the writer to stop. Safe to call multiple times.
func (c *Client) kill() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
	})
}

// Close is the external shutdown trigger.
func (c *Client) Close() error {
	c.kill()
	return nil
}

func NewHub(name ...string) *Hub {
	hubName := "main"
	if len(name) > 0 && name[0] != "" {
		hubName = name[0]
	}

	h := &Hub{
		name:         hubName,
		tracer:       otel.Tracer(wsTracerName),
		clients:      make(map[uuid.UUID][]*Client),
		rooms:        make(map[uuid.UUID]map[uuid.UUID]bool),
		viewers:      make(map[uuid.UUID]map[uuid.UUID]*viewerInfo),
		userRooms:    make(map[uuid.UUID]map[uuid.UUID]struct{}),
		userViewing:  make(map[uuid.UUID]map[uuid.UUID]struct{}),
		anon:         make(map[*Client]any),
		alwaysOnline: make(map[uuid.UUID]struct{}),
		postWatchers: make(map[uuid.UUID]map[*Client]struct{}),
	}
	h.sidebar = coalesce.NewKeyed(defaultSidebarInterval, coalesce.Latest[time.Time], h.broadcastSidebarActivity)

	return h
}

func addToIndex(index map[uuid.UUID]map[uuid.UUID]struct{}, outer, inner uuid.UUID) {
	set := index[outer]
	if set == nil {
		set = make(map[uuid.UUID]struct{})
		index[outer] = set
	}

	set[inner] = struct{}{}
}

func removeFromIndex(index map[uuid.UUID]map[uuid.UUID]struct{}, outer, inner uuid.UUID) {
	set := index[outer]
	if set == nil {
		return
	}

	delete(set, inner)
	if len(set) == 0 {
		delete(index, outer)
	}
}

func (h *Hub) RegisterAnon(client *Client) {
	h.mu.Lock()
	h.anon[client] = struct{}{}
	h.mu.Unlock()

	recordConnectionOpened(h.name, false)

	client.Start()
}

func (h *Hub) UnregisterAnon(client *Client) {
	client.kill()

	h.mu.Lock()
	_, present := h.anon[client]
	delete(h.anon, client)
	h.mu.Unlock()

	if present {
		recordConnectionClosed(h.name, false)
	}
}

func (h *Hub) AddViewer(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[roomID] == nil {
		h.viewers[roomID] = make(map[uuid.UUID]*viewerInfo)
	}
	info := h.viewers[roomID][userID]
	if info == nil {
		info = &viewerInfo{}
		h.viewers[roomID][userID] = info
	}
	info.tabs++
	info.state = ViewerStateActive
	addToIndex(h.userViewing, userID, roomID)
}

func (h *Hub) RemoveViewer(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[roomID] == nil {
		return
	}
	info := h.viewers[roomID][userID]
	if info == nil {
		return
	}
	info.tabs--
	if info.tabs <= 0 {
		delete(h.viewers[roomID], userID)
		removeFromIndex(h.userViewing, userID, roomID)
	}
	if len(h.viewers[roomID]) == 0 {
		delete(h.viewers, roomID)
	}
}

func (h *Hub) SetViewerState(roomID, userID uuid.UUID, state string) bool {
	if state != ViewerStateActive && state != ViewerStateIdle {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[roomID] == nil {
		return false
	}
	info := h.viewers[roomID][userID]
	if info == nil || info.tabs <= 0 {
		return false
	}
	if info.state == state {
		return false
	}
	info.state = state
	return true
}

func (h *Hub) IsUserViewing(roomID, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.viewers[roomID] == nil {
		return false
	}
	info := h.viewers[roomID][userID]
	return info != nil && info.tabs > 0
}

func (h *Hub) GetRoomPresence(roomID uuid.UUID) map[uuid.UUID]string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[uuid.UUID]string)
	for uid, info := range h.viewers[roomID] {
		if info != nil && info.tabs > 0 {
			out[uid] = info.state
		}
	}
	return out
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	h.clients[client.UserID] = append(h.clients[client.UserID], client)
	h.mu.Unlock()

	recordConnectionOpened(h.name, true)

	client.Start()
}

func (h *Hub) Unregister(client *Client) []uuid.UUID {
	client.kill()

	h.mu.Lock()
	defer h.mu.Unlock()

	h.unwatchPostsLocked(client)

	var clearedRooms []uuid.UUID
	conns := h.clients[client.UserID]
	for i, c := range conns {
		if c == client {
			h.clients[client.UserID] = append(conns[:i], conns[i+1:]...)
			recordConnectionClosed(h.name, true)
			break
		}
	}
	if len(h.clients[client.UserID]) == 0 {
		delete(h.clients, client.UserID)

		for roomID := range h.userRooms[client.UserID] {
			members := h.rooms[roomID]
			delete(members, client.UserID)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}
		delete(h.userRooms, client.UserID)

		for roomID := range h.userViewing[client.UserID] {
			viewers := h.viewers[roomID]
			if _, ok := viewers[client.UserID]; ok {
				delete(viewers, client.UserID)
				clearedRooms = append(clearedRooms, roomID)
			}
			if len(viewers) == 0 {
				delete(h.viewers, roomID)
			}
		}
		delete(h.userViewing, client.UserID)
	}
	return clearedRooms
}

func (h *Hub) DisconnectUser(userID uuid.UUID) int {
	conns := h.snapshotConnsForUser(userID)
	for _, c := range conns {
		c.kill()
	}

	return len(conns)
}

// snapshotConnsForUser returns a copy of the user's clients without holding the lock for writes.
func (h *Hub) snapshotConnsForUser(userID uuid.UUID) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	src := h.clients[userID]
	if len(src) == 0 {
		return nil
	}
	out := make([]*Client, len(src))
	copy(out, src)
	return out
}

func (h *Hub) snapshotConnsForUsers(userIDs []uuid.UUID) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var out []*Client
	for _, uid := range userIDs {
		out = append(out, h.clients[uid]...)
	}
	return out
}

func (h *Hub) reapDead(dead []*Client) {
	for _, c := range dead {
		h.Unregister(c)
	}
}

func (h *Hub) sendToConns(conns []*Client, msg Message) {
	if len(conns) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var dead []*Client
	for _, client := range conns {
		if !client.enqueue(data) {
			dead = append(dead, client)
		}
	}

	if len(dead) > 0 {
		h.reapDead(dead)
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, msg Message) {
	h.sendToConns(h.snapshotConnsForUser(userID), msg)
}

func (h *Hub) SendToUsers(userIDs []uuid.UUID, msg Message) {
	h.sendToConns(h.snapshotConnsForUsers(userIDs), msg)
}

func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	allConns := make([]*Client, 0)
	for _, conns := range h.clients {
		allConns = append(allConns, conns...)
	}
	h.mu.RUnlock()

	h.sendToConns(allConns, msg)
}

func (h *Hub) BroadcastPublic(msg Message) {
	h.mu.RLock()
	authedConns := make([]*Client, 0, len(h.clients))
	for _, conns := range h.clients {
		authedConns = append(authedConns, conns...)
	}
	anonConns := make([]*Client, 0, len(h.anon))
	for c := range h.anon {
		anonConns = append(anonConns, c)
	}
	h.mu.RUnlock()

	if len(authedConns) == 0 && len(anonConns) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var deadAuthed []*Client
	for _, c := range authedConns {
		if !c.enqueue(data) {
			deadAuthed = append(deadAuthed, c)
		}
	}

	var deadAnon []*Client
	for _, c := range anonConns {
		if !c.enqueue(data) {
			deadAnon = append(deadAnon, c)
		}
	}

	h.reapDead(deadAuthed)
	for _, c := range deadAnon {
		h.UnregisterAnon(c)
	}
}

func (h *Hub) WatchPosts(client *Client, postIDs []uuid.UUID) {
	next := make(map[uuid.UUID]struct{}, min(len(postIDs), maxWatchedPosts))
	for _, postID := range postIDs {
		if len(next) >= maxWatchedPosts {
			break
		}
		next[postID] = struct{}{}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.unwatchPostsLocked(client)

	if len(next) == 0 || client.closed() {
		return
	}

	client.watchedPosts = next
	for postID := range next {
		watchers := h.postWatchers[postID]
		if watchers == nil {
			watchers = make(map[*Client]struct{})
			h.postWatchers[postID] = watchers
		}
		watchers[client] = struct{}{}
	}
}

func (h *Hub) unwatchPostsLocked(client *Client) {
	for postID := range client.watchedPosts {
		watchers := h.postWatchers[postID]
		delete(watchers, client)
		if len(watchers) == 0 {
			delete(h.postWatchers, postID)
		}
	}

	client.watchedPosts = nil
}

func (h *Hub) SendToPostWatchers(postID uuid.UUID, msg Message) {
	h.mu.RLock()
	watchers := h.postWatchers[postID]
	conns := make([]*Client, 0, len(watchers))
	for client := range watchers {
		conns = append(conns, client)
	}
	h.mu.RUnlock()

	h.sendToConns(conns, msg)
}

func (h *Hub) BumpSidebarActivity(key string) {
	if h == nil {
		return
	}

	h.sidebar.Add(context.Background(), key, time.Now())
}

func (h *Hub) broadcastSidebarActivity(_ context.Context, key string, at time.Time) {
	h.Broadcast(Message{
		Type: "sidebar_activity",
		Data: map[string]any{
			"key": key,
			"at":  at.UTC().Format(time.RFC3339),
		},
	})
}

func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if _, always := h.alwaysOnline[userID]; always {
		return true
	}

	return len(h.clients[userID]) > 0
}

func (h *Hub) IsAlwaysOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, always := h.alwaysOnline[userID]

	return always
}

func (h *Hub) SetAlwaysOnline(userIDs []uuid.UUID) {
	next := make(map[uuid.UUID]struct{}, len(userIDs))
	for _, id := range userIDs {
		next[id] = struct{}{}
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.alwaysOnline = next
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func TopicUUID(topic string) uuid.UUID {
	return uuid.NewSHA1(topicNamespace, []byte(topic))
}

func (h *Hub) JoinTopic(topic string, userID uuid.UUID) {
	h.JoinRoom(TopicUUID(topic), userID)
}

func (h *Hub) LeaveTopic(topic string, userID uuid.UUID) {
	h.LeaveRoom(TopicUUID(topic), userID)
}

func (h *Hub) BroadcastToTopic(topic string, msg Message) {
	_, span := h.tracer.Start(
		context.Background(),
		"ws.broadcast_topic",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("ws.topic", topic),
			attribute.String("ws.message_type", msg.Type),
		),
	)
	defer span.End()

	roomID := TopicUUID(topic)
	recipients := h.broadcastToRoomTraced(roomID, msg, uuid.Nil)
	span.SetAttributes(attribute.Int("ws.recipients", recipients))
}

func (h *Hub) JoinRoom(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.joinRoomLocked(roomID, userID)
}

func (h *Hub) JoinRooms(roomIDs []uuid.UUID, userID uuid.UUID) {
	if len(roomIDs) == 0 {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, roomID := range roomIDs {
		h.joinRoomLocked(roomID, userID)
	}
}

func (h *Hub) joinRoomLocked(roomID, userID uuid.UUID) {
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[uuid.UUID]bool)
	}
	h.rooms[roomID][userID] = true
	addToIndex(h.userRooms, userID, roomID)
}

func (h *Hub) LeaveRoom(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] != nil {
		delete(h.rooms[roomID], userID)
		if len(h.rooms[roomID]) == 0 {
			delete(h.rooms, roomID)
		}
	}
	removeFromIndex(h.userRooms, userID, roomID)
}

func (h *Hub) BroadcastToViewers(roomID uuid.UUID, msg Message, excludeUserID uuid.UUID) int {
	h.mu.RLock()
	recipients := 0
	var conns []*Client
	for uid, info := range h.viewers[roomID] {
		if uid == excludeUserID || info == nil || info.tabs <= 0 {
			continue
		}

		recipients++
		conns = append(conns, h.clients[uid]...)
	}
	h.mu.RUnlock()

	if recipients == 0 {
		return 0
	}

	h.sendToConns(conns, msg)

	return recipients
}

func (h *Hub) IsUserInRoom(roomID, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.rooms[roomID] == nil {
		return false
	}
	return h.rooms[roomID][userID]
}

func (h *Hub) BroadcastToRoom(roomID uuid.UUID, msg Message, excludeUserID uuid.UUID) {
	h.broadcastToRoomTraced(roomID, msg, excludeUserID)
}

func (h *Hub) BroadcastFrameToRoom(roomID uuid.UUID, msg Message) int {
	h.mu.RLock()
	var conns []*Client
	for uid := range h.rooms[roomID] {
		conns = append(conns, h.clients[uid]...)
	}
	h.mu.RUnlock()

	if len(conns) == 0 {
		return 0
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0
	}

	sent := 0
	for _, client := range conns {
		if client.enqueueFrame(data) {
			sent++
			continue
		}

		wsOutboundFramesDropped.Inc()
	}

	return sent
}

func (h *Hub) broadcastToRoomTraced(roomID uuid.UUID, msg Message, excludeUserID uuid.UUID) int {
	h.mu.RLock()
	members := h.rooms[roomID]
	recipients := 0
	var conns []*Client
	for uid := range members {
		if uid == excludeUserID {
			continue
		}

		recipients++
		conns = append(conns, h.clients[uid]...)
	}
	h.mu.RUnlock()

	if recipients == 0 {
		return 0
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return 0
	}

	var dead []*Client
	for _, client := range conns {
		if !client.enqueue(data) {
			dead = append(dead, client)
		}
	}
	if len(dead) > 0 {
		h.reapDead(dead)
	}
	return recipients
}
