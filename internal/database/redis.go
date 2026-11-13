package database

import (
	"context"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient connects to Redis at redisURL and checks that it answers a ping.
// If the ping fails, the client is closed and the error is returned.
func NewRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
