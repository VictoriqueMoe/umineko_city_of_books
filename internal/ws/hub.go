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
		mu      sync.RWMutex
	}
)

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID][]*Client),
	}
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

func (h *Hub) IsOnline(userID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
