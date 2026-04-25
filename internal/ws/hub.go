package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
)

const (
	sendBufferSize = 64
	writeTimeout   = 10 * time.Second
	pingInterval   = 30 * time.Second
)

type (
	Message struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	Client struct {
		UserID    uuid.UUID
		Conn      *websocket.Conn
		send      chan []byte
		closeCh   chan struct{}
		closeOnce sync.Once
	}

	viewerInfo struct {
		tabs  int
		state string
	}

	Hub struct {
		clients map[uuid.UUID][]*Client
		rooms   map[uuid.UUID]map[uuid.UUID]bool
		viewers map[uuid.UUID]map[uuid.UUID]*viewerInfo
		mu      sync.RWMutex
	}
)

const (
	ViewerStateActive = "active"
	ViewerStateIdle   = "idle"
)

func NewClient(userID uuid.UUID, conn *websocket.Conn) *Client {
	return &Client{
		UserID:  userID,
		Conn:    conn,
		send:    make(chan []byte, sendBufferSize),
		closeCh: make(chan struct{}),
	}
}

// Start launches the writer goroutine. Call once per client.
func (c *Client) Start() {
	go c.writeLoop()
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
			if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-pingTicker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

// enqueue tries to push data onto the send channel without blocking.
// Returns false if the buffer is full (slow consumer); caller should drop the client.
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

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID][]*Client),
		rooms:   make(map[uuid.UUID]map[uuid.UUID]bool),
		viewers: make(map[uuid.UUID]map[uuid.UUID]*viewerInfo),
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

func (h *Hub) GetViewerState(roomID, userID uuid.UUID) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.viewers[roomID] == nil {
		return ""
	}
	info := h.viewers[roomID][userID]
	if info == nil || info.tabs <= 0 {
		return ""
	}
	return info.state
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
	client.Start()
}

func (h *Hub) Unregister(client *Client) []uuid.UUID {
	client.kill()

	h.mu.Lock()
	defer h.mu.Unlock()

	var clearedRooms []uuid.UUID
	conns := h.clients[client.UserID]
	for i, c := range conns {
		if c == client {
			h.clients[client.UserID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.clients[client.UserID]) == 0 {
		delete(h.clients, client.UserID)

		for roomID, members := range h.rooms {
			delete(members, client.UserID)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}

		for roomID, viewers := range h.viewers {
			if _, ok := viewers[client.UserID]; ok {
				delete(viewers, client.UserID)
				clearedRooms = append(clearedRooms, roomID)
			}
			if len(viewers) == 0 {
				delete(h.viewers, roomID)
			}
		}
	}
	return clearedRooms
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

func (h *Hub) SendToUser(userID uuid.UUID, msg Message) {
	conns := h.snapshotConnsForUser(userID)
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

func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	allConns := make([]*Client, 0)
	for _, conns := range h.clients {
		allConns = append(allConns, conns...)
	}
	h.mu.RUnlock()

	if len(allConns) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var dead []*Client
	for _, client := range allConns {
		if !client.enqueue(data) {
			dead = append(dead, client)
		}
	}
	if len(dead) > 0 {
		h.reapDead(dead)
	}
}

func (h *Hub) BumpSidebarActivity(key string) {
	if h == nil {
		return
	}
	h.Broadcast(Message{
		Type: "sidebar_activity",
		Data: map[string]interface{}{
			"key": key,
			"at":  time.Now().UTC().Format(time.RFC3339),
		},
	})
}

func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

var topicNamespace = uuid.MustParse("1b671a64-40d5-491e-99b0-da01ff1f3341")

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
	h.BroadcastToRoom(TopicUUID(topic), msg, uuid.Nil)
}

func (h *Hub) JoinRoom(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[uuid.UUID]bool)
	}
	h.rooms[roomID][userID] = true
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
	h.mu.RLock()
	members := h.rooms[roomID]
	targetUserIDs := make([]uuid.UUID, 0, len(members))
	for uid := range members {
		if uid != excludeUserID {
			targetUserIDs = append(targetUserIDs, uid)
		}
	}
	h.mu.RUnlock()

	if len(targetUserIDs) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	conns := h.snapshotConnsForUsers(targetUserIDs)

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
