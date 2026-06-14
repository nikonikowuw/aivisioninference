package mqttmux

import (
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// HandlerFunc is the signature of callbacks registered in the Mux.
type HandlerFunc func(message mqtt.Message)

// route represents a registered pattern and its handler.
type route struct {
	pattern string
	handler HandlerFunc
}

// Mux is a thread-safe MQTT topic multiplexer.
type Mux struct {
	mu     sync.RWMutex
	routes []route
}

// NewMux creates a new MQTT topic multiplexer.
func NewMux() *Mux {
	return &Mux{
		routes: make([]route, 0),
	}
}

// Register registers a handler function for a topic pattern.
// Supported wildcards:
// - '+' matches a single topic level (e.g. 'aivision/edge/+/status')
// - '#' matches all remaining levels, must be at the end (e.g. 'aivision/edge/#')
func (m *Mux) Register(pattern string, handler HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routes = append(m.routes, route{
		pattern: pattern,
		handler: handler,
	})
}

// Dispatch matches the topic of the message and runs all matching handlers.
func (m *Mux) Dispatch(message mqtt.Message) {
	m.mu.RLock()
	routes := make([]route, len(m.routes))
	copy(routes, m.routes)
	m.mu.RUnlock()

	topic := message.Topic()
	for _, r := range routes {
		if Match(r.pattern, topic) {
			go r.handler(message)
		}
	}
}

// Match checks if an MQTT topic matches a pattern.
func Match(pattern, topic string) bool {
	if pattern == topic {
		return true
	}

	pParts := strings.Split(pattern, "/")
	tParts := strings.Split(topic, "/")

	for i := 0; i < len(pParts); i++ {
		pPart := pParts[i]

		// Multi-level wildcard '#' matches everything remaining.
		if pPart == "#" {
			// # must be at the end, or it's invalid. If it's valid, it matches everything here on.
			return i == len(pParts)-1 && len(tParts) >= i
		}

		// If topic ran out of parts but pattern hasn't, and it's not '#', no match.
		if i >= len(tParts) {
			return false
		}

		// Single-level wildcard '+' matches any single part.
		if pPart == "+" {
			continue
		}

		if pPart != tParts[i] {
			return false
		}
	}

	// Topic must not have remaining parts unless pattern ended with #.
	return len(pParts) == len(tParts)
}
