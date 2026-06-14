package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHub_Run(t *testing.T) {
	// Initialize hub
	hub := NewHub()
	go hub.Run()

	// Create test server
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade connection: %v", err)
			return
		}
		userID := r.URL.Query().Get("user_id")
		hub.HandleConnection(conn, userID)
	}))
	defer server.Close()

	// Dial WebSocket client 1
	u := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=user1"
	wsConn1, _, err := websocket.DefaultDialer.Dial(u, nil)
	require.NoError(t, err)
	defer wsConn1.Close()

	// Dial WebSocket client 2
	u2 := "ws" + strings.TrimPrefix(server.URL, "http") + "?user_id=user2"
	wsConn2, _, err := websocket.DefaultDialer.Dial(u2, nil)
	require.NoError(t, err)
	defer wsConn2.Close()

	// Wait for connections to register
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 2, hub.OnlineCount())
	assert.Equal(t, 1, hub.userConnectionCount("user1"))
	assert.Equal(t, 1, hub.userConnectionCount("user2"))

	// Test Broadcast
	broadcastMsg := &Message{
		Type:    "broadcast_test",
		Payload: "hello all",
	}
	hub.Broadcast(broadcastMsg)

	// Read message on client 1
	var readMsg1 Message
	_, p1, err := wsConn1.ReadMessage()
	require.NoError(t, err)
	err = json.Unmarshal(p1, &readMsg1)
	require.NoError(t, err)
	assert.Equal(t, "broadcast_test", readMsg1.Type)
	assert.Equal(t, "hello all", readMsg1.Payload)

	// Read message on client 2
	var readMsg2 Message
	_, p2, err := wsConn2.ReadMessage()
	require.NoError(t, err)
	err = json.Unmarshal(p2, &readMsg2)
	require.NoError(t, err)
	assert.Equal(t, "broadcast_test", readMsg2.Type)
	assert.Equal(t, "hello all", readMsg2.Payload)

	// Test SendToUser
	userMsg := &Message{
		Type:    "user_test",
		Payload: "hello user1",
	}
	hub.SendToUser("user1", userMsg)

	// Read message on client 1 (should be user_test)
	_, p1, err = wsConn1.ReadMessage()
	require.NoError(t, err)
	err = json.Unmarshal(p1, &readMsg1)
	require.NoError(t, err)
	assert.Equal(t, "user_test", readMsg1.Type)
	assert.Equal(t, "hello user1", readMsg1.Payload)

	// Test Client Ping response (readPump switches type "ping")
	pingMsg := &Message{
		Type:    "ping",
		Payload: nil,
	}
	pingBytes, _ := json.Marshal(pingMsg)
	err = wsConn1.WriteMessage(websocket.TextMessage, pingBytes)
	require.NoError(t, err)

	// Read message on client 1 (should be pong)
	_, p1, err = wsConn1.ReadMessage()
	require.NoError(t, err)
	err = json.Unmarshal(p1, &readMsg1)
	require.NoError(t, err)
	assert.Equal(t, "pong", readMsg1.Type)

	// Test Unregister
	wsConn1.Close()
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, hub.OnlineCount())
	assert.Equal(t, 0, hub.userConnectionCount("user1"))
}
