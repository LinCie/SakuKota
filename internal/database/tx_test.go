//go:build integration

package database_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/database"
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

// insertUser inserts a user named name through database.Conn(ctx, testDB).
func insertUser(ctx context.Context, name string) error {
	_, err := database.Conn(ctx, testDB).ExecContext(ctx,
		`INSERT INTO USERS (name, email, password_hash) VALUES ($1, $2, 'hash')`, name, name+"@example.com")
	return err
}

// userNames returns the stored user names in id order.
func userNames(t *testing.T) []string {
	t.Helper()

	names := []string{}
	require.NoError(t, testDB.Select(&names, `SELECT name FROM USERS ORDER BY id`), "select users")
	return names
}

func TestTransactor_WithinTx(t *testing.T) {
	boom := errors.New("boom")

	t.Run("without_tx_conn_uses_db", func(t *testing.T) {
		pgtest.Reset(t, testDB)

		require.NoError(t, insertUser(t.Context(), "ana"))

		assert.Equal(t, []string{"ana"}, userNames(t))
	})

	t.Run("commits_when_fn_succeeds", func(t *testing.T) {
		pgtest.Reset(t, testDB)

		err := database.NewTransactor(testDB).WithinTx(t.Context(), func(ctx context.Context) error {
			if err := insertUser(ctx, "ana"); err != nil {
				return err
			}
			return insertUser(ctx, "budi")
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"ana", "budi"}, userNames(t))
	})

	t.Run("rolls_back_and_returns_fn_error", func(t *testing.T) {
		pgtest.Reset(t, testDB)

		err := database.NewTransactor(testDB).WithinTx(t.Context(), func(ctx context.Context) error {
			if err := insertUser(ctx, "ana"); err != nil {
				return err
			}
			return boom
		})

		assert.ErrorIs(t, err, boom)
		assert.Empty(t, userNames(t))
	})

	t.Run("nested_call_joins_outer_tx", func(t *testing.T) {
		pgtest.Reset(t, testDB)
		transactor := database.NewTransactor(testDB)

		err := transactor.WithinTx(t.Context(), func(ctx context.Context) error {
			err := transactor.WithinTx(ctx, func(ctx context.Context) error {
				return insertUser(ctx, "ana")
			})
			if err != nil {
				return err
			}
			return boom
		})

		assert.ErrorIs(t, err, boom)
		assert.Empty(t, userNames(t), "the inner write outlived the outer rollback")
	})
}
