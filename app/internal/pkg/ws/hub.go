// Package ws provides a WebSocket hub and client implementation for
// real-time bidirectional communication.
package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Message is the envelope for all WebSocket messages.
type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// Hub manages a set of active WebSocket clients and broadcasts messages.
type Hub struct {
	// clients maps user IDs to their connected WebSocket clients.
	clients    map[string]map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the Hub's main event loop in the current goroutine.
// It processes client registrations, unregistrations, and broadcasts.
// This method blocks — call it in a dedicated goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.userID] == nil {
				h.clients[client.userID] = make(map[*Client]bool)
			}
			h.clients[client.userID][client] = true
			h.mu.Unlock()

			zap.L().Debug("client registered",
				zap.String("user_id", client.userID),
				zap.Int("total_connections", h.userConnectionCount(client.userID)),
			)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.userID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.send)

					// Clean up user entry if no more connections
					if len(clients) == 0 {
						delete(h.clients, client.userID)
					}
				}
			}
			h.mu.Unlock()

			zap.L().Debug("client unregistered",
				zap.String("user_id", client.userID),
			)

		case msg := <-h.broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				zap.L().Error("failed to marshal broadcast message", zap.Error(err))
				continue
			}

			h.mu.RLock()
			for userID, clients := range h.clients {
				for client := range clients {
					select {
					case client.send <- data:
					default:
						// Client send buffer full, close connection
						close(client.send)
						delete(clients, client)
						zap.L().Warn("dropped slow client",
							zap.String("user_id", userID),
						)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToUser sends a message to all WebSocket connections of a specific user.
func (h *Hub) SendToUser(userID string, msg *Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		zap.L().Error("failed to marshal user message",
			zap.String("user_id", userID),
			zap.Error(err),
		)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.clients[userID]
	if !ok {
		zap.L().Debug("no connections for user", zap.String("user_id", userID))
		return
	}

	for client := range clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(clients, client)
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg *Message) {
	h.broadcast <- msg
}

// OnlineCount returns the total number of connected clients.
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.clients {
		count += len(clients)
	}
	return count
}

// HandleConnection registers a new WebSocket connection with the Hub and
// starts the read/write pump goroutines. This method is the primary entry
// point for external packages (e.g., the WebSocket handler) to add clients.
func (h *Hub) HandleConnection(conn *websocket.Conn, userID string) {
	client := newClient(h, conn, userID)
	h.register <- client

	go client.writePump()
	go client.readPump()
}

// userConnectionCount returns the number of connections for a specific user.
func (h *Hub) userConnectionCount(userID string) int {
	if clients, ok := h.clients[userID]; ok {
		return len(clients)
	}
	return 0
}
