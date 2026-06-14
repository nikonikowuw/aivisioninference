package mqttsync

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestMqttSyncManager_WaitAndResolve(t *testing.T) {
	// Attempt to connect to a local Redis instance.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running on localhost:6379, skipping integration test")
		return
	}
	defer rdb.Close()

	mgr := NewMqttSyncManager(rdb)
	traceID := "test-trace-id-123"
	expectedPayload := "test-success-response"

	// Launch a goroutine to resolve the traceID after a short sleep.
	go func() {
		time.Sleep(100 * time.Millisecond)
		resolveCtx, resolveCancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer resolveCancel()
		_ = mgr.Resolve(resolveCtx, traceID, expectedPayload)
	}()

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()

	payload, err := mgr.Wait(waitCtx, traceID, 1*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, expectedPayload, payload)
}

func TestMqttSyncManager_Timeout(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Redis is not running on localhost:6379, skipping integration test")
		return
	}
	defer rdb.Close()

	mgr := NewMqttSyncManager(rdb)
	traceID := "test-trace-id-timeout"

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer waitCancel()

	_, err := mgr.Wait(waitCtx, traceID, 100*time.Millisecond)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), context.DeadlineExceeded.Error())
}
