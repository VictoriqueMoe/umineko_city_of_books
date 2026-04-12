package ws

import (
	"encoding/json"
	"sync"

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
	}

	Hub struct {
		clients map[uuid.UUID][]*Client
		rooms   map[uuid.UUID]map[uuid.UUID]bool
		viewers map[uuid.UUID]map[uuid.UUID]int
		mu      sync.RWMutex
	}
)

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID][]*Client),
		rooms:   make(map[uuid.UUID]map[uuid.UUID]bool),
		viewers: make(map[uuid.UUID]map[uuid.UUID]int),
	}
}

func (h *Hub) AddViewer(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[roomID] == nil {
		h.viewers[roomID] = make(map[uuid.UUID]int)
	}
	h.viewers[roomID][userID]++
}

func (h *Hub) RemoveViewer(roomID, userID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.viewers[roomID] == nil {
		return
	}
	h.viewers[roomID][userID]--
	if h.viewers[roomID][userID] <= 0 {
		delete(h.viewers[roomID], userID)
	}
	if len(h.viewers[roomID]) == 0 {
		delete(h.viewers, roomID)
	}
}

func (h *Hub) IsUserViewing(roomID, userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.viewers[roomID] == nil {
		return false
	}
	return h.viewers[roomID][userID] > 0
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.UserID] = append(h.clients[client.UserID], client)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

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
	}
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
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
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
		client.Conn.WriteMessage(websocket.TextMessage, data)
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
			client.Conn.WriteMessage(websocket.TextMessage, data)
		}
	}
}
