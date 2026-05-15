// Package task provides background task queue management using Asynq.
package task

import (
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

// NewServer creates a new Asynq server for processing tasks.
func NewServer(rdb *redis.Client) *asynq.Server {
	opts := rdb.Options()
	return asynq.NewServer(asynq.RedisClientOpt{
		Addr:         opts.Addr,
		Password:     opts.Password,
		DB:           opts.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
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
