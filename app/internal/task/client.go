package task

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// Client is a wrapper around asynq.Client that implements the service.taskClient interface.
type Client struct {
	client    *asynq.Client
	redisOpts asynq.RedisClientOpt
}

// NewClient creates a new Client using the connection details from the given redis.Client.
func NewClient(rdb *redis.Client) *Client {
	opts := rdb.Options()
	redisOpts := asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	}
	client := asynq.NewClient(redisOpts)
	return &Client{client: client, redisOpts: redisOpts}
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

// EnqueueWithID 与 Enqueue 相同，但使用指定的 taskID 替代自动生成的随机 ID。
// 同一 ID 在队列中唯一存在，可用于后续通过 RemovePending 精准删除。
// 若任务已存在，自动忽略 asynq.ErrTaskIDConflict，实现幂等。
func (c *Client) EnqueueWithID(ctx context.Context, taskType string, payload interface{}, taskID string) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	t := asynq.NewTask(taskType, payloadBytes)
	_, err = c.client.EnqueueContext(ctx, t, asynq.TaskID(taskID))
	if err != nil && errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil
	}
	return err
}

// RemovePending 删除指定类型+实体ID 对应的待处理任务。
// taskID 的构造规则为 taskType + ":" + entityID，必须与 EnqueueWithID 保持一致。
func (c *Client) RemovePending(ctx context.Context, taskType, entityID string) error {
	taskID := taskType + ":" + entityID
	inspector := asynq.NewInspector(c.redisOpts)
	return inspector.DeleteTask("default", taskID)
}
