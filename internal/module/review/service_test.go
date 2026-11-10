package review_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/review"
	"rantau-go/internal/module/review/mock"
)

// createReview runs Service.Create for r. It returns the review the repository
// was asked to store, or nil if storage was never reached.
func createReview(t *testing.T, r *review.Review) (*review.Review, error) {
	t.Helper()

	repo := mock.NewMockRepository(gomock.NewController(t))

	var stored *review.Review
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, got *review.Review) (*review.Review, error) {
			stored = got
			return got, nil
		}).AnyTimes()

	_, err := review.NewService(repo).Create(context.Background(), r)
	return stored, err
}

// -----------------------------------------------------------------------------
// Creating reviews
// -----------------------------------------------------------------------------

func TestService_Create_Rating(t *testing.T) {
	tests := []struct {
		name    string
		rating  int
		wantErr bool
	}{
		{name: "one star", rating: 1, wantErr: false},
		{name: "three stars", rating: 3, wantErr: false},
		{name: "five stars", rating: 5, wantErr: false},
		{name: "zero", rating: 0, wantErr: true},
		{name: "six", rating: 6, wantErr: true},
		{name: "negative", rating: -1, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &review.Review{PlaceID: 3, UserID: 7, Rating: tc.rating})
			if tc.wantErr {
				require.ErrorIs(t, err, review.ErrInvalidRating)
				require.Nil(t, stored, "rating %d reached storage", tc.rating)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted review was not stored")
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
		{name: "absent comment stays absent", comment: nil, want: nil},
		{name: "surrounding spaces are removed", comment: new("  Murah, porsi besar  "), want: new("Murah, porsi besar")},
		{name: "tabs and newlines are removed", comment: new("\tMurah\n"), want: new("Murah")},
		{name: "inner line breaks are kept", comment: new("Murah.\nPorsi besar."), want: new("Murah.\nPorsi besar.")},
		{name: "empty comment is stored as absent", comment: new(""), want: nil},
		{name: "blank comment is stored as absent", comment: new(" \t\n "), want: nil},
		{name: "exactly 1000 characters", comment: new(thousand), want: new(thousand)},
		{name: "1000 multi-byte characters", comment: new(thousandMultiByte), want: new(thousandMultiByte)},
		{name: "surrounding spaces do not count toward the limit", comment: new(" " + thousand + " "), want: new(thousand)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &review.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: tc.comment})
			require.NoError(t, err)
			require.NotNil(t, stored, "accepted review was not stored")

			if tc.want == nil {
				assert.Nil(t, stored.Comment)
				return
			}
			require.NotNil(t, stored.Comment, "stored no comment, want %q", *tc.want)
			assert.Equal(t, *tc.want, *stored.Comment)
		})
	}
}

func TestService_Create_CommentTooLong(t *testing.T) {
	tests := []struct {
		name    string
		comment string
	}{
		{name: "1001 characters", comment: strings.Repeat("a", 1001)},
		{name: "1001 multi-byte characters", comment: strings.Repeat("é", 1001)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := createReview(t, &review.Review{PlaceID: 3, UserID: 7, Rating: 4, Comment: new(tc.comment)})
			require.ErrorIs(t, err, review.ErrCommentTooLong)
			require.Nil(t, stored, "too long comment reached storage")
		})
	}
}

func TestService_Create_KeepsPlaceAndAuthor(t *testing.T) {
	stored, err := createReview(t, &review.Review{PlaceID: 3, UserID: 7, Rating: 4})
	require.NoError(t, err)
	require.NotNil(t, stored, "accepted review was not stored")
	assert.Equal(t, int64(3), stored.PlaceID)
	assert.Equal(t, int64(7), stored.UserID)
}

func TestService_Create_Storage(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name     string
		storeErr error // returned when the review is stored
		wantErr  error // nil means the call succeeds
	}{
		{name: "storage reports an unknown place", storeErr: review.ErrPlaceNotFound, wantErr: review.ErrPlaceNotFound},
		{name: "storage reports an earlier review", storeErr: review.ErrAlreadyReviewed, wantErr: review.ErrAlreadyReviewed},
		{name: "storage failure is returned", storeErr: boom, wantErr: boom},
		{name: "stored review is returned", storeErr: nil, wantErr: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(&review.Review{ID: 9}, tc.storeErr)

			got, err := review.NewService(repo).Create(context.Background(), &review.Review{
				PlaceID: 3, UserID: 7, Rating: 4,
			})
			require.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				require.NotNil(t, got, "stored review was not returned")
				assert.Equal(t, int64(9), got.ID)
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Listing reviews
// -----------------------------------------------------------------------------

func TestService_GetByPlace(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name    string
		reviews []review.Review // returned by storage
		repoErr error           // returned by storage
		wantIDs []int64
	}{
		{name: "reviews keep the stored order", reviews: []review.Review{{ID: 5}, {ID: 2}, {ID: 9}}, repoErr: nil, wantIDs: []int64{5, 2, 9}},
		{name: "no reviews", reviews: []review.Review{}, repoErr: nil, wantIDs: []int64{}},
		{name: "unknown place", reviews: nil, repoErr: review.ErrPlaceNotFound, wantIDs: nil},
		{name: "storage failure", reviews: nil, repoErr: boom, wantIDs: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := mock.NewMockRepository(gomock.NewController(t))
			repo.EXPECT().GetByPlace(gomock.Any(), int64(3)).Return(tc.reviews, tc.repoErr)

			got, err := review.NewService(repo).GetByPlace(context.Background(), 3)
			require.ErrorIs(t, err, tc.repoErr)
			if tc.repoErr != nil {
				return
			}

			ids := make([]int64, 0, len(got))
			for _, r := range got {
				ids = append(ids, r.ID)
			}
			assert.Equal(t, tc.wantIDs, ids)
		})
	}
}
