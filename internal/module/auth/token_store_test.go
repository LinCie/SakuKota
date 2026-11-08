package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMemoryRefreshStore_SaveEvictsExpiredTokens(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryRefreshStore().(*memoryRefreshStore)

	// An expired token that nobody presents again, and a valid one.
	assert.NoError(t, store.Save(ctx, "expired", 1, -time.Minute))
	assert.NoError(t, store.Save(ctx, "valid", 1, time.Hour))
	assert.Contains(t, store.tokens, "expired", "swept too early: the sweep is rate limited")

	// Once the sweep is due, the next Save removes what has expired.
	store.nextSweep = time.Time{}
	assert.NoError(t, store.Save(ctx, "fresh", 2, time.Hour))

	assert.NotContains(t, store.tokens, "expired")
	assert.Contains(t, store.tokens, "valid")
	assert.Contains(t, store.tokens, "fresh")
}
