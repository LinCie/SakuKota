package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/extra/redisotel/v9"
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
	// Traces commands and reports pool usage through the global OpenTelemetry
	// providers, which do nothing until a command sets them.
	if err := errors.Join(redisotel.InstrumentTracing(client), redisotel.InstrumentMetrics(client)); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("instrument redis: %w", err)
	}

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
