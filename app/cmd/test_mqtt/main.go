package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func runTest() error {
	broker := "tcp://test.mosquitto.org:1883"
	nodeID := fmt.Sprintf("test-node-integration-%d", time.Now().UnixNano())

	// Create channels to receive messages
	lifecycleChan := make(chan string, 1)
	responseChan := make(chan string, 1)

	// Set up MQTT client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(fmt.Sprintf("integration-test-go-%d", time.Now().UnixNano()))
	opts.SetCleanSession(true)

	// Connect to broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to broker: %w", token.Error())
	}
	defer client.Disconnect(250)

	log.Println("[Go Test Client] Connected to MQTT broker:", broker)

	// Subscribe to lifecycle topic
	lifecycleTopic := fmt.Sprintf("aivision/edge/%s/status/lifecycle", nodeID)
	log.Println("[Go Test Client] Subscribing to:", lifecycleTopic)
	if token := client.Subscribe(lifecycleTopic, 1, func(c mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		log.Println("[Go Test Client] Received lifecycle message:", payload)
		lifecycleChan <- payload
	}); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to lifecycle: %w", token.Error())
	}

	// Subscribe to response topic
	responseTopic := fmt.Sprintf("aivision/edge/%s/response/self_check", nodeID)
	log.Println("[Go Test Client] Subscribing to:", responseTopic)
	if token := client.Subscribe(responseTopic, 1, func(c mqtt.Client, msg mqtt.Message) {
		payload := string(msg.Payload())
		log.Println("[Go Test Client] Received response message:", payload)
		responseChan <- payload
	}); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to subscribe to response: %w", token.Error())
	}

	// Launch C++ engine in background
	// Command: ./build/aivision-engine --enable-mqtt --mqtt-broker tcp://test.mosquitto.org:1883
	cmd := exec.Command("../engine/build/aivision-engine",
		"--enable-mqtt",
		"--mqtt-broker", broker,
	)
	// Set node_id env variable for engine config
	cmd.Env = append(os.Environ(), "NIKO_ENGINE_NODE_ID="+nodeID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Println("[Go Test Client] Starting C++ engine...")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start C++ engine: %w", err)
	}
	defer func() {
		log.Println("[Go Test Client] Terminating C++ engine...")
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	// Wait for engine online
	select {
	case payload := <-lifecycleChan:
		if payload != "online" {
			return fmt.Errorf("expected lifecycle 'online', got: %s", payload)
		}
		log.Println("[Go Test Client] C++ engine is ONLINE.")
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for engine lifecycle 'online'")
	}

	// Publish command to engine
	cmdTopic := fmt.Sprintf("aivision/edge/%s/cmd/self_check", nodeID)
	traceID := "trace-integration-test-999"
	cmdPayload := map[string]interface{}{
		"trace_id":     traceID,
		"download_url": "http://example.com/invalid.tar",
		"token":        "testtoken",
		"algo_name":    "face_detect",
		"version":      "1.0.0",
	}
	payloadBytes, _ := json.Marshal(cmdPayload)
	log.Printf("[Go Test Client] Publishing command to %s: %s", cmdTopic, string(payloadBytes))
	token := client.Publish(cmdTopic, 1, false, payloadBytes)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("failed to publish command: %w", token.Error())
	}

	// Wait for response
	select {
	case responsePayload := <-responseChan:
		log.Printf("[Go Test Client] Received response payload: %s", responsePayload)
		var responseMap map[string]interface{}
		if err := json.Unmarshal([]byte(responsePayload), &responseMap); err != nil {
			return fmt.Errorf("failed to parse response JSON: %w", err)
		}
		if responseMap["trace_id"] != traceID {
			return fmt.Errorf("expected trace_id %s, got %v", traceID, responseMap["trace_id"])
		}
		if responseMap["success"] != false {
			return fmt.Errorf("expected success to be false, got %v", responseMap["success"])
		}
		if responseMap["algo_name"] != "face_detect" {
			return fmt.Errorf("expected algo_name 'face_detect', got %v", responseMap["algo_name"])
		}
		log.Println("[Go Test Client] Integration test PASSED successfully!")
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("timeout waiting for command response")
	}
}

func main() {
	if err := runTest(); err != nil {
		log.Fatalf("Integration test FAILED: %v", err)
	}
	log.Println("Done.")
}
