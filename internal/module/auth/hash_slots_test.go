package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/config"
)

func TestService_HashWaitsForFreeSlotAndStopsWhenContextEnds(t *testing.T) {
	s := NewService(config.Config{}, nil, NewMemoryRefreshStore()).(*service)

	// Every slot is taken, as if maxConcurrentHashes hashes were running.
	for range maxConcurrentHashes {
		s.hashSlots <- struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.hash(ctx, "correct-horse")
	assert.ErrorIs(t, err, context.Canceled, "hash with no free slot")

	// A freed slot lets the next hash through, and returns it afterwards.
	<-s.hashSlots
	_, err = s.hash(context.Background(), "correct-horse")
	require.NoError(t, err, "hash with a free slot")
	assert.Len(t, s.hashSlots, maxConcurrentHashes-1, "slots held after the hash finished")
}
