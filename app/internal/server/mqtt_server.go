package server

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/handler"
	"github.com/niko-admin/niko-admin/internal/pkg/mqttmux"
)

// MqttServer manages the MQTT subscription lifecycle and routes messages to the multiplexer.
type MqttServer struct {
	client  mqtt.Client
	mux     *mqttmux.Mux
	handler *handler.EdgeMqttHandler
}

// NewMqttServer creates a new MqttServer.
func NewMqttServer(client mqtt.Client, mux *mqttmux.Mux, handler *handler.EdgeMqttHandler) *MqttServer {
	return &MqttServer{
		client:  client,
		mux:     mux,
		handler: handler,
	}
}

// Start registers handlers and subscribes to the edge topics.
func (s *MqttServer) Start() error {
	// Register callbacks in the multiplexer
	s.mux.Register("aivision/edge/+/status/heartbeat", s.handler.HandleHeartbeat)
	s.mux.Register("aivision/edge/+/status/lifecycle", s.handler.HandleLifecycle)
	s.mux.Register("aivision/edge/+/response/+", s.handler.HandleStreamStatus)
	s.mux.Register("aivision/edge/+/event/inference", s.handler.HandleInferenceResult)
	s.mux.Register("aivision/edge/+/event/metrics", s.handler.HandleEngineMetrics)

	// Subscribe to aivision/edge/# (QoS 1)
	token := s.client.Subscribe("aivision/edge/#", 1, func(c mqtt.Client, msg mqtt.Message) {
		s.mux.Dispatch(msg)
	})
	token.Wait()
	if token.Error() != nil {
		return token.Error()
	}

	zap.L().Info("MQTT Server: subscribed to aivision/edge/#")
	return nil
}

// Stop unsubscribes from topics and disconnects the client.
func (s *MqttServer) Stop() {
	zap.L().Info("MQTT Server: unsubscribing and disconnecting...")
	s.client.Unsubscribe("aivision/edge/#")
	s.client.Disconnect(250)
}
