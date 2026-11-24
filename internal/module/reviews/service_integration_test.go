//go:build integration

package reviews_test

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/reviews"
	"sakukota-go/internal/module/reviews/mock"
	"sakukota-go/internal/testutil/pgtest"
)

// newWiredService returns a Service over an emptied database, wired as in
// production: ratings go through the place service in the review's transaction.
func newWiredService(t *testing.T) reviews.Service {
	t.Helper()
	pgtest.Reset(t, testDB)
	placeService := places.NewService(places.NewPostgresRepository(testDB), nil)
	return reviews.NewService(reviews.NewPostgresRepository(testDB), database.NewTransactor(testDB), placeService, nil)
}

func TestService_Create_RatingTotals(t *testing.T) {
	t.Run("adds_each_rating_to_totals", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		otherPlaceID := insertPlace(t, "Other")

		for i, rating := range []int{5, 3, 1} {
			userID := pgtest.InsertUser(t, testDB, fmt.Sprintf("User%d", i))
			_, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: rating})
			require.NoError(t, err, "Create(rating %d)", rating)
		}

		assert.Equal(t, [2]int{3, 9}, ratingTotals(t, placeID))
		// Other places are untouched.
		assert.Equal(t, [2]int{0, 0}, ratingTotals(t, otherPlaceID))
	})

	t.Run("concurrent_reviews_lose_no_rating", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		const reviewers = 20

		userIDs := make([]int64, reviewers)
		for i := range userIDs {
			userIDs[i] = pgtest.InsertUser(t, testDB, fmt.Sprintf("User%d", i))
		}

		var wg sync.WaitGroup
		errs := make([]error, reviewers)
		for i, userID := range userIDs {
			wg.Go(func() {
				_, errs[i] = service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: 4})
			})
		}
		wg.Wait()

		for i, err := range errs {
			require.NoError(t, err, "Create() by reviewer %d", i)
		}
		assert.Equal(t, [2]int{reviewers, reviewers * 4}, ratingTotals(t, placeID))
	})

	t.Run("second_review_adds_no_rating", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		userID := pgtest.InsertUser(t, testDB, "Ana")
		_, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: 5})
		require.NoError(t, err, "first Create()")

		_, err = service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: userID, Rating: 1})

		assert.ErrorIs(t, err, domain.ErrPlaceAlreadyReviewed)
		assert.Equal(t, [2]int{1, 5}, ratingTotals(t, placeID))
	})

	t.Run("unknown_user_adds_no_rating", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")

		_, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: 999, Rating: 5})

		assert.Error(t, err)
		assert.Equal(t, [2]int{0, 0}, ratingTotals(t, placeID))
	})

	t.Run("failed_rating_discards_review", func(t *testing.T) {
		pgtest.Reset(t, testDB)
		placeID := insertPlace(t, "Laundry")
		boom := errors.New("boom")
		placeService := mock.NewMockPlaceService(gomock.NewController(t))
		placeService.EXPECT().AddRating(gomock.Any(), placeID, 4).Return(boom)
		repo := reviews.NewPostgresRepository(testDB)
		service := reviews.NewService(repo, database.NewTransactor(testDB), placeService, nil)

		_, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 4})

		assert.ErrorIs(t, err, boom)
		stored, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)
		assert.Empty(t, stored, "the review outlived the rollback")
	})
}

func TestService_Delete_RatingTotals(t *testing.T) {
	// createReview stores a review through service and returns its id.
	createReview := func(t *testing.T, service reviews.Service, placeID int64, name string, rating int) int64 {
		t.Helper()
		got, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, name), Rating: rating})
		require.NoError(t, err, "Create(%s, rating %d)", name, rating)
		return got.ID
	}

	t.Run("removes_the_rating", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		reviewID := createReview(t, service, placeID, "Ana", 5)
		createReview(t, service, placeID, "Budi", 3)

		require.NoError(t, service.Delete(t.Context(), placeID, reviewID))

		assert.Equal(t, [2]int{1, 3}, ratingTotals(t, placeID))
	})

	t.Run("concurrent_deletes_remove_it_once", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		reviewID := createReview(t, service, placeID, "Ana", 5)
		createReview(t, service, placeID, "Budi", 3)
		const deleters = 10

		var wg sync.WaitGroup
		errs := make([]error, deleters)
		for i := range deleters {
			wg.Go(func() {
				errs[i] = service.Delete(t.Context(), placeID, reviewID)
			})
		}
		wg.Wait()

		deleted := 0
		for i, err := range errs {
			if err == nil { // if NO error
				deleted++
				continue
			}
			assert.ErrorIs(t, err, domain.ErrReviewNotFound, "Delete() by deleter %d", i)
		}
		assert.Equal(t, 1, deleted, "successful deletes")
		assert.Equal(t, [2]int{1, 3}, ratingTotals(t, placeID))
	})

	t.Run("author_deletes_own_review", func(t *testing.T) {
		service := newWiredService(t)
		placeID := insertPlace(t, "Laundry")
		anaID := pgtest.InsertUser(t, testDB, "Ana")
		_, err := service.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: anaID, Rating: 5})
		require.NoError(t, err)
		createReview(t, service, placeID, "Budi", 3)

		require.NoError(t, service.DeleteMine(t.Context(), placeID, anaID))

		assert.Equal(t, [2]int{1, 3}, ratingTotals(t, placeID))
		assert.ErrorIs(t, service.DeleteMine(t.Context(), placeID, anaID), domain.ErrReviewNotFound, "second DeleteMine()")
	})

	t.Run("failed_rating_keeps_review", func(t *testing.T) {
		pgtest.Reset(t, testDB)
		placeID := insertPlace(t, "Laundry")
		repo := reviews.NewPostgresRepository(testDB)
		created, err := repo.Create(t.Context(), &domain.Review{PlaceID: placeID, UserID: pgtest.InsertUser(t, testDB, "Ana"), Rating: 4})
		require.NoError(t, err)
		boom := errors.New("boom")
		placeService := mock.NewMockPlaceService(gomock.NewController(t))
		placeService.EXPECT().RemoveRating(gomock.Any(), placeID, 4).Return(boom)
		service := reviews.NewService(repo, database.NewTransactor(testDB), placeService, nil)

		err = service.Delete(t.Context(), placeID, created.ID)

		assert.ErrorIs(t, err, boom)
		stored, err := repo.ListByPlace(t.Context(), placeID)
		require.NoError(t, err)
		assert.Len(t, stored, 1, "the review was deleted despite the rollback")
	})
}
