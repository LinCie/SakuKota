package reviews_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/domain"
	"rantau-go/internal/module/reviews"
	"rantau-go/internal/module/reviews/mock"
)

// txKey marks the context that newTransactor passes to the transaction's function.
type txKey struct{}

// inTx matches a context carrying the transaction newTransactor started.
var inTx = gomock.Cond(func(ctx context.Context) bool { return ctx.Value(txKey{}) != nil })

// newTransactor returns a mock Transactor that runs each function it is given
// with a context marked by txKey, returning the function's error.
func newTransactor(ctrl *gomock.Controller) *mock.MockTransactor {
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(context.WithValue(ctx, txKey{}, true))
	}).AnyTimes()
	return tx
}

// createReview runs Service.Create for r. It returns the review the repository
// was asked to store, or nil if storage was never reached.
func createReview(t *testing.T, r *domain.Review) (*domain.Review, error) {
	t.Helper()

	ctrl := gomock.NewController(t)
	var stored *domain.Review
	repo := mock.NewMockRepository(ctrl)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, got *domain.Review) (*domain.Review, error) {
		stored = got
		return got, nil
	}).AnyTimes()
	places := mock.NewMockPlaceService(ctrl)
	places.EXPECT().AddRating(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	_, err := reviews.NewService(repo, newTransactor(ctrl), places).Create(t.Context(), r)
	return stored, err
}

func TestService_Create_Rating(t *testing.T) {
	tests := []struct {
		name    string
		rating  int
		wantErr bool
	}{
		{name: "one_star", rating: 1, wantErr: false},
		{name: "three_stars", rating: 3, wantErr: false},
		{name: "five_stars", rating: 5, wantErr: false},
		{name: "zero", rating: 0, wantErr: true},
		{name: "six", rating: 6, wantErr: true},
		{name: "negative", rating: -1, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &domain.Review{PlaceID: 3, UserID: 7, Rating: tc.rating})
			if tc.wantErr {
				assert.ErrorIs(t, err, domain.ErrReviewRatingInvalid, "rating %d", tc.rating)
				assert.Nil(t, stored, "rating %d reached storage", tc.rating)
				return
			}
			require.NoError(t, err, "rating %d", tc.rating)
			require.NotNil(t, stored, "rating %d did not store the accepted review", tc.rating)
			assert.Equal(t, tc.rating, stored.Rating)
		})
	}
}

func TestService_Create_Comment(t *testing.T) {
	thousand := strings.Repeat("a", 1000)
	thousandMultiByte := strings.Repeat("é", 1000)

	tests := []struct {
		name    string
		comment *string
		want    *string
	}{
		{name: "absent_comment_stays_absent", comment: nil, want: nil},
		{name: "surrounding_spaces_are_removed", comment: new("  Murah, porsi besar  "), want: new("Murah, porsi besar")},
		{name: "tabs_and_newlines_are_removed", comment: new("\tMurah\n"), want: new("Murah")},
		{name: "inner_line_breaks_are_kept", comment: new("Murah.\nPorsi besar."), want: new("Murah.\nPorsi besar.")},
		{name: "empty_comment_is_stored_as_absent", comment: new(""), want: nil},
		{name: "blank_comment_is_stored_as_absent", comment: new(" \t\n "), want: nil},
		{name: "exactly_1000_characters", comment: new(thousand), want: new(thousand)},
		{name: "1000_multibyte_characters", comment: new(thousandMultiByte), want: new(thousandMultiByte)},
		{name: "surrounding_spaces_do_not_count_toward_the_limit", comment: new(" " + thousand + " "), want: new(thousand)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &domain.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: tc.comment})
			require.NoError(t, err, "comment %q", ptrString(tc.comment))
			require.NotNil(t, stored, "comment %q did not store the accepted review", ptrString(tc.comment))
			assert.Equal(t, tc.want, stored.Comment, "comment %q", ptrString(tc.comment))
		})
	}
}

// ptrString returns *s, or "<nil>" if s is nil, for failure messages.
func ptrString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestService_Create_CommentTooLong(t *testing.T) {
	tests := []struct {
		name    string
		comment string
	}{
		{name: "1001_characters", comment: strings.Repeat("a", 1001)},
		{name: "1001_multibyte_characters", comment: strings.Repeat("é", 1001)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &domain.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new(tc.comment)})
			assert.ErrorIs(t, err, domain.ErrReviewCommentTooLong, tc.name)
			assert.Nil(t, stored, "%s reached storage", tc.name)
		})
	}
}

func TestService_Create_KeepsPlaceAndAuthor(t *testing.T) {
	stored, err := createReview(t, &domain.Review{PlaceID: 3, UserID: 7, Rating: 4})
	require.NoError(t, err)
	require.NotNil(t, stored, "Create() did not store the accepted review")
	assert.EqualValues(t, 3, stored.PlaceID)
	assert.EqualValues(t, 7, stored.UserID)
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		storeErr  error // returned when the review is stored
		ratingErr error // returned when the rating is added; not reached if storeErr is set
		wantErr   error // nil means the call succeeds
	}{
		{name: "unknown_place", storeErr: domain.ErrPlaceNotFound, ratingErr: nil, wantErr: domain.ErrPlaceNotFound},
		{name: "earlier_review", storeErr: domain.ErrPlaceAlreadyReviewed, ratingErr: nil, wantErr: domain.ErrPlaceAlreadyReviewed},
		{name: "storage_failure", storeErr: boom, ratingErr: nil, wantErr: boom},
		{name: "place_gone_before_rating", storeErr: nil, ratingErr: domain.ErrPlaceNotFound, wantErr: domain.ErrPlaceNotFound},
		{name: "rating_failure", storeErr: nil, ratingErr: boom, wantErr: boom},
		{name: "stored_review_is_returned", storeErr: nil, ratingErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			places := mock.NewMockPlaceService(ctrl)
			// Both writes run, in order, inside the one transaction.
			store := repo.EXPECT().Create(inTx, &domain.Review{PlaceID: 3, UserID: 7, Rating: 4}).Return(&domain.Review{ID: 9}, tc.storeErr)
			if tc.storeErr == nil {
				places.EXPECT().AddRating(inTx, int64(3), 4).Return(tc.ratingErr).After(store)
			}

			got, err := reviews.NewService(repo, newTransactor(ctrl), places).Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4})
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got, "Create() = nil, want the stored review")
			assert.EqualValues(t, 9, got.ID)
		})
	}
}

func TestService_Create_InvalidReviewStartsNoTransaction(t *testing.T) {
	ctrl := gomock.NewController(t)
	service := reviews.NewService(mock.NewMockRepository(ctrl), mock.NewMockTransactor(ctrl), mock.NewMockPlaceService(ctrl))

	_, err := service.Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 0})

	assert.ErrorIs(t, err, domain.ErrReviewRatingInvalid)
}

func TestService_Create_TransactionFailure(t *testing.T) {
	boom := errors.New("connection reset")
	ctrl := gomock.NewController(t)
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).Return(boom)

	got, err := reviews.NewService(mock.NewMockRepository(ctrl), tx, mock.NewMockPlaceService(ctrl)).Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4})

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestService_ListByPlace(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		reviews []domain.Review // returned by storage
		repoErr error           // returned by storage
		wantIDs []int64
	}{
		{name: "reviews_keep_the_stored_order", reviews: []domain.Review{{ID: 5}, {ID: 2}, {ID: 9}}, repoErr: nil, wantIDs: []int64{5, 2, 9}},
		{name: "no_reviews", reviews: []domain.Review{}, repoErr: nil, wantIDs: []int64{}},
		{name: "unknown_place", reviews: nil, repoErr: domain.ErrPlaceNotFound, wantIDs: nil},
		{name: "storage_failure", reviews: nil, repoErr: boom, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().ListByPlace(gomock.Any(), int64(3)).Return(tc.reviews, tc.repoErr)

			got, err := reviews.NewService(repo, nil, nil).ListByPlace(t.Context(), 3)
			if tc.repoErr != nil {
				assert.ErrorIs(t, err, tc.repoErr)
				return
			}
			require.NoError(t, err)

			ids := make([]int64, 0, len(got))
			for _, r := range got {
				ids = append(ids, r.ID)
			}
			assert.Equal(t, tc.wantIDs, ids)
		})
	}
}
