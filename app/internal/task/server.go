// Package task provides background task queue management using Asynq.
package task

import (
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// NewServer creates a new Asynq server for processing tasks.
func NewServer(rdb *redis.Client) *asynq.Server {
	return asynq.NewServer(asynq.RedisClientOpt{
		Addr:     rdb.Options().Addr,
		Password: rdb.Options().Password,
		DB:       rdb.Options().DB,
	}, asynq.Config{
		Concurrency: 10,
	})
}

// NewMux creates a new ServeMux and registers all task handlers.
func NewMux() *asynq.ServeMux {
	mux := asynq.NewServeMux()
	RegisterHandlers(mux)
	return mux
}
