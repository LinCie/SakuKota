//go:build integration

package reviews_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/database"
	"rantau-go/internal/domain"
	"rantau-go/internal/module/reviews"
	"rantau-go/internal/testutil/pgtest"
)

// checkViolation is the PostgreSQL error code for CHECK constraint violations.
const checkViolation = "23514"

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
func newRepository(t *testing.T) *reviews.PostgresRepository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return reviews.NewPostgresRepository(testDB)
}

// insertPlace inserts a place and returns its id.
func insertPlace(t *testing.T, name string) int64 {
	t.Helper()

	subDistrictID := pgtest.InsertSubDistrict(t, testDB, "Setiabudi "+name)
	return pgtest.InsertPlace(t, testDB, subDistrictID, pgtest.CategoryID(t, testDB, "laundry"), name)
}

// ratingTotals returns the place's rating count and sum.
func ratingTotals(t *testing.T, placeID int64) [2]int {
	t.Helper()

	var totals struct {
		Count int `db:"rating_count"`
		Sum   int `db:"rating_sum"`
	}
	err := testDB.Get(&totals, `SELECT rating_count, rating_sum FROM PLACES WHERE id = $1`, placeID)
	require.NoError(t, err, "read rating totals of place %d", placeID)

	return [2]int{totals.Count, totals.Sum}
}

// mustCreate creates the review and returns it, failing the test on error.
func mustCreate(t *testing.T, repo *reviews.PostgresRepository, r *domain.Review) *domain.Review {
	t.Helper()

	got, err := repo.Create(t.Context(), r)
	require.NoError(t, err, "Create(%+v)", *r)
	return got
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Run("stores_review_with_author", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		userID := pgtest.InsertUser(t, testDB, "Ana")
		comment := "Bersih dan cepat"

		got, err := repo.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: 4, Comment: &comment})
		require.NoError(t, err)

		assert.NotZero(t, got.ID, "Create().ID")
		assert.Equal(t, placeID, got.PlaceID)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, 4, got.Rating)
		assert.Equal(t, &comment, got.Comment)
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute, "Create().CreatedAt")
		assert.Equal(t, domain.Author{ID: userID, Name: "Ana"}, got.User)
	})

	t.Run("stores_review_without_comment", func(t *testing.T) {
		repo := newRepository(t)

		got := mustCreate(t, repo, &domain.Review{
			PlaceID: insertPlace(t, "Laundry"),
			UserID:  pgtest.InsertUser(t, testDB, "Ana"),
			Rating:  5,
		})

		assert.Nil(t, got.Comment)
	})

	t.Run("second_review_by_user", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		userID := pgtest.InsertUser(t, testDB, "Ana")
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: userID, Rating: 5})

		got, err := repo.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: 1})

		assert.ErrorIs(t, err, domain.ErrPlaceAlreadyReviewed)
		assert.Nil(t, got)
	})

	t.Run("user_reviews_different_places", func(t *testing.T) {
		repo := newRepository(t)
		userID := pgtest.InsertUser(t, testDB, "Ana")

		mustCreate(t, repo, &domain.Review{PlaceID: insertPlace(t, "First"), UserID: userID, Rating: 5})
		_, err := repo.Create(t.Context(), &domain.Review{PlaceID: insertPlace(t, "Second"), UserID: userID, Rating: 2})

		assert.NoError(t, err, "Create() for a second place")
	})

	t.Run("unknown_place", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(t.Context(), &domain.Review{PlaceID: 999, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 5})

		assert.ErrorIs(t, err, domain.ErrPlaceNotFound)
		assert.Nil(t, got)
	})

	t.Run("unknown_user", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")

		got, err := repo.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: 999, Rating: 5})

		assert.True(t, database.IsCode(err, database.ForeignKeyViolation), "Create() error = %v, want a foreign key violation", err)
		assert.Nil(t, got)
	})

	t.Run("rating_outside_range", func(t *testing.T) {
		for _, rating := range []int{0, 6} {
			t.Run(fmt.Sprint(rating), func(t *testing.T) {
				repo := newRepository(t)
				placeID := insertPlace(t, "Laundry")

				got, err := repo.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: rating})

				assert.True(t, database.IsCode(err, checkViolation), "Create(rating %d) error = %v, want a check violation", rating, err)
				assert.Nil(t, got, "Create(rating %d)", rating)
			})
		}
	})
}

func TestPostgresRepository_ListByPlace(t *testing.T) {
	t.Run("newest_first_with_authors", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		otherPlaceID := insertPlace(t, "Other")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		budiID := pgtest.InsertUser(t, testDB, "Budi")
		citraID := pgtest.InsertUser(t, testDB, "Citra")

		oldest := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: anaID, Rating: 5})
		newest := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: budiID, Rating: 2})
		mustCreate(t, repo, &domain.Review{PlaceID: otherPlaceID, UserID: citraID, Rating: 3})
		_, err := testDB.Exec(`UPDATE PLACE_REVIEWS SET created_at = NOW() - INTERVAL '1 day' WHERE id = $1`, oldest.ID)
		require.NoError(t, err, "backdate review")

		got, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)

		require.Len(t, got, 2)
		assert.Equal(t, newest.ID, got[0].ID)
		assert.Equal(t, domain.Author{ID: budiID, Name: "Budi"}, got[0].User)
		assert.Equal(t, 2, got[0].Rating)
		assert.Equal(t, oldest.ID, got[1].ID)
		assert.Equal(t, domain.Author{ID: anaID, Name: "Ana"}, got[1].User)
	})

	t.Run("simultaneous_reviews_later_first", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		first := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 5})
		second := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Budi"), Rating: 4})
		_, err := testDB.Exec(`UPDATE PLACE_REVIEWS SET created_at = '2026-01-01T00:00:00Z'`)
		require.NoError(t, err, "equalize review times")

		got, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)

		require.Len(t, got, 2)
		assert.Equal(t, []int64{second.ID, first.ID}, []int64{got[0].ID, got[1].ID})
	})

	t.Run("no_reviews", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")

		got, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)

		assert.NotNil(t, got, "want an empty, non-nil slice")
		assert.Empty(t, got)
	})

	t.Run("unknown_place", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.ListByPlace(t.Context(), 999)

		assert.ErrorIs(t, err, domain.ErrPlaceNotFound)
		assert.Nil(t, got)
	})
}
