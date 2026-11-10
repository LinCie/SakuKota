//go:build integration

package storagecleanup_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/storagecleanup"
	"rantau-go/internal/testutil/pgtest"
)

// testDB is shared by the tests in this file. pgtest.Reset empties it before
// each test.
var testDB *sqlx.DB

func TestMain(m *testing.M) {
	db, stop, err := pgtest.Start(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testDB = db

	code := m.Run()
	stop()
	os.Exit(code)
}

// newRepository returns a repository over an emptied database.
func newRepository(t *testing.T) storagecleanup.Repository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return storagecleanup.NewPostgresRepository(testDB)
}

// queue adds key to the queue, due dueIn from now (negative for already due),
// after the given number of failed attempts.
func queue(t *testing.T, key string, attempts int, dueIn time.Duration) {
	t.Helper()

	_, err := testDB.Exec(
		`INSERT INTO STORAGE_DELETIONS (key, attempts, next_attempt_at)
		VALUES ($1, $2, NOW() + make_interval(secs => $3))`,
		key, attempts, dueIn.Seconds(),
	)
	require.NoError(t, err)
}

// queueState is a queued deletion as stored.
type queueState struct {
	Attempts  int     `db:"attempts"`
	LastError *string `db:"last_error"`
	// DueIn is the seconds until the next attempt, negative if it is due.
	DueIn float64 `db:"due_in"`
}

func stateOf(t *testing.T, key string) queueState {
	t.Helper()

	var s queueState
	err := testDB.Get(&s,
		`SELECT attempts, last_error, EXTRACT(EPOCH FROM next_attempt_at - NOW())::float8 AS due_in
		FROM STORAGE_DELETIONS WHERE key = $1`, key)
	require.NoError(t, err)

	return s
}

func keysOf(deletions []storagecleanup.Deletion) []string {
	keys := make([]string, 0, len(deletions))
	for _, d := range deletions {
		keys = append(keys, d.Key)
	}

	return keys
}

func TestRepository_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("takes the key off the queue and leaves the others", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, 0)
		queue(t, "b.jpg", 0, 0)

		require.NoError(t, repo.Remove(ctx, "a.jpg"))

		assert.Equal(t, []string{"b.jpg"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("is not an error for a key that is not queued", func(t *testing.T) {
		repo := newRepository(t)

		assert.NoError(t, repo.Remove(ctx, "missing.jpg"))
	})
}

func TestRepository_ClaimDue(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the due deletions with their attempts, longest due first", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "newer.jpg", 0, -time.Minute)
		queue(t, "older.jpg", 3, -time.Hour)
		queue(t, "oldest.jpg", 1, -2*time.Hour)

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []storagecleanup.Deletion{
			{Key: "oldest.jpg", Attempts: 1},
			{Key: "older.jpg", Attempts: 3},
			{Key: "newer.jpg", Attempts: 0},
		}, got)
	})

	t.Run("orders deletions that became due together by key", func(t *testing.T) {
		repo := newRepository(t)
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key, next_attempt_at)
			VALUES ('b.jpg', '2026-01-01T00:00:00Z'), ('a.jpg', '2026-01-01T00:00:00Z')`)
		require.NoError(t, err)

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, keysOf(got))
	})

	t.Run("skips deletions that are not due yet", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "due.jpg", 0, -time.Minute)
		queue(t, "later.jpg", 0, time.Hour)

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []string{"due.jpg"}, keysOf(got))
	})

	t.Run("claims at most limit, the longest due first", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "c.jpg", 0, -time.Minute)
		queue(t, "a.jpg", 0, -3*time.Hour)
		queue(t, "b.jpg", 0, -2*time.Hour)

		got, err := repo.ClaimDue(ctx, 2)

		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, keysOf(got))
		assert.Less(t, stateOf(t, "c.jpg").DueIn, 0.0, "the rest stay due")
	})

	t.Run("holds claimed deletions back for a lease", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 2, -time.Minute)

		first, err := repo.ClaimDue(ctx, 10)
		require.NoError(t, err)
		second, err := repo.ClaimDue(ctx, 10)
		require.NoError(t, err)

		assert.Equal(t, []string{"a.jpg"}, keysOf(first))
		assert.Empty(t, second, "a claimed deletion is not claimed again")
		state := stateOf(t, "a.jpg")
		assert.InDelta(t, (15 * time.Minute).Seconds(), state.DueIn, 30, "the lease is 15 minutes")
		assert.Equal(t, 2, state.Attempts, "claiming does not count as an attempt")
	})

	t.Run("returns an empty, non-nil slice when nothing is due", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("gives each deletion to one of several workers claiming at once", func(t *testing.T) {
		repo := newRepository(t)
		const keys, workers = 60, 6
		for i := range keys {
			queue(t, fmt.Sprintf("key-%02d.jpg", i), 0, -time.Minute)
		}

		var wg sync.WaitGroup
		claimed := make([][]storagecleanup.Deletion, workers)
		errs := make([]error, workers)
		for w := range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				claimed[w], errs[w] = repo.ClaimDue(ctx, 20)
			}()
		}
		wg.Wait()

		seen := map[string]int{}
		for w := range workers {
			require.NoError(t, errs[w])
			for _, key := range keysOf(claimed[w]) {
				seen[key]++
			}
		}
		assert.Len(t, seen, keys, "every deletion is claimed")
		for key, n := range seen {
			assert.Equal(t, 1, n, "%s is claimed once", key)
		}
	})
}

func TestRepository_Reschedule(t *testing.T) {
	ctx := context.Background()

	t.Run("records the failed attempt and makes the deletion due after the delay", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", 30*time.Minute, "storage unavailable"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 1, state.Attempts)
		assert.Equal(t, "storage unavailable", *state.LastError)
		assert.InDelta(t, (30 * time.Minute).Seconds(), state.DueIn, 30)
	})

	t.Run("counts each failure and keeps the latest error", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Minute, "first"))
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "second"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 2, state.Attempts)
		assert.Equal(t, "second", *state.LastError)
		assert.InDelta(t, time.Hour.Seconds(), state.DueIn, 30)
	})

	t.Run("keeps the deletion from being claimed until the delay passes", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, -time.Minute)
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "failed"))

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("makes the deletion claimable at once when the delay is zero", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, time.Hour)
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", 0, "failed"))

		got, err := repo.ClaimDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []storagecleanup.Deletion{{Key: "a.jpg", Attempts: 1}}, got)
	})

	t.Run("leaves other deletions alone", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0, -time.Minute)
		queue(t, "b.jpg", 0, -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "failed"))

		state := stateOf(t, "b.jpg")
		assert.Zero(t, state.Attempts)
		assert.Nil(t, state.LastError)
	})

	t.Run("is not an error for a key that is not queued, and queues nothing", func(t *testing.T) {
		repo := newRepository(t)

		require.NoError(t, repo.Reschedule(ctx, "missing.jpg", time.Hour, "failed"))

		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})
}
