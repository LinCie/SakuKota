//go:build integration

package users_test

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

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/users"
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

// avatar is a promoted profile photo stored under key.
func avatar(key string) domain.Media {
	return domain.Media{Key: key, ContentType: "image/webp", Size: 1024}
}

func TestRepository_Rename(t *testing.T) {
	t.Run("SetsName", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")
		budi := createUser(t, repo, "Budi", "budi@example.com", "h")

		got, err := repo.Rename(t.Context(), ana.ID, "Ana Lestari")
		require.NoError(t, err)
		assert.Equal(t, "Ana Lestari", got.Name)
		assert.Equal(t, ana.Email, got.Email)
		assert.False(t, got.UpdatedAt.Before(ana.UpdatedAt), "UpdatedAt moved back")

		other, err := repo.Fetch(t.Context(), budi.ID)
		require.NoError(t, err)
		assert.Equal(t, budi, other, "another user changed")
	})

	t.Run("NotFound", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Rename(t.Context(), 999, "Ana")

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_SetAvatar(t *testing.T) {
	t.Run("FirstPhoto", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")

		got, err := repo.SetAvatar(t.Context(), ana.ID, avatar("avatars/1/a.webp"))
		require.NoError(t, err)
		assert.Equal(t, new("avatars/1/a.webp"), got.AvatarKey)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))

		byEmail, err := repo.FetchByEmail(t.Context(), ana.Email)
		require.NoError(t, err)
		assert.Equal(t, got, byEmail)
	})

	t.Run("ReplacesAndQueuesOldPhoto", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")
		_, err := repo.SetAvatar(t.Context(), ana.ID, avatar("avatars/1/a.webp"))
		require.NoError(t, err)

		got, err := repo.SetAvatar(t.Context(), ana.ID, avatar("avatars/1/b.webp"))
		require.NoError(t, err)
		assert.Equal(t, new("avatars/1/b.webp"), got.AvatarKey)
		assert.Equal(t, []string{"avatars/1/a.webp"}, pgtest.QueuedKeys(t, testDB))

		var medias int
		require.NoError(t, testDB.Get(&medias, `SELECT COUNT(*) FROM MEDIAS`))
		assert.Equal(t, 1, medias, "the old MEDIAS row must be deleted")
	})

	t.Run("ConcurrentChangesQueueEveryReplacedPhoto", func(t *testing.T) {
		repo := newRepository(t)
		ana := createUser(t, repo, "Ana", "ana@example.com", "h")

		const n = 5
		var wg sync.WaitGroup
		for i := range n {
			wg.Go(func() {
				_, err := repo.SetAvatar(t.Context(), ana.ID, avatar(fmt.Sprintf("avatars/1/%d.webp", i)))
				assert.NoError(t, err)
			})
		}
		wg.Wait()

		var medias int
		require.NoError(t, testDB.Get(&medias, `SELECT COUNT(*) FROM MEDIAS`))
		assert.Equal(t, 1, medias, "only the current photo stays")
		assert.Len(t, pgtest.QueuedKeys(t, testDB), n-1)
	})

	t.Run("NotFound", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.SetAvatar(t.Context(), 999, avatar("avatars/999/a.webp"))

		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, got)
		var medias int
		require.NoError(t, testDB.Get(&medias, `SELECT COUNT(*) FROM MEDIAS`))
		assert.Zero(t, medias, "nothing may be stored")
	})
}
