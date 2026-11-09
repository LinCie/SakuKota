package auth

import (
	"context"
	"errors"
	"strconv"
	"sync"
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

type memoryRefreshEntry struct {
	userID    int64
	expiresAt time.Time
}

// memoryRefreshStore keeps refresh tokens in process memory. Tokens are lost
// on restart and are not shared between instances.
type memoryRefreshStore struct {
	mu     sync.Mutex
	tokens map[string]memoryRefreshEntry
	// nextSweep is when Save next removes expired tokens.
	nextSweep time.Time
}

// refreshSweepInterval is the least time between two sweeps for expired
// tokens. A sweep visits every token, so it must not run on every Save.
const refreshSweepInterval = time.Minute

func NewMemoryRefreshStore() RefreshStore {
	return &memoryRefreshStore{
		tokens: make(map[string]memoryRefreshEntry),
	}
}

// -----------------------------------------------------------------------------
// memoryRefreshStore
// -----------------------------------------------------------------------------

func (s *memoryRefreshStore) Save(_ context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if now.After(s.nextSweep) {
		s.sweepExpired(now)
		s.nextSweep = now.Add(refreshSweepInterval)
	}

	s.tokens[tokenHash] = memoryRefreshEntry{
		userID:    userID,
		expiresAt: now.Add(ttl),
	}

	return nil
}

// sweepExpired removes tokens that expired before now. Tokens are otherwise
// removed only when presented, so one that is never used again would stay.
// The caller holds s.mu.
func (s *memoryRefreshStore) sweepExpired(now time.Time) {
	for hash, e := range s.tokens {
		if now.After(e.expiresAt) {
			delete(s.tokens, hash)
		}
	}
}

// Consume atomically looks up and removes a token, so each refresh token can
// be used only once.
func (s *memoryRefreshStore) Consume(_ context.Context, tokenHash string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.tokens[tokenHash]
	delete(s.tokens, tokenHash)
	if !ok || time.Now().After(e.expiresAt) {
		return 0, ErrInvalidRefreshToken
	}
	return e.userID, nil
}

func (s *memoryRefreshStore) Delete(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tokens, tokenHash)
	return nil
}

// -----------------------------------------------------------------------------
// redisRefreshStore
// -----------------------------------------------------------------------------

// redisRefreshKeyPrefix namespaces refresh token keys in Redis.
const redisRefreshKeyPrefix = "auth:refresh:"

// redisRefreshStore keeps refresh tokens in Redis. Redis expires them, and
// instances that share it also share the tokens.
type redisRefreshStore struct {
	client *redis.Client
}

func NewRedisRefreshStore(client *redis.Client) RefreshStore {
	return &redisRefreshStore{client: client}
}

func (s *redisRefreshStore) key(tokenHash string) string {
	return redisRefreshKeyPrefix + tokenHash
}

func (s *redisRefreshStore) Save(ctx context.Context, tokenHash string, userID int64, ttl time.Duration) error {
	// Redis treats a zero ttl as no expiry and rejects a negative one. A
	// token that is already expired is not worth storing.
	if ttl <= 0 {
		return nil
	}

	return s.client.Set(ctx, s.key(tokenHash), userID, ttl).Err()
}

// Consume atomically gets and removes a token, so each refresh token can be
// used only once.
func (s *redisRefreshStore) Consume(ctx context.Context, tokenHash string) (int64, error) {
	val, err := s.client.GetDel(ctx, s.key(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrInvalidRefreshToken
	}
	if err != nil {
		return 0, err
	}

	return strconv.ParseInt(val, 10, 64)
}

func (s *redisRefreshStore) Delete(ctx context.Context, tokenHash string) error {
	return s.client.Del(ctx, s.key(tokenHash)).Err()
}
