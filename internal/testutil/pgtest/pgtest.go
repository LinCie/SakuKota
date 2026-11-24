// Package pgtest starts a throwaway PostgreSQL+PostGIS container with the
// project's migrations applied, for repository tests.
package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	_ "github.com/jackc/pgx/v5/stdlib"

	"sakukota-go/internal/database"
)

// image provides PostgreSQL with PostGIS, like docker/postgres/Dockerfile.
const image = "postgis/postgis:17-3.5"

// keptTables hold data that migrations or extensions insert. Reset leaves them
// alone.
var keptTables = []string{"goose_db_version", "categories", "category_fields", "spatial_ref_sys"}

// Start runs a PostgreSQL container with the migrations applied, and returns a connection.
// Call stop to remove it. Start one per test package, not per test, since starting is slow.
func Start(ctx context.Context) (db *sqlx.DB, stop func(), err error) {
	container, err := postgres.Run(ctx, image,
		postgres.WithDatabase("sakukotago_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("pgtest: start container: %w", err)
	}

	stop = func() {
		if db != nil {
			_ = db.Close()
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

	url, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, fmt.Errorf("pgtest: connection string: %w", err)
	}

	if err := migrate(ctx, url); err != nil {
		return nil, nil, err
	}

	db, err = database.NewPostgresDB(ctx, url)
	if err != nil {
		return nil, nil, fmt.Errorf("pgtest: connect: %w", err)
	}

	return db, stop, nil
}

// migrate applies every migration in migrations/ to the database at url.
func migrate(ctx context.Context, url string) error {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("pgtest: open: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("pgtest: goose dialect: %w", err)
	}
	goose.SetLogger(goose.NopLogger())

	if err := goose.UpContext(ctx, sqlDB, migrationsDir()); err != nil {
		return fmt.Errorf("pgtest: migrate: %w", err)
	}

	return nil
}

// migrationsDir finds migrations/ at the repository root from this file's
// location, so it works whichever package runs the tests.
func migrationsDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "migrations")
}

// Reset empties every table the migrations do not keep and restarts their ids,
// so each test starts from the categories and fields the migrations define.
func Reset(t testing.TB, db *sqlx.DB) {
	t.Helper()

	var tables []string
	err := db.Select(&tables, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' AND tablename <> ALL($1)`, keptTables)
	require.NoError(t, err, "list tables")
	if len(tables) == 0 {
		return
	}

	_, err = db.Exec(`TRUNCATE ` + strings.Join(tables, ", ") + ` RESTART IDENTITY CASCADE`)
	require.NoError(t, err, "truncate tables")
}
