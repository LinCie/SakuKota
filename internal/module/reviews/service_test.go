package reviews_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/reviews"
	"sakukota-go/internal/module/reviews/mock"
	"sakukota-go/internal/storage"
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

	_, err := reviews.NewService(repo, newTransactor(ctrl), places, mock.NewMockMediaStorage(ctrl)).Create(t.Context(), r)
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

			got, err := reviews.NewService(repo, newTransactor(ctrl), places, mock.NewMockMediaStorage(ctrl)).Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4})
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
	service := reviews.NewService(mock.NewMockRepository(ctrl), mock.NewMockTransactor(ctrl), mock.NewMockPlaceService(ctrl), mock.NewMockMediaStorage(ctrl))

	_, err := service.Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 0})

	assert.ErrorIs(t, err, domain.ErrReviewRatingInvalid)
}

func TestService_Create_TransactionFailure(t *testing.T) {
	boom := errors.New("connection reset")
	ctrl := gomock.NewController(t)
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).Return(boom)

	got, err := reviews.NewService(mock.NewMockRepository(ctrl), tx, mock.NewMockPlaceService(ctrl), mock.NewMockMediaStorage(ctrl)).Create(t.Context(), &domain.Review{PlaceID: 3, UserID: 7, Rating: 4})

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestService_Delete(t *testing.T) {
	boom := errors.New("connection reset")

	tests := []struct {
		name      string
		deleteErr error // returned when the review is deleted
		ratingErr error // returned when the rating is removed; not reached if deleteErr is set
	}{
		{name: "deleted", deleteErr: nil, ratingErr: nil},
		{name: "unknown_review", deleteErr: domain.ErrReviewNotFound, ratingErr: nil},
		{name: "storage_failure", deleteErr: boom, ratingErr: nil},
		{name: "rating_failure", deleteErr: nil, ratingErr: boom},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			places := mock.NewMockPlaceService(ctrl)
			// Both writes run, in order, inside the one transaction.
			del := repo.EXPECT().Delete(inTx, int64(3), int64(9)).Return(4, tc.deleteErr)
			if tc.deleteErr == nil {
				places.EXPECT().RemoveRating(inTx, int64(3), 4).Return(tc.ratingErr).After(del)
			}

			err := reviews.NewService(repo, newTransactor(ctrl), places, mock.NewMockMediaStorage(ctrl)).Delete(t.Context(), 3, 9)

			wantErr := tc.deleteErr
			if wantErr == nil {
				wantErr = tc.ratingErr
			}
			if wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, wantErr)
		})
	}
}

func TestService_DeleteMine(t *testing.T) {
	t.Run("deletes_the_users_review", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		repo := mock.NewMockRepository(ctrl)
		places := mock.NewMockPlaceService(ctrl)
		fetch := repo.EXPECT().Fetch(gomock.Any(), int64(3), int64(7)).Return(&domain.Review{ID: 9, PlaceID: 3, UserID: 7, Rating: 4}, nil)
		del := repo.EXPECT().Delete(inTx, int64(3), int64(9)).Return(4, nil).After(fetch)
		places.EXPECT().RemoveRating(inTx, int64(3), 4).Return(nil).After(del)

		err := reviews.NewService(repo, newTransactor(ctrl), places, mock.NewMockMediaStorage(ctrl)).DeleteMine(t.Context(), 3, 7)

		assert.NoError(t, err)
	})

	t.Run("lookup_failure_deletes_nothing", func(t *testing.T) {
		for _, lookupErr := range []error{domain.ErrReviewNotFound, errors.New("connection reset")} {
			ctrl := gomock.NewController(t)
			repo := mock.NewMockRepository(ctrl)
			repo.EXPECT().Fetch(gomock.Any(), int64(3), int64(7)).Return(nil, lookupErr)

			err := reviews.NewService(repo, mock.NewMockTransactor(ctrl), mock.NewMockPlaceService(ctrl), mock.NewMockMediaStorage(ctrl)).DeleteMine(t.Context(), 3, 7)

			assert.ErrorIs(t, err, lookupErr)
		}
	})
}

// Review media fixtures: user 7's review of place 3.
const (
	mediaPrefix = "reviews/3/7"
	uploadKey   = "uploads/reviews/3/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp"
)

var promoted = domain.Media{Key: "reviews/3/7/0b1e6d4c-3f2a-4b8e-9c7d-5a6f1e2d3c4b.webp", ContentType: "image/webp", Size: 2048}

// newMediaMocks returns mocks of the repository and media storage that fail the
// test on any call without an expectation, and a service using them.
func newMediaMocks(t *testing.T) (*mock.MockRepository, *mock.MockMediaStorage, reviews.Service) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mock.NewMockRepository(ctrl)
	media := mock.NewMockMediaStorage(ctrl)
	return repo, media, reviews.NewService(repo, mock.NewMockTransactor(ctrl), mock.NewMockPlaceService(ctrl), media)
}

// fetchReview expects the lookup of user 7's review of place 3, which it finds.
func fetchReview(repo *mock.MockRepository) *gomock.Call {
	return repo.EXPECT().Fetch(gomock.Any(), int64(3), int64(7)).Return(&domain.Review{ID: 9, PlaceID: 3, UserID: 7}, nil)
}

func TestService_MediaUploadURL(t *testing.T) {
	repo, media, service := newMediaMocks(t)
	upload := storage.Upload{Key: uploadKey, URL: "http://localhost:3900/signed", Method: "PUT"}
	gomock.InOrder(
		fetchReview(repo),
		media.EXPECT().PresignUpload(gomock.Any(), mediaPrefix, "image/webp").Return(upload, nil),
	)

	got, err := service.MediaUploadURL(t.Context(), 3, 7, "image/webp")

	require.NoError(t, err)
	assert.Equal(t, upload, got)
}

func TestService_MediaUploadURL_LookupFailure(t *testing.T) {
	for _, lookupErr := range []error{domain.ErrReviewNotFound, errors.New("connection reset")} {
		// Nothing may be presigned without the user's review.
		repo, _, service := newMediaMocks(t)
		repo.EXPECT().Fetch(gomock.Any(), int64(3), int64(7)).Return(nil, lookupErr)

		_, err := service.MediaUploadURL(t.Context(), 3, 7, "image/webp")

		assert.ErrorIs(t, err, lookupErr)
	}
}

func TestService_MediaUploadURL_PresignFailure(t *testing.T) {
	for _, presignErr := range []error{storage.ErrInvalidMediaType, errors.New("connection reset")} {
		repo, media, service := newMediaMocks(t)
		fetchReview(repo)
		media.EXPECT().PresignUpload(gomock.Any(), mediaPrefix, "image/gif").Return(storage.Upload{}, presignErr)

		_, err := service.MediaUploadURL(t.Context(), 3, 7, "image/gif")

		assert.ErrorIs(t, err, presignErr)
	}
}

func TestService_AddMedia(t *testing.T) {
	// The promoted media is stored as it is, and nothing is deleted.
	repo, media, service := newMediaMocks(t)
	updated := &domain.Review{ID: 9, Medias: []domain.Media{{ID: 3, Key: promoted.Key, Position: 1}}}
	gomock.InOrder(
		fetchReview(repo),
		media.EXPECT().Promote(gomock.Any(), mediaPrefix, uploadKey).Return(promoted, nil),
		repo.EXPECT().AddMedia(gomock.Any(), int64(3), int64(7), promoted).Return(updated, nil),
	)

	got, err := service.AddMedia(t.Context(), 3, 7, uploadKey)

	require.NoError(t, err)
	assert.Same(t, updated, got, "AddMedia() must return the stored review")
}

func TestService_AddMedia_LookupFailure(t *testing.T) {
	for _, lookupErr := range []error{domain.ErrReviewNotFound, errors.New("connection reset")} {
		// Nothing may be promoted without the user's review.
		repo, _, service := newMediaMocks(t)
		repo.EXPECT().Fetch(gomock.Any(), int64(3), int64(7)).Return(nil, lookupErr)

		_, err := service.AddMedia(t.Context(), 3, 7, uploadKey)

		assert.ErrorIs(t, err, lookupErr)
	}
}

func TestService_AddMedia_PromoteFailure(t *testing.T) {
	tests := []struct {
		name       string
		promoteErr error
	}{
		{name: "invalid_key", promoteErr: storage.ErrInvalidUploadKey},
		{name: "not_uploaded", promoteErr: storage.ErrMediaNotUploaded},
		{name: "invalid_type", promoteErr: storage.ErrInvalidMediaType},
		{name: "image_too_large", promoteErr: storage.ErrImageTooLarge},
		{name: "storage_failure", promoteErr: errors.New("connection reset")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Nothing may be stored.
			repo, media, service := newMediaMocks(t)
			fetchReview(repo)
			media.EXPECT().Promote(gomock.Any(), mediaPrefix, uploadKey).Return(domain.Media{}, tc.promoteErr)

			_, err := service.AddMedia(t.Context(), 3, 7, uploadKey)

			assert.ErrorIs(t, err, tc.promoteErr)
		})
	}
}

func TestService_AddMedia_StoreFailure(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name      string
		addErr    error
		deleteErr error
	}{
		{name: "review_deleted", addErr: domain.ErrReviewNotFound},
		{name: "media_limit", addErr: domain.ErrMediaLimitReached},
		{name: "storage_failure", addErr: boom},
		{name: "delete_failure_ignored", addErr: boom, deleteErr: errors.New("timeout")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The promoted file is removed, even once the request is cancelled.
			ctx, cancel := context.WithCancel(t.Context())
			repo, media, service := newMediaMocks(t)
			gomock.InOrder(
				fetchReview(repo),
				media.EXPECT().Promote(gomock.Any(), mediaPrefix, uploadKey).Return(promoted, nil),
				repo.EXPECT().AddMedia(gomock.Any(), int64(3), int64(7), promoted).DoAndReturn(func(context.Context, int64, int64, domain.Media) (*domain.Review, error) {
					cancel()
					return nil, tc.addErr
				}),
				media.EXPECT().Delete(gomock.Any(), promoted.Key).DoAndReturn(func(ctx context.Context, _ string) error {
					assert.NoError(t, ctx.Err(), "cleanup ran with a cancelled context")
					return tc.deleteErr
				}),
			)

			_, err := service.AddMedia(ctx, 3, 7, uploadKey)

			assert.ErrorIs(t, err, tc.addErr)
		})
	}
}

func TestService_DeleteMedia(t *testing.T) {
	for _, repoErr := range []error{nil, domain.ErrMediaNotFound, errors.New("connection reset")} {
		// The repository queues the file for deletion, so the service must
		// not delete it itself.
		repo, _, service := newMediaMocks(t)
		repo.EXPECT().DeleteMedia(gomock.Any(), int64(3), int64(7), int64(4)).Return(repoErr)

		err := service.DeleteMedia(t.Context(), 3, 7, 4)

		assert.ErrorIs(t, err, repoErr)
	}
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

			got, err := reviews.NewService(repo, nil, nil, nil).ListByPlace(t.Context(), 3)
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
