// Package redistest starts a throwaway Redis container, for cache tests.
package redistest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// image is the Redis version compose.yaml runs.
const image = "redis:8-alpine"

// Start runs a Redis container and returns a client for it. Call stop to remove
// it. Start one per test package, not per test, since starting is slow.
func Start(ctx context.Context) (client *redis.Client, stop func(), err error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("redistest: start container: %w", err)
	}

	stop = func() {
		if client != nil {
			_ = client.Close()
		}
		stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = container.Terminate(stopCtx)
	}
	defer func() {
		if err != nil {
			stop()
		}
	}()

	endpoint, err := container.PortEndpoint(ctx, "6379/tcp", "")
	if err != nil {
		return nil, nil, fmt.Errorf("redistest: endpoint: %w", err)
	}

	client = redis.NewClient(&redis.Options{Addr: endpoint})
	if err = client.Ping(ctx).Err(); err != nil {
		return nil, nil, fmt.Errorf("redistest: ping: %w", err)
	}

	return client, stop, nil
}

// Reset empties every Redis database, so each test starts from a cold cache.
func Reset(t testing.TB, client *redis.Client) {
	t.Helper()
	require.NoError(t, client.FlushAll(context.Background()).Err())
}
