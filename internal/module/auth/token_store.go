package auth

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RefreshStore keeps refresh tokens by their hash.
type RefreshStore interface {
	// Save stores the token for the user until ttl has passed.
	Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error
	// Consume removes the token and returns its user. It returns
	// ErrInvalidRefreshToken if the token is unknown or expired.
	Consume(ctx context.Context, tokenHash string) (int64, error)
	// Delete removes the token if it is stored.
	Delete(ctx context.Context, tokenHash string) error
}

// redisRefreshStore keeps refresh tokens in Redis. Redis expires them, and
// instances that share it also share the tokens.
type redisRefreshStore struct {
	client *redis.Client
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

func NewRedisRefreshStore(client *redis.Client) RefreshStore {
	return &redisRefreshStore{
		client: client,
	}
}

func (s *redisRefreshStore) Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	return s.client.Set(ctx, refreshKey(tokenHash), userID, ttl).Err()
}

// Consume uses GETDEL, so each refresh token can be used only once.
func (s *redisRefreshStore) Consume(ctx context.Context, tokenHash string) (int64, error) {
	val, err := s.client.GetDel(ctx, refreshKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrInvalidRefreshToken
	}
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

func (s *redisRefreshStore) Delete(ctx context.Context, tokenHash string) error {
	return s.client.Del(ctx, refreshKey(tokenHash)).Err()
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// refreshKey returns the Redis key for a token hash.
func refreshKey(tokenHash string) string {
	return refreshKeyPrefix + tokenHash
}
