//go:build integration

package reviews_test

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

	"rantau-go/internal/database"
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
func newRepository(t *testing.T) reviews.Repository {
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
func ratingTotals(t *testing.T, placeID int64) (count, sum int) {
	t.Helper()

	var totals struct {
		Count int `db:"rating_count"`
		Sum   int `db:"rating_sum"`
	}
	require.NoError(t, testDB.Get(&totals, `SELECT rating_count, rating_sum FROM PLACES WHERE id = $1`, placeID))

	return totals.Count, totals.Sum
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the review and returns it with its author", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		userID := pgtest.InsertUser(t, testDB, "Ana")
		comment := "Bersih dan cepat"

		got, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: userID, Rating: 4, Comment: &comment})

		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, placeID, got.PlaceID)
		assert.Equal(t, userID, got.UserID)
		assert.Equal(t, 4, got.Rating)
		assert.Equal(t, &comment, got.Comment)
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.Equal(t, reviews.Author{ID: userID, Name: "Ana"}, got.User)
	})

	t.Run("stores a review without a comment", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, &reviews.Review{
			PlaceID: insertPlace(t, "Laundry"),
			UserID:  pgtest.InsertUser(t, testDB, "Ana"),
			Rating:  5,
		})

		require.NoError(t, err)
		assert.Nil(t, got.Comment)
	})

	t.Run("adds each rating to the place's totals", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		otherPlaceID := insertPlace(t, "Other")

		for i, rating := range []int{5, 3, 1} {
			userID := pgtest.InsertUser(t, testDB, fmt.Sprintf("User%d", i))
			_, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: userID, Rating: rating})
			require.NoError(t, err)
		}

		count, sum := ratingTotals(t, placeID)
		assert.Equal(t, 3, count)
		assert.Equal(t, 9, sum)
		count, sum = ratingTotals(t, otherPlaceID)
		assert.Zero(t, count, "other places are untouched")
		assert.Zero(t, sum)
	})

	t.Run("loses no rating when users review the same place at once", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		const reviewers = 20

		userIDs := make([]int64, reviewers)
		for i := range userIDs {
			userIDs[i] = pgtest.InsertUser(t, testDB, fmt.Sprintf("User%d", i))
		}

		var wg sync.WaitGroup
		errs := make([]error, reviewers)
		for i, userID := range userIDs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: userID, Rating: 4})
			}()
		}
		wg.Wait()

		for _, err := range errs {
			require.NoError(t, err)
		}
		count, sum := ratingTotals(t, placeID)
		assert.Equal(t, reviewers, count)
		assert.Equal(t, reviewers*4, sum)
	})

	t.Run("returns ErrAlreadyReviewed for a second review by the same user and keeps the totals", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		userID := pgtest.InsertUser(t, testDB, "Ana")
		_, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: userID, Rating: 5})
		require.NoError(t, err)

		got, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: userID, Rating: 1})

		assert.ErrorIs(t, err, reviews.ErrAlreadyReviewed)
		assert.Nil(t, got)
		count, sum := ratingTotals(t, placeID)
		assert.Equal(t, 1, count)
		assert.Equal(t, 5, sum)
	})

	t.Run("lets a user review different places", func(t *testing.T) {
		repo := newRepository(t)
		userID := pgtest.InsertUser(t, testDB, "Ana")

		_, err := repo.Create(ctx, &reviews.Review{PlaceID: insertPlace(t, "First"), UserID: userID, Rating: 5})
		require.NoError(t, err)
		_, err = repo.Create(ctx, &reviews.Review{PlaceID: insertPlace(t, "Second"), UserID: userID, Rating: 2})

		assert.NoError(t, err)
	})

	t.Run("returns ErrPlaceNotFound for an unknown place", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, &reviews.Review{PlaceID: 999, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 5})

		assert.ErrorIs(t, err, reviews.ErrPlaceNotFound)
		assert.Nil(t, got)
	})

	t.Run("fails for an unknown user and keeps the totals", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")

		got, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: 999, Rating: 5})

		assert.True(t, database.IsCode(err, database.ForeignKeyViolation), "got %v", err)
		assert.Nil(t, got)
		count, _ := ratingTotals(t, placeID)
		assert.Zero(t, count)
	})

	t.Run("rejects a rating outside 1 to 5 and keeps the totals", func(t *testing.T) {
		for _, rating := range []int{0, 6} {
			t.Run(fmt.Sprint(rating), func(t *testing.T) {
				repo := newRepository(t)
				placeID := insertPlace(t, "Laundry")

				got, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: rating})

				assert.True(t, database.IsCode(err, checkViolation), "got %v", err)
				assert.Nil(t, got)
				count, _ := ratingTotals(t, placeID)
				assert.Zero(t, count)
			})
		}
	})
}

func TestRepository_GetByPlace(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the place's reviews with their authors, newest first", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		otherPlaceID := insertPlace(t, "Other")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		budiID := pgtest.InsertUser(t, testDB, "Budi")
		citraID := pgtest.InsertUser(t, testDB, "Citra")

		oldest, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: anaID, Rating: 5})
		require.NoError(t, err)
		newest, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: budiID, Rating: 2})
		require.NoError(t, err)
		_, err = repo.Create(ctx, &reviews.Review{PlaceID: otherPlaceID, UserID: citraID, Rating: 3})
		require.NoError(t, err)
		_, err = testDB.Exec(`UPDATE PLACE_REVIEWS SET created_at = NOW() - INTERVAL '1 day' WHERE id = $1`, oldest.ID)
		require.NoError(t, err)

		got, err := repo.GetByPlace(ctx, placeID)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, newest.ID, got[0].ID)
		assert.Equal(t, reviews.Author{ID: budiID, Name: "Budi"}, got[0].User)
		assert.Equal(t, 2, got[0].Rating)
		assert.Equal(t, oldest.ID, got[1].ID)
		assert.Equal(t, reviews.Author{ID: anaID, Name: "Ana"}, got[1].User)
	})

	t.Run("lists the later of two simultaneous reviews first", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		first, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 5})
		require.NoError(t, err)
		second, err := repo.Create(ctx, &reviews.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Budi"), Rating: 4})
		require.NoError(t, err)
		_, err = testDB.Exec(`UPDATE PLACE_REVIEWS SET created_at = '2026-01-01T00:00:00Z'`)
		require.NoError(t, err)

		got, err := repo.GetByPlace(ctx, placeID)

		require.NoError(t, err)
		require.Len(t, got, 2)
		assert.Equal(t, []int64{second.ID, first.ID}, []int64{got[0].ID, got[1].ID})
	})

	t.Run("returns an empty, non-nil slice for a place without reviews", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetByPlace(ctx, insertPlace(t, "Laundry"))

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})

	t.Run("returns ErrPlaceNotFound for an unknown place", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetByPlace(ctx, 999)

		assert.ErrorIs(t, err, reviews.ErrPlaceNotFound)
		assert.Nil(t, got)
	})
}
