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

	"sakukota-go/internal/module/storagecleanup"
	"sakukota-go/internal/testutil/pgtest"
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
func newRepository(t *testing.T) *storagecleanup.PostgresRepository {
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
	require.NoError(t, err, "queue %q", key)
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
	require.NoError(t, err, "read queue state of %q", key)

	return s
}

func TestRepository_Remove(t *testing.T) {
	ctx := t.Context()

	t.Run("RemovesKeyAndLeavesOthers", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", 0)
		queue(t, "b.jpg", 0)

		require.NoError(t, repo.Remove(ctx, "a.jpg"))

		got := pgtest.QueuedKeys(t, testDB)
		assert.Equal(t, []string{"b.jpg"}, got)
	})

	t.Run("MissingKey", func(t *testing.T) {
		repo := newRepository(t)

		assert.NoError(t, repo.Remove(ctx, "missing.jpg"))
	})
}

func TestRepository_ListDue(t *testing.T) {
	ctx := t.Context()

	t.Run("LongestDueFirst", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "newer.jpg", -time.Minute)
		queue(t, "older.jpg", -time.Hour)
		queue(t, "oldest.jpg", -2*time.Hour)

		got, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"oldest.jpg", "older.jpg", "newer.jpg"}, got)
	})

	t.Run("TiesOrderedByKey", func(t *testing.T) {
		repo := newRepository(t)
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key, next_attempt_at)
			VALUES ('b.jpg', '2026-01-01T00:00:00Z'), ('a.jpg', '2026-01-01T00:00:00Z')`)
		require.NoError(t, err, "insert queue rows")

		got, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, got)
	})

	t.Run("SkipsNotYetDue", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "due.jpg", -time.Minute)
		queue(t, "later.jpg", time.Hour)

		got, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		assert.Equal(t, []string{"due.jpg"}, got)
	})

	t.Run("AtMostLimit", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "c.jpg", -time.Minute)
		queue(t, "a.jpg", -3*time.Hour)
		queue(t, "b.jpg", -2*time.Hour)

		got, err := repo.ListDue(ctx, 2)
		require.NoError(t, err)
		assert.Equal(t, []string{"a.jpg", "b.jpg"}, got)
	})

	t.Run("LeavesKeysDue", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		first, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		second, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)

		assert.Equal(t, []string{"a.jpg"}, first)
		assert.Equal(t, first, second)
	})

	t.Run("NothingDue", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestRepository_Reschedule(t *testing.T) {
	ctx := t.Context()

	t.Run("RecordsAttemptAndDelay", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", 30*time.Minute, "storage unavailable"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 1, state.Attempts)
		require.NotNil(t, state.LastError)
		assert.Equal(t, "storage unavailable", *state.LastError)
		assert.InDelta(t, (30 * time.Minute).Seconds(), state.DueIn, 30)
	})

	t.Run("CountsFailuresKeepsLatestError", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Minute, "first"))
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "second"))

		state := stateOf(t, "a.jpg")
		assert.Equal(t, 2, state.Attempts)
		require.NotNil(t, state.LastError)
		assert.Equal(t, "second", *state.LastError)
		assert.InDelta(t, time.Hour.Seconds(), state.DueIn, 30)
	})

	t.Run("NotDueUntilDelayPasses", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)
		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "failed"))

		got, err := repo.ListDue(ctx, 10)
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("LeavesOtherDeletions", func(t *testing.T) {
		repo := newRepository(t)
		queue(t, "a.jpg", -time.Minute)
		queue(t, "b.jpg", -time.Minute)

		require.NoError(t, repo.Reschedule(ctx, "a.jpg", time.Hour, "failed"))

		state := stateOf(t, "b.jpg")
		assert.Equal(t, 0, state.Attempts)
		assert.Nil(t, state.LastError)
	})

	t.Run("MissingKeyQueuesNothing", func(t *testing.T) {
		repo := newRepository(t)

		require.NoError(t, repo.Reschedule(ctx, "missing.jpg", time.Hour, "failed"))

		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})
}
