package reports_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/domain"
	"rantau-go/internal/module/reports"
	"rantau-go/internal/module/reports/mock"
)

// txKey marks the context that newService's transactor passes to the
// transaction's function.
type txKey struct{}

// inTx matches a context carrying the transaction newService's transactor started.
var inTx = gomock.Cond(func(ctx context.Context) bool { return ctx.Value(txKey{}) != nil })

// collaborators are the mocks behind a Service built by newService.
type collaborators struct {
	repo    *mock.MockRepository
	places  *mock.MockPlaceService
	reviews *mock.MockReviewService
}

// newService returns a Service over mock collaborators. Its transactor runs
// each function it is given with a context marked by txKey, returning the
// function's error.
func newService(t *testing.T) (reports.Service, collaborators) {
	t.Helper()

	ctrl := gomock.NewController(t)
	c := collaborators{
		repo:    mock.NewMockRepository(ctrl),
		places:  mock.NewMockPlaceService(ctrl),
		reviews: mock.NewMockReviewService(ctrl),
	}
	tx := mock.NewMockTransactor(ctrl)
	tx.EXPECT().WithinTx(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(context.WithValue(ctx, txKey{}, true))
	}).AnyTimes()

	return reports.NewService(c.repo, tx, c.places, c.reviews), c
}

func TestService_Create_MaxPendingPerUser(t *testing.T) {
	assert.Equal(t, 5, reports.MaxPendingPerUser)
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name     string
		reason   domain.ReportReason
		note     *string
		wantNote *string
		wantErr  error
	}{
		{name: "spam", reason: domain.ReportReasonSpam},
		{name: "offensive", reason: domain.ReportReasonOffensive},
		{name: "fake", reason: domain.ReportReasonFake},
		{name: "other_with_note", reason: domain.ReportReasonOther, note: new("  Nomor palsu  "), wantNote: new("Nomor palsu")},
		{name: "blank_note_is_nothing", reason: domain.ReportReasonSpam, note: new("   ")},
		{name: "note_at_limit", reason: domain.ReportReasonSpam, note: new(strings.Repeat("é", 1000)), wantNote: new(strings.Repeat("é", 1000))},
		{name: "note_over_limit", reason: domain.ReportReasonSpam, note: new(strings.Repeat("é", 1001)), wantErr: domain.ErrReportNoteTooLong},
		{name: "missing_reason", reason: "", wantErr: domain.ErrReportReasonInvalid},
		{name: "unknown_reason", reason: "rude", wantErr: domain.ErrReportReasonInvalid},
		{name: "reason_wrong_case", reason: "Spam", wantErr: domain.ErrReportReasonInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc, c := newService(t)
			input := &domain.Report{UserID: 7, PlaceID: 4, Reason: tc.reason, Note: tc.note}
			if tc.wantErr == nil {
				want := &domain.Report{UserID: 7, PlaceID: 4, Reason: tc.reason, Note: tc.wantNote}
				c.repo.EXPECT().Create(gomock.Any(), want).Return(want, nil)
			}

			_, err := svc.Create(t.Context(), input)
			assert.ErrorIs(t, err, tc.wantErr)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("passes_repository_errors", func(t *testing.T) {
		svc, c := newService(t)
		c.repo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, domain.ErrAlreadyReported)

		_, err := svc.Create(t.Context(), &domain.Report{UserID: 7, PlaceID: 4, Reason: domain.ReportReasonSpam})
		assert.ErrorIs(t, err, domain.ErrAlreadyReported)
	})
}

func TestService_List(t *testing.T) {
	for _, status := range []domain.ReportStatus{"", domain.ReportStatusPending, domain.ReportStatusRemoved, domain.ReportStatusDismissed} {
		t.Run("status_"+string(status), func(t *testing.T) {
			svc, c := newService(t)
			c.repo.EXPECT().List(gomock.Any(), reports.Filter{Status: status}).Return(nil, nil)

			_, err := svc.List(t.Context(), reports.Filter{Status: status})
			assert.NoError(t, err)
		})
	}

	t.Run("unknown_status", func(t *testing.T) {
		svc, _ := newService(t)

		_, err := svc.List(t.Context(), reports.Filter{Status: "approved"})
		assert.ErrorIs(t, err, domain.ErrReportStatusInvalid)
	})
}

func TestService_Fetch(t *testing.T) {
	place := &domain.Place{ID: 4, Name: "Warung"}
	review := &domain.Review{ID: 9, PlaceID: 4, Rating: 1}

	t.Run("place_report_loads_the_place", func(t *testing.T) {
		svc, c := newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Report{ID: 3, PlaceID: 4}, nil)
		c.places.EXPECT().Fetch(gomock.Any(), int64(4)).Return(place, nil)

		got, err := svc.Fetch(t.Context(), 3)
		require.NoError(t, err)
		assert.Equal(t, place, got.Place)
		assert.Nil(t, got.Review)
	})

	t.Run("review_report_loads_the_place_and_review", func(t *testing.T) {
		svc, c := newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Report{ID: 3, PlaceID: 4, ReviewID: new(int64(9))}, nil)
		c.places.EXPECT().Fetch(gomock.Any(), int64(4)).Return(place, nil)
		c.reviews.EXPECT().FetchByID(gomock.Any(), int64(4), int64(9)).Return(review, nil)

		got, err := svc.Fetch(t.Context(), 3)
		require.NoError(t, err)
		assert.Equal(t, place, got.Place)
		assert.Equal(t, review, got.Review)
	})

	t.Run("deleted_content_is_nil", func(t *testing.T) {
		svc, c := newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Report{ID: 3, PlaceID: 4, ReviewID: new(int64(9))}, nil)
		c.places.EXPECT().Fetch(gomock.Any(), int64(4)).Return(nil, domain.ErrPlaceNotFound)
		c.reviews.EXPECT().FetchByID(gomock.Any(), int64(4), int64(9)).Return(nil, domain.ErrReviewNotFound)

		got, err := svc.Fetch(t.Context(), 3)
		require.NoError(t, err)
		assert.Nil(t, got.Place)
		assert.Nil(t, got.Review)
	})

	t.Run("passes_other_errors", func(t *testing.T) {
		boom := errors.New("boom")

		svc, c := newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(nil, domain.ErrReportNotFound)
		_, err := svc.Fetch(t.Context(), 3)
		assert.ErrorIs(t, err, domain.ErrReportNotFound)

		svc, c = newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Report{ID: 3, PlaceID: 4}, nil)
		c.places.EXPECT().Fetch(gomock.Any(), int64(4)).Return(nil, boom)
		_, err = svc.Fetch(t.Context(), 3)
		assert.ErrorIs(t, err, boom)

		svc, c = newService(t)
		c.repo.EXPECT().Fetch(gomock.Any(), int64(3)).Return(&domain.Report{ID: 3, PlaceID: 4, ReviewID: new(int64(9))}, nil)
		c.places.EXPECT().Fetch(gomock.Any(), int64(4)).Return(place, nil)
		c.reviews.EXPECT().FetchByID(gomock.Any(), int64(4), int64(9)).Return(nil, boom)
		_, err = svc.Fetch(t.Context(), 3)
		assert.ErrorIs(t, err, boom)
	})
}

func TestService_Remove(t *testing.T) {
	placeReport := &domain.Report{ID: 3, PlaceID: 4, Status: domain.ReportStatusRemoved}
	reviewReport := &domain.Report{ID: 3, PlaceID: 4, ReviewID: new(int64(9)), Status: domain.ReportStatusRemoved}

	t.Run("place", func(t *testing.T) {
		svc, c := newService(t)
		gomock.InOrder(
			c.repo.EXPECT().Resolve(inTx, int64(3), int64(1), domain.ReportStatusRemoved).Return(placeReport, nil),
			c.places.EXPECT().Delete(inTx, int64(4)).Return(nil),
			c.repo.EXPECT().ResolveContent(inTx, int64(4), nil, int64(1)).Return(nil),
		)

		got, err := svc.Remove(t.Context(), 3, 1)
		require.NoError(t, err)
		assert.Equal(t, placeReport, got)
	})

	t.Run("review", func(t *testing.T) {
		svc, c := newService(t)
		gomock.InOrder(
			c.repo.EXPECT().Resolve(inTx, int64(3), int64(1), domain.ReportStatusRemoved).Return(reviewReport, nil),
			c.reviews.EXPECT().Delete(inTx, int64(4), int64(9)).Return(nil),
			c.repo.EXPECT().ResolveContent(inTx, int64(4), new(int64(9)), int64(1)).Return(nil),
		)

		got, err := svc.Remove(t.Context(), 3, 1)
		require.NoError(t, err)
		assert.Equal(t, reviewReport, got)
	})

	t.Run("content_already_gone_still_resolves", func(t *testing.T) {
		tests := []struct {
			name   string
			report *domain.Report
			err    error
		}{
			{name: "place", report: placeReport, err: domain.ErrPlaceNotFound},
			{name: "review", report: reviewReport, err: domain.ErrReviewNotFound},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				svc, c := newService(t)
				c.repo.EXPECT().Resolve(inTx, int64(3), int64(1), domain.ReportStatusRemoved).Return(tc.report, nil)
				if tc.report.ReviewID == nil {
					c.places.EXPECT().Delete(inTx, int64(4)).Return(tc.err)
				} else {
					c.reviews.EXPECT().Delete(inTx, int64(4), int64(9)).Return(tc.err)
				}
				c.repo.EXPECT().ResolveContent(inTx, int64(4), tc.report.ReviewID, int64(1)).Return(nil)

				_, err := svc.Remove(t.Context(), 3, 1)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("failed_delete_fails_the_transaction", func(t *testing.T) {
		svc, c := newService(t)
		boom := errors.New("boom")
		c.repo.EXPECT().Resolve(inTx, int64(3), int64(1), domain.ReportStatusRemoved).Return(placeReport, nil)
		c.places.EXPECT().Delete(inTx, int64(4)).Return(boom)

		got, err := svc.Remove(t.Context(), 3, 1)
		assert.ErrorIs(t, err, boom)
		assert.Nil(t, got)
	})

	t.Run("unresolvable_report_deletes_nothing", func(t *testing.T) {
		for _, want := range []error{domain.ErrReportNotFound, domain.ErrReportAlreadyResolved} {
			svc, c := newService(t)
			c.repo.EXPECT().Resolve(inTx, int64(3), int64(1), domain.ReportStatusRemoved).Return(nil, want)

			_, err := svc.Remove(t.Context(), 3, 1)
			assert.ErrorIs(t, err, want)
		}
	})
}

func TestService_Dismiss(t *testing.T) {
	svc, c := newService(t)
	dismissed := &domain.Report{ID: 3, PlaceID: 4, Status: domain.ReportStatusDismissed}
	c.repo.EXPECT().Resolve(gomock.Any(), int64(3), int64(1), domain.ReportStatusDismissed).Return(dismissed, nil)

	got, err := svc.Dismiss(t.Context(), 3, 1)
	require.NoError(t, err)
	assert.Equal(t, dismissed, got)
}
