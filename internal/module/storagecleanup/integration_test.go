//go:build integration

package storagecleanup_test

import (
	"context"
	"log"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"rantau-go/internal/database"
	"rantau-go/internal/module/storagecleanup"
)

// testDB points at the PostGIS container started by TestMain.
var testDB *sqlx.DB

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Same stack as docker/postgres: PostgreSQL 17 with PostGIS 3.
	container, err := postgres.Run(ctx, "postgis/postgis:17-3.5",
		postgres.WithDatabase("rantaugo"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("start postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	testDB, err = database.NewDB(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, testDB.DB, "../../../migrations"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	code := m.Run()

	testDB.Close()
	if err := container.Terminate(ctx); err != nil {
		log.Printf("terminate container: %v", err)
	}
	os.Exit(code)
}

// -----------------------------------------------------------------------------
// Fixtures and helpers
// -----------------------------------------------------------------------------

func mustExec(t *testing.T, query string) {
	t.Helper()
	if _, err := testDB.Exec(query); err != nil {
		t.Fatalf("exec failed: %v\nquery: %s", err, query)
	}
}

// seedQueue empties the queue and fills it with, in order of when they are due:
//
//	places/1/a.webp  due 2 hours ago   0 attempts
//	places/2/b.webp  due 1 hour ago    2 attempts
//	places/3/c.webp  due 1 minute ago  0 attempts
//	places/4/d.webp  due in 1 hour    1 attempt
func seedQueue(t *testing.T) {
	t.Helper()
	mustExec(t, `TRUNCATE STORAGE_DELETIONS`)
	mustExec(t, `INSERT INTO STORAGE_DELETIONS (key, attempts, next_attempt_at) VALUES
		('places/1/a.webp', 0, NOW() - INTERVAL '2 hours'),
		('places/2/b.webp', 2, NOW() - INTERVAL '1 hour'),
		('places/3/c.webp', 0, NOW() - INTERVAL '1 minute'),
		('places/4/d.webp', 1, NOW() + INTERVAL '1 hour')`)
}

// queuedRow is a row of the queue, with the time until it is due.
type queuedRow struct {
	Key       string  `db:"key"`
	Attempts  int     `db:"attempts"`
	LastError *string `db:"last_error"`
	DueIn     float64 `db:"due_in"`
}

// queuedRows returns the rows of the queue ordered by key. DueIn is in seconds.
func queuedRows(t *testing.T) []queuedRow {
	t.Helper()
	rows := []queuedRow{}
	err := testDB.Select(&rows, `SELECT key, attempts, last_error,
		EXTRACT(EPOCH FROM next_attempt_at - NOW())::float8 AS due_in
		FROM STORAGE_DELETIONS ORDER BY key`)
	if err != nil {
		t.Fatalf("select queue: %v", err)
	}
	return rows
}

func queuedKeys(t *testing.T) []string {
	t.Helper()
	keys := []string{}
	for _, r := range queuedRows(t) {
		keys = append(keys, r.Key)
	}
	return keys
}

// -----------------------------------------------------------------------------
// postgresRepository
// -----------------------------------------------------------------------------

func TestIntegration_ClaimDue(t *testing.T) {
	seedQueue(t)
	repo := storagecleanup.NewPostgresRepository(testDB)
	ctx := context.Background()

	got, err := repo.ClaimDue(ctx, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []storagecleanup.Deletion{
		{Key: "places/1/a.webp", Attempts: 0},
		{Key: "places/2/b.webp", Attempts: 2},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("first claim = %+v, want the two longest due %+v", got, want)
	}

	got, err = repo.ClaimDue(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = []storagecleanup.Deletion{{Key: "places/3/c.webp", Attempts: 0}}
	if !slices.Equal(got, want) {
		t.Fatalf("second claim = %+v, want the remaining due deletion %+v", got, want)
	}

	// Claimed deletions are held back, and the last one is not due yet.
	got, err = repo.ClaimDue(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("third claim = %+v, want nothing", got)
	}

	// Claiming does not take deletions off the queue.
	wantKeys := []string{"places/1/a.webp", "places/2/b.webp", "places/3/c.webp", "places/4/d.webp"}
	if keys := queuedKeys(t); !slices.Equal(keys, wantKeys) {
		t.Errorf("queued keys = %v, want %v", keys, wantKeys)
	}
}

func TestIntegration_Remove(t *testing.T) {
	seedQueue(t)
	repo := storagecleanup.NewPostgresRepository(testDB)
	ctx := context.Background()

	if err := repo.Remove(ctx, "places/2/b.webp"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := repo.Remove(ctx, "places/99/missing.webp"); err != nil {
		t.Fatalf("removing a missing key: err = %v, want nil", err)
	}

	want := []string{"places/1/a.webp", "places/3/c.webp", "places/4/d.webp"}
	if keys := queuedKeys(t); !slices.Equal(keys, want) {
		t.Errorf("queued keys = %v, want %v", keys, want)
	}
}

func TestIntegration_Reschedule(t *testing.T) {
	seedQueue(t)
	repo := storagecleanup.NewPostgresRepository(testDB)
	ctx := context.Background()

	if err := repo.Reschedule(ctx, "places/2/b.webp", time.Hour, "connection reset"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var b queuedRow
	for _, r := range queuedRows(t) {
		if r.Key == "places/2/b.webp" {
			b = r
		}
	}
	if b.Attempts != 3 {
		t.Errorf("attempts = %d, want 3", b.Attempts)
	}
	if b.LastError == nil || *b.LastError != "connection reset" {
		t.Errorf("last_error = %v, want %q", b.LastError, "connection reset")
	}
	if b.DueIn < (59*time.Minute).Seconds() || b.DueIn > (61*time.Minute).Seconds() {
		t.Errorf("due in %.0fs, want about an hour", b.DueIn)
	}

	// The rescheduled deletion is not due, so only a and c are claimed.
	got, err := repo.ClaimDue(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []storagecleanup.Deletion{
		{Key: "places/1/a.webp", Attempts: 0},
		{Key: "places/3/c.webp", Attempts: 0},
	}
	if !slices.Equal(got, want) {
		t.Errorf("claim after rescheduling = %+v, want %+v", got, want)
	}

	// A claimed deletion becomes due again when it is rescheduled without a delay.
	if err := repo.Reschedule(ctx, "places/1/a.webp", 0, "timeout"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, err = repo.ClaimDue(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want = []storagecleanup.Deletion{{Key: "places/1/a.webp", Attempts: 1}}
	if !slices.Equal(got, want) {
		t.Errorf("claim after rescheduling without a delay = %+v, want %+v", got, want)
	}
}
