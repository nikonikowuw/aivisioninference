package mqttsync

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MqttSyncManager coordinates sync-over-async communication using Redis Pub/Sub.
type MqttSyncManager struct {
	rdb *redis.Client
}

// NewMqttSyncManager creates a new MqttSyncManager.
func NewMqttSyncManager(rdb *redis.Client) *MqttSyncManager {
	return &MqttSyncManager{
		rdb: rdb,
	}
}

// channelName returns the Redis channel name for a specific trace ID.
func (m *MqttSyncManager) channelName(traceID string) string {
	return fmt.Sprintf("mqttsync:channel:%s", traceID)
}

// Wait blocks until the response for the given traceID is received, or the timeout/context expires.
func (m *MqttSyncManager) Wait(ctx context.Context, traceID string, timeout time.Duration) (string, error) {
	if m.rdb == nil {
		return "", fmt.Errorf("redis client is nil")
	}

	pubsub := m.rdb.Subscribe(ctx, m.channelName(traceID))
	defer pubsub.Close()

	// Ensure the subscription is registered before returning or waiting.
	_, err := pubsub.Receive(ctx)
	if err != nil {
		return "", fmt.Errorf("subscribe failed: %w", err)
	}

	ch := pubsub.Channel()

	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		timeoutChan = timer.C
	}

	select {
	case msg := <-ch:
		return msg.Payload, nil
	case <-ctx.Done():
		return "", ctx.Err()
	case <-timeoutChan:
		return "", context.DeadlineExceeded
	}
}

// Resolve publishes the response payload to the Redis channel for traceID, waking up the blocked routine.
func (m *MqttSyncManager) Resolve(ctx context.Context, traceID string, payload string) error {
	if m.rdb == nil {
		return fmt.Errorf("redis client is nil")
	}

	err := m.rdb.Publish(ctx, m.channelName(traceID), payload).Err()
	if err != nil {
		return fmt.Errorf("publish failed: %w", err)
	}
	return nil
}
