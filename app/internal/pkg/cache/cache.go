// Package cache provides Redis client connection management.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// New creates a new Redis client, pings the server, and returns the connection.
// Use db=0 for the default Redis database.
func New(host string, port int, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	zap.L().Info("redis connected",
		zap.String("addr", fmt.Sprintf("%s:%d", host, port)),
		zap.Int("db", db),
	)

	return rdb, nil
}
