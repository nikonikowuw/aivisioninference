package ws

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 4096
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

// newClient creates a new Client. It does not start the read/write pumps.
func newClient(hub *Hub, conn *websocket.Conn, userID string) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, 256),
		userID: userID,
	}
}

// readPump pumps messages from the websocket connection to the hub.
// It runs in its own goroutine and handles ping/pong, close, and
// incoming messages.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				zap.L().Warn("websocket unexpected close",
					zap.String("user_id", c.userID),
					zap.Error(err),
				)
			}
			break
		}

		// Handle incoming messages — parse as Message envelope
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			zap.L().Debug("invalid message format",
				zap.String("user_id", c.userID),
				zap.Error(err),
			)
			continue
		}

		// Dispatch based on message type
		switch msg.Type {
		case "ping":
			// Client-initiated ping — respond with pong
			pong := &Message{Type: "pong", Payload: time.Now().Unix()}
			data, _ := json.Marshal(pong)
			select {
			case c.send <- data:
			default:
			}
		default:
			zap.L().Debug("unhandled message type",
				zap.String("user_id", c.userID),
				zap.String("type", msg.Type),
			)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
// It runs in its own goroutine and handles pings, writes, and deadlines.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain queued messages into current write
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
