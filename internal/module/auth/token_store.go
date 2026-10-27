package auth

import (
	"context"
	"sync"
	"time"
)

type memoryRefreshEntry struct {
	userID    int64
	expiresAt time.Time
}

// memoryRefreshStore keeps refresh tokens in process memory. Tokens are lost
// on restart and are not shared between instances.
type memoryRefreshStore struct {
	mu     sync.Mutex
	tokens map[string]memoryRefreshEntry
}

func NewMemoryRefreshStore() *memoryRefreshStore {
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

	s.tokens[tokenHash] = memoryRefreshEntry{
		userID:    userID,
		expiresAt: time.Now().Add(ttl),
	}

	return nil
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
