package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/google/uuid"
)

type (
	Message struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	Client struct {
		UserID uuid.UUID
		Conn   *websocket.Conn
		mu     sync.Mutex
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

func (c *Client) WriteMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteMessage(messageType, data)
}

func (c *Client) WriteControl(messageType int, data []byte, deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteControl(messageType, data, deadline)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.Close()
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
	defer h.mu.Unlock()
	h.clients[client.UserID] = append(h.clients[client.UserID], client)
}

func (h *Hub) Unregister(client *Client) []uuid.UUID {
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

func (h *Hub) SendToUser(userID uuid.UUID, msg Message) {
	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	var dead []*Client
	for _, client := range conns {
		if err := client.WriteMessage(websocket.TextMessage, data); err != nil {
			dead = append(dead, client)
		}
	}

	for _, client := range dead {
		h.Unregister(client)
	}
}

func (h *Hub) Broadcast(msg Message) {
	h.mu.RLock()
	var allConns []*Client
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

	for _, client := range allConns {
		_ = client.WriteMessage(websocket.TextMessage, data)
	}
}

func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
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
	var targetUserIDs []uuid.UUID
	for uid := range members {
		if uid != excludeUserID {
			targetUserIDs = append(targetUserIDs, uid)
		}
	}
	h.mu.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for _, uid := range targetUserIDs {
		h.mu.RLock()
		conns := h.clients[uid]
		h.mu.RUnlock()

		for _, client := range conns {
			_ = client.WriteMessage(websocket.TextMessage, data)
		}
	}
}
