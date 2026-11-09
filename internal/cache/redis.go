package cache

import (
	"context"

	"rantau-go/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisCache(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opt)

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}
