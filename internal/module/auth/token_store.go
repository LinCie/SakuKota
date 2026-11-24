package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"sakukota-go/internal/domain"
)

// RedisRefreshStore keeps refresh tokens in Redis. Redis expires them, and
// instances that share it also share the tokens.
type RedisRefreshStore struct {
	client *redis.Client
}

// NewRedisRefreshStore returns a RedisRefreshStore that uses client.
func NewRedisRefreshStore(client *redis.Client) *RedisRefreshStore {
	return &RedisRefreshStore{client: client}
}

// Save stores the token for the user until ttl has passed.
func (s *RedisRefreshStore) Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	return s.client.Set(ctx, refreshKey(tokenHash), userID, ttl).Err()
}

// Consume removes the token and returns its user. It uses GETDEL, so each
// refresh token can be used only once. It returns domain.ErrInvalidRefreshToken
// if the token is unknown or expired.
func (s *RedisRefreshStore) Consume(ctx context.Context, tokenHash string) (int64, error) {
	val, err := s.client.GetDel(ctx, refreshKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, domain.ErrInvalidRefreshToken
	}
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

// Delete removes the token if it is stored.
func (s *RedisRefreshStore) Delete(ctx context.Context, tokenHash string) error {
	return s.client.Del(ctx, refreshKey(tokenHash)).Err()
}

// refreshKey returns the Redis key for a token hash.
func refreshKey(tokenHash string) string {
	return refreshKeyPrefix + tokenHash
}
