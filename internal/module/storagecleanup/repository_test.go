//go:build integration

package storagecleanup_test

import (
	"context"
	"fmt"
	"os"
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

// queue adds key to the queue, due dueIn from now (negative for already due).
func queue(t *testing.T, key string, dueIn time.Duration) {
	t.Helper()

	_, err := testDB.Exec(
		`INSERT INTO STORAGE_DELETIONS (key, next_attempt_at)
		VALUES ($1, NOW() + make_interval(secs => $2))`,
		key, dueIn.Seconds(),
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

func TestRepository_Remove(t *testing.T) {
	ctx := context.Background()

	t.Run("takes the key off the queue and leaves the others", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0)
		queue(t, "b.jpg", 0)

		require.NoError(t, repo.Remove(ctx, "a.jpg"))

		assert.Equal(t, []string{"b.jpg"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("is not an error for a key that is not queued", func(t *testing.T) {
		repo := newRepository(t)

		assert.NoError(t, repo.Remove(ctx, "missing.jpg"))
	})
}

func TestRepository_GetDue(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the due keys, longest due first", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "newer.jpg", -time.Minute)
		queue(t, "older.jpg", -time.Hour)
		queue(t, "oldest.jpg", -2*time.Hour)

		got, err := repo.GetDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []string{"oldest.jpg", "older.jpg", "newer.jpg"}, got)
	})

	t.Run("orders keys that became due together by key", func(t *testing.T) {
		repo := newRepository(t)
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key, next_attempt_at)
			VALUES ('b.jpg', '2026-01-01T00:00:00Z'), ('a.jpg', '2026-01-01T00:00:00Z')`)
		require.NoError(t, err)

		got, err := repo.GetDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, got)
	})

	t.Run("skips keys that are not due yet", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "due.jpg", -time.Minute)
		queue(t, "later.jpg", time.Hour)

		got, err := repo.GetDue(ctx, 10)

		require.NoError(t, err)
		assert.Equal(t, []string{"due.jpg"}, got)
	})

	t.Run("returns at most limit, the longest due first", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "c.jpg", -time.Minute)
		queue(t, "a.jpg", -3*time.Hour)
		queue(t, "b.jpg", -2*time.Hour)

		got, err := repo.GetDue(ctx, 2)

		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, got)
	})

	t.Run("leaves the returned keys due", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		first, err := repo.GetDue(ctx, 10)
		require.NoError(t, err)
		second, err := repo.GetDue(ctx, 10)
		require.NoError(t, err)

		assert.Equal(t, []string{"a.jpg"}, first)
		assert.Equal(t, first, second)
	})

	t.Run("returns nothing when nothing is due", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetDue(ctx, 10)

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestRepository_Reschedule(t *testing.T) {
	ctx := context.Background()

	t.Run("records the failed attempt and makes the deletion due after the delay", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", 30*time.Minute, "storage unavailable"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 1, state.Attempts)
		assert.Equal(t, "storage unavailable", *state.LastError)
		assert.InDelta(t, (30 * time.Minute).Seconds(), state.DueIn, 30)
	})

	t.Run("counts each failure and keeps the latest error", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Minute, "first"))
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "second"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 2, state.Attempts)
		assert.Equal(t, "second", *state.LastError)
		assert.InDelta(t, time.Hour.Seconds(), state.DueIn, 30)
	})

	t.Run("keeps the deletion from being due until the delay passes", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "failed"))

		got, err := repo.GetDue(ctx, 10)

		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("leaves other deletions alone", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)
		queue(t, "b.jpg", -time.Minute)

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
