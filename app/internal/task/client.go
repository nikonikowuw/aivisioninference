package task

import (
	"context"
	"encoding/json"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Client is a wrapper around asynq.Client that implements the service.taskClient interface.
type Client struct {
	client *asynq.Client
}

// NewClient creates a new Client using the connection details from the given redis.Client.
func NewClient(rdb *redis.Client) *Client {
	opts := rdb.Options()
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	})
	return &Client{client: client}
}

// Close closes the connection to the redis-backed task queue.
func (c *Client) Close() error {
	return c.client.Close()
}

// Enqueue serializes the payload to JSON and enqueues the task with the given type.
func (c *Client) Enqueue(ctx context.Context, taskType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	t := asynq.NewTask(taskType, payloadBytes)
	_, err = c.client.EnqueueContext(ctx, t)
	return err
}
