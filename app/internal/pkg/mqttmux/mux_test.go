package mqttmux

import (
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

type mockMessage struct {
	topic   string
	payload []byte
}

func (m *mockMessage) Duplicate() bool   { return false }
func (m *mockMessage) Qos() byte         { return 0 }
func (m *mockMessage) Retained() bool    { return false }
func (m *mockMessage) Topic() string     { return m.topic }
func (m *mockMessage) MessageID() uint16 { return 0 }
func (m *mockMessage) Payload() []byte   { return m.payload }
func (m *mockMessage) Ack()              {}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern string
		topic   string
		matched bool
	}{
		// Exact matches
		{"aivision/edge", "aivision/edge", true},
		{"aivision/edge/1", "aivision/edge/2", false},

		// Single wildcard '+' matches
		{"aivision/edge/+", "aivision/edge/123", true},
		{"aivision/edge/+/status", "aivision/edge/123/status", true},
		{"aivision/edge/+/status", "aivision/edge/123/event", false},
		{"aivision/+/+/status", "aivision/edge/123/status", true},

		// Multi-level wildcard '#' matches
		{"aivision/edge/#", "aivision/edge/123", true},
		{"aivision/edge/#", "aivision/edge/123/status", true},
		{"aivision/edge/#", "aivision/edge/123/status/lifecycle", true},
		{"aivision/edge/#", "aivision/other", false},
		{"#", "aivision/edge/123", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" vs "+tt.topic, func(t *testing.T) {
			assert.Equal(t, tt.matched, Match(tt.pattern, tt.topic))
		})
	}
}

func TestMuxRegisterAndDispatch(t *testing.T) {
	mux := NewMux()
	var wg sync.WaitGroup

	var received1 string
	var received2 string

	wg.Add(2)

	mux.Register("aivision/edge/+/status", func(msg mqtt.Message) {
		received1 = msg.Topic()
		wg.Done()
	})

	mux.Register("aivision/edge/#", func(msg mqtt.Message) {
		received2 = msg.Topic()
		wg.Done()
	})

	msg := &mockMessage{
		topic:   "aivision/edge/device1/status",
		payload: []byte("online"),
	}

	mux.Dispatch(msg)

	// Wait with timeout to prevent blocking tests forever if dispatch fails
	c := make(chan struct{})
	go func() {
		wg.Wait()
		close(c)
	}()

	select {
	case <-c:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for dispatch handlers")
	}

	assert.Equal(t, "aivision/edge/device1/status", received1)
	assert.Equal(t, "aivision/edge/device1/status", received2)
}
