//go:build integration

package reviews_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/reviews"
	"sakukota-go/internal/testutil/pgtest"
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

	t.Run("with_medias", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		withMedias := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: anaID, Rating: 5})
		mustAddMedia(t, repo, placeID, anaID, "reviews/1/1/a.png")
		mustAddMedia(t, repo, placeID, anaID, "reviews/1/1/b.png")
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Budi"), Rating: 4})
		want, err := repo.Fetch(t.Context(), placeID, anaID)
		require.NoError(t, err)

		got, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)

		require.Len(t, got, 2)
		assert.Empty(t, got[0].Medias, "medias of the review without any")
		assert.Equal(t, withMedias.ID, got[1].ID)
		assert.Equal(t, want.Medias, got[1].Medias)
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

func TestPostgresRepository_Fetch(t *testing.T) {
	t.Run("users_review_of_the_place", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		created := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: anaID, Rating: 4})
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Budi"), Rating: 2})
		mustCreate(t, repo, &domain.Review{PlaceID: insertPlace(t, "Other"), UserID: anaID, Rating: 1})

		got, err := repo.Fetch(t.Context(), placeID, anaID)
		require.NoError(t, err)

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, domain.Author{ID: anaID, Name: "Ana"}, got.User)
		assert.Empty(t, got.Medias)
	})

	t.Run("no_review", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")

		got, err := repo.Fetch(t.Context(), placeID, pgtest.InsertUser(t, testDB, "Ana"))

		assert.ErrorIs(t, err, domain.ErrReviewNotFound)
		assert.Nil(t, got)
	})
}

func TestPostgresRepository_FetchByID(t *testing.T) {
	t.Run("review_of_the_place_with_author_and_medias", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		created := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: anaID, Rating: 4, Comment: new("Bersih")})
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Budi"), Rating: 2})
		media := mustAddMedia(t, repo, placeID, anaID, "reviews/1/1/a.webp")

		got, err := repo.FetchByID(t.Context(), placeID, created.ID)
		require.NoError(t, err)

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, new("Bersih"), got.Comment)
		assert.Equal(t, domain.Author{ID: anaID, Name: "Ana"}, got.User)
		assert.Equal(t, []domain.Media{media}, got.Medias)
	})

	t.Run("unknown_or_of_another_place", func(t *testing.T) {
		repo := newRepository(t)
		placeID := insertPlace(t, "Laundry")
		created := mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 4})

		for _, ids := range [][2]int64{{insertPlace(t, "Other"), created.ID}, {placeID, created.ID + 1}} {
			got, err := repo.FetchByID(t.Context(), ids[0], ids[1])
			assert.ErrorIs(t, err, domain.ErrReviewNotFound, "FetchByID(%d, %d)", ids[0], ids[1])
			assert.Nil(t, got)
		}
	})
}

// newMedia returns a promoted image to store under key.
func newMedia(key string) domain.Media {
	return domain.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

// mustAddMedia stores an image under key for the user's review of the place and returns it.
func mustAddMedia(t *testing.T, repo *reviews.PostgresRepository, placeID, userID int64, key string) domain.Media {
	t.Helper()

	got, err := repo.AddMedia(t.Context(), placeID, userID, newMedia(key))
	require.NoError(t, err, "AddMedia(%d, %d, %q)", placeID, userID, key)

	for _, m := range got.Medias {
		if m.Key == key {
			return m
		}
	}
	require.Failf(t, "media missing", "AddMedia(%d, %d, %q) returned a review without the added media", placeID, userID, key)

	return domain.Media{}
}

// mediaKeys returns the keys of medias, in order.
func mediaKeys(medias []domain.Media) []string {
	out := make([]string, 0, len(medias))
	for _, m := range medias {
		out = append(out, m.Key)
	}

	return out
}

// mediaPositions returns the positions of medias, in order.
func mediaPositions(medias []domain.Media) []int {
	out := make([]int, 0, len(medias))
	for _, m := range medias {
		out = append(out, m.Position)
	}

	return out
}

// storedMediaCount returns the number of MEDIAS rows.
func storedMediaCount(t *testing.T) int {
	t.Helper()

	var count int
	require.NoError(t, testDB.Get(&count, `SELECT COUNT(*) FROM MEDIAS`), "count medias")

	return count
}

// newReview returns a repository over an emptied database with a review by a
// new user of a new place, and the ids of the place and the user.
func newReview(t *testing.T) (*reviews.PostgresRepository, int64, int64) {
	t.Helper()

	repo := newRepository(t)
	placeID := insertPlace(t, "Laundry")
	userID := pgtest.InsertUser(t, testDB, "Ana")
	mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: userID, Rating: 4})

	return repo, placeID, userID
}

func TestPostgresRepository_AddMedia(t *testing.T) {
	ctx := t.Context()

	t.Run("first_position", func(t *testing.T) {
		repo, placeID, userID := newReview(t)

		got, err := repo.AddMedia(ctx, placeID, userID, newMedia("reviews/1/1/a.webp"))
		require.NoError(t, err)

		assert.Equal(t, domain.Author{ID: userID, Name: "Ana"}, got.User)
		require.Len(t, got.Medias, 1)
		m := got.Medias[0]
		assert.NotZero(t, m.ID, "Medias[0].ID")
		assert.Equal(t, domain.Media{ID: m.ID, Key: "reviews/1/1/a.webp", ContentType: "image/webp", Size: 2048, Position: 1}, m)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys (adding a media replaces nothing)")

		stored, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		assert.Equal(t, got, stored, "Fetch() differs from the returned review")
	})

	t.Run("after_delete", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")
		removed := mustAddMedia(t, repo, placeID, userID, "reviews/1/1/b.webp")
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/c.webp")
		require.NoError(t, repo.DeleteMedia(ctx, placeID, userID, removed.ID))

		got, err := repo.AddMedia(ctx, placeID, userID, newMedia("reviews/1/1/d.webp"))
		require.NoError(t, err)

		assert.Equal(t, []string{"reviews/1/1/a.webp", "reviews/1/1/c.webp", "reviews/1/1/d.webp"}, mediaKeys(got.Medias))
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias))
	})

	t.Run("limit_reached", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		for i := range domain.MaxMedias {
			mustAddMedia(t, repo, placeID, userID, fmt.Sprintf("reviews/1/1/%d.webp", i))
		}

		got, err := repo.AddMedia(ctx, placeID, userID, newMedia("reviews/1/1/extra.webp"))

		assert.ErrorIs(t, err, domain.ErrMediaLimitReached)
		assert.Nil(t, got)
		assert.Equal(t, domain.MaxMedias, storedMediaCount(t), "stored medias (the extra media must not be stored)")
	})

	t.Run("concurrent_limit", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		const attempts = domain.MaxMedias + 4

		errs := make([]error, attempts)
		var wg sync.WaitGroup
		for i := range attempts {
			wg.Go(func() {
				_, errs[i] = repo.AddMedia(ctx, placeID, userID, newMedia(fmt.Sprintf("reviews/1/1/%d.webp", i)))
			})
		}
		wg.Wait()

		added, limited := 0, 0
		for _, err := range errs {
			switch {
			case err == nil: // if NO error
				added++
			case errors.Is(err, domain.ErrMediaLimitReached):
				limited++
			default:
				assert.Fail(t, "unexpected error", "AddMedia() unexpected error: %v", err)
			}
		}
		assert.Equal(t, domain.MaxMedias, added, "concurrent AddMedia() added")
		assert.Equal(t, attempts-domain.MaxMedias, limited, "concurrent AddMedia() limited")

		stored, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, mediaPositions(stored.Medias))
	})

	t.Run("no_review", func(t *testing.T) {
		repo, placeID, _ := newReview(t)

		got, err := repo.AddMedia(ctx, placeID, pgtest.InsertUser(t, testDB, "Budi"), newMedia("reviews/1/2/a.webp"))

		assert.ErrorIs(t, err, domain.ErrReviewNotFound)
		assert.Nil(t, got)
		assert.Zero(t, storedMediaCount(t), "stored medias")
	})
}

func TestPostgresRepository_DeleteMedia(t *testing.T) {
	ctx := t.Context()

	t.Run("queues_image", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")
		removed := mustAddMedia(t, repo, placeID, userID, "reviews/1/1/b.webp")
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/c.webp")

		require.NoError(t, repo.DeleteMedia(ctx, placeID, userID, removed.ID))

		stored, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		assert.Equal(t, []string{"reviews/1/1/a.webp", "reviews/1/1/c.webp"}, mediaKeys(stored.Medias))
		// The later medias shift down.
		assert.Equal(t, []int{1, 2}, mediaPositions(stored.Medias))
		assert.Equal(t, 2, storedMediaCount(t), "stored medias")
		assert.Equal(t, []string{"reviews/1/1/b.webp"}, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("already_queued", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		removed := mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('reviews/1/1/a.webp')`)
		require.NoError(t, err, "queue key")

		require.NoError(t, repo.DeleteMedia(ctx, placeID, userID, removed.ID))

		assert.Equal(t, []string{"reviews/1/1/a.webp"}, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("other_users_review", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		m := mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")
		budiID := pgtest.InsertUser(t, testDB, "Budi")
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: budiID, Rating: 3})

		err := repo.DeleteMedia(ctx, placeID, budiID, m.ID)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound)
		assert.Equal(t, 1, storedMediaCount(t), "stored medias (the media must be kept)")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("unknown_media_or_review", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		m := mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")

		assert.ErrorIs(t, repo.DeleteMedia(ctx, placeID, userID, 999), domain.ErrMediaNotFound, "unknown media")
		assert.ErrorIs(t, repo.DeleteMedia(ctx, 999, userID, m.ID), domain.ErrMediaNotFound, "unknown review")
		assert.Equal(t, 1, storedMediaCount(t), "stored medias")
	})
}

func TestPostgresRepository_Delete(t *testing.T) {
	ctx := t.Context()

	t.Run("removes_review_and_queues_medias", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/a.webp")
		mustAddMedia(t, repo, placeID, userID, "reviews/1/1/b.webp")
		review, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		budiID := pgtest.InsertUser(t, testDB, "Budi")
		mustCreate(t, repo, &domain.Review{PlaceID: placeID, UserID: budiID, Rating: 2})
		mustAddMedia(t, repo, placeID, budiID, "reviews/1/2/a.webp")

		rating, err := repo.Delete(ctx, placeID, review.ID)
		require.NoError(t, err)

		assert.Equal(t, 4, rating, "Delete() rating")
		_, err = repo.Fetch(ctx, placeID, userID)
		assert.ErrorIs(t, err, domain.ErrReviewNotFound, "Fetch() after Delete()")
		assert.Equal(t, []string{"reviews/1/1/a.webp", "reviews/1/1/b.webp"}, pgtest.QueuedKeys(t, testDB), "queued keys")
		// Other reviews and their medias are untouched.
		assert.Equal(t, 1, storedMediaCount(t), "stored medias")
		_, err = repo.Fetch(ctx, placeID, budiID)
		assert.NoError(t, err, "Fetch() of the other review")
	})

	t.Run("author_can_review_again", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		review, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		_, err = repo.Delete(ctx, placeID, review.ID)
		require.NoError(t, err)

		_, err = repo.Create(ctx, &domain.Review{PlaceID: placeID, UserID: userID, Rating: 5})

		assert.NoError(t, err, "Create() after Delete()")
	})

	t.Run("review_of_another_place", func(t *testing.T) {
		repo, placeID, userID := newReview(t)
		review, err := repo.Fetch(ctx, placeID, userID)
		require.NoError(t, err)
		otherPlaceID := insertPlace(t, "Other")

		_, err = repo.Delete(ctx, otherPlaceID, review.ID)

		assert.ErrorIs(t, err, domain.ErrReviewNotFound)
		_, err = repo.Fetch(ctx, placeID, userID)
		assert.NoError(t, err, "the review must be kept")
	})

	t.Run("unknown_review", func(t *testing.T) {
		repo, placeID, _ := newReview(t)

		_, err := repo.Delete(ctx, placeID, 999)

		assert.ErrorIs(t, err, domain.ErrReviewNotFound)
	})
}
