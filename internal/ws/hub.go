package ws

import (
	"encoding/json"
	"sync"

	"github.com/fasthttp/websocket"
)

type (
	Message struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}

	Client struct {
		UserID int
		Conn   *websocket.Conn
	}

	Hub struct {
		clients map[int][]*Client
		mu      sync.RWMutex
	}
)

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int][]*Client),
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

func (h *Hub) SendToUser(userID int, msg Message) {
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

func (h *Hub) IsOnline(userID int) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}
