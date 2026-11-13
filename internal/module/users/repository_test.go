//go:build integration

package users_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/users"
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
func newRepository(t *testing.T) users.Repository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return users.NewPostgresRepository(testDB)
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the user and returns its generated fields", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "$2a$hash"})

		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, "Ana", got.Name)
		assert.Equal(t, "ana@example.com", got.Email)
		assert.Equal(t, "$2a$hash", got.PasswordHash)
		assert.Equal(t, "user", got.Role, "new users are not admins")
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.WithinDuration(t, time.Now(), got.UpdatedAt, time.Minute)

		stored, err := repo.GetUser(ctx, got.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("gives each user its own id", func(t *testing.T) {
		repo := newRepository(t)

		first, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "h"})
		require.NoError(t, err)
		second, err := repo.Create(ctx, &users.User{Name: "Budi", Email: "budi@example.com", PasswordHash: "h"})
		require.NoError(t, err)

		assert.Less(t, first.ID, second.ID)
	})

	t.Run("returns ErrEmailTaken for a registered email", func(t *testing.T) {
		repo := newRepository(t)
		_, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "h"})
		require.NoError(t, err)

		got, err := repo.Create(ctx, &users.User{Name: "Other Ana", Email: "ana@example.com", PasswordHash: "h2"})

		assert.ErrorIs(t, err, users.ErrEmailTaken)
		assert.Nil(t, got)
	})
}

func TestRepository_GetByEmail(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the user with the email", func(t *testing.T) {
		repo := newRepository(t)
		_, err := repo.Create(ctx, &users.User{Name: "Budi", Email: "budi@example.com", PasswordHash: "h"})
		require.NoError(t, err)
		ana, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "h"})
		require.NoError(t, err)

		got, err := repo.GetByEmail(ctx, "ana@example.com")

		require.NoError(t, err)
		assert.Equal(t, ana, got)
	})

	t.Run("returns ErrNotFound for an unknown email", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetByEmail(ctx, "nobody@example.com")

		assert.ErrorIs(t, err, users.ErrNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_GetUser(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the user with the id", func(t *testing.T) {
		repo := newRepository(t)
		ana, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "h"})
		require.NoError(t, err)

		got, err := repo.GetUser(ctx, ana.ID)

		require.NoError(t, err)
		assert.Equal(t, ana, got)
	})

	t.Run("reads the role of an admin", func(t *testing.T) {
		repo := newRepository(t)
		ana, err := repo.Create(ctx, &users.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "h"})
		require.NoError(t, err)
		_, err = testDB.Exec(`UPDATE USERS SET role = $2 WHERE id = $1`, ana.ID, users.RoleAdmin)
		require.NoError(t, err)

		got, err := repo.GetUser(ctx, ana.ID)

		require.NoError(t, err)
		assert.Equal(t, users.RoleAdmin, got.Role)
	})

	t.Run("returns ErrNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetUser(ctx, 999)

		assert.ErrorIs(t, err, users.ErrNotFound)
		assert.Nil(t, got)
	})
}
