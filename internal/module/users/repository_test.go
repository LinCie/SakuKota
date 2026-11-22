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

	"rantau-go/internal/domain"
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
func newRepository(t *testing.T) *users.PostgresRepository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return users.NewPostgresRepository(testDB)
}

// createUser stores a user and returns it.
func createUser(t *testing.T, repo *users.PostgresRepository, name, email, hash string) *domain.User {
	t.Helper()
	u, err := repo.Create(t.Context(), &domain.User{Name: name, Email: email, PasswordHash: hash})
	require.NoError(t, err, "Create(%q)", email)
	return u
}

func TestRepository_Create(t *testing.T) {
	t.Run("StoresUserAndReturnsGeneratedFields", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(t.Context(), &domain.User{Name: "Ana", Email: "ana@example.com", PasswordHash: "$2a$hash"})
		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, "Ana", got.Name)
		assert.Equal(t, "ana@example.com", got.Email)
		assert.Equal(t, "$2a$hash", got.PasswordHash)
		assert.Equal(t, "user", got.Role) // new users are not admins
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.WithinDuration(t, time.Now(), got.UpdatedAt, time.Minute)

		stored, err := repo.Fetch(t.Context(), got.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("GivesEachUserItsOwnID", func(t *testing.T) {
		repo := newRepository(t)

		first := createUser(t, repo, "Ana", "ana@example.com", "h")
		second := createUser(t, repo, "Budi", "budi@example.com", "h")

		assert.Less(t, first.ID, second.ID)
	})

	t.Run("EmailTaken", func(t *testing.T) {
		repo := newRepository(t)
		createUser(t, repo, "Ana", "ana@example.com", "h")

		got, err := repo.Create(t.Context(), &domain.User{Name: "Other Ana", Email: "ana@example.com", PasswordHash: "h2"})

		assert.ErrorIs(t, err, domain.ErrEmailTaken)
		assert.Nil(t, got)
	})
}

func TestRepository_FetchByEmail(t *testing.T) {
	t.Run("ReturnsUserWithEmail", func(t *testing.T) {
		repo := newRepository(t)
		createUser(t, repo, "Budi", "budi@example.com", "h")
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")

		got, err := repo.FetchByEmail(t.Context(), "ana@example.com")
		require.NoError(t, err)
		assert.Equal(t, ana, got)
	})

	t.Run("NotFound", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.FetchByEmail(t.Context(), "nobody@example.com")

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_Fetch(t *testing.T) {
	t.Run("ReturnsUserWithID", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")

		got, err := repo.Fetch(t.Context(), ana.ID)
		require.NoError(t, err)
		assert.Equal(t, ana, got)
	})

	t.Run("ReadsAdminRole", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")
		_, err := testDB.Exec(`UPDATE USERS SET role = $2 WHERE id = $1`, ana.ID, domain.RoleAdmin)
		require.NoError(t, err, "promote user %d", ana.ID)

		got, err := repo.Fetch(t.Context(), ana.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.RoleAdmin, got.Role)
	})

	t.Run("NotFound", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Fetch(t.Context(), 999)

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
	})
}
