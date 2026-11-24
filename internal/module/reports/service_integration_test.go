//go:build integration

package reports_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/reports"
	"sakukota-go/internal/module/reports/mock"
	"sakukota-go/internal/module/reviews"
)

// newWiredService returns a Service over the fixture's database, wired as in
// production: content is removed through the place and review services in
// the report's transaction.
func newWiredService(f fixture) reports.Service {
	placeService := places.NewService(places.NewPostgresRepository(testDB), nil)
	reviewService := reviews.NewService(reviews.NewPostgresRepository(testDB), database.NewTransactor(testDB), placeService, nil)
	return reports.NewService(f.repo, database.NewTransactor(testDB), placeService, reviewService)
}

// exists reports whether the query finds a row with the id.
func exists(t *testing.T, query string, id int64) bool {
	t.Helper()
	var found bool
	require.NoError(t, testDB.Get(&found, `SELECT EXISTS (`+query+`)`, id))
	return found
}

// ratingCount returns the place's rating_count.
func ratingCount(t *testing.T, placeID int64) int {
	t.Helper()
	var count int
	require.NoError(t, testDB.Get(&count, `SELECT rating_count FROM PLACES WHERE id = $1`, placeID))
	return count
}

func TestService_Fetch_Wired(t *testing.T) {
	t.Run("review_report_with_its_place_and_review", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		r := f.report(t, f.userID, f.placeID, &f.reviewID)

		got, err := svc.Fetch(t.Context(), r.ID)
		require.NoError(t, err)
		require.NotNil(t, got.Place)
		assert.Equal(t, "Warung", got.Place.Name)
		require.NotNil(t, got.Review)
		assert.Equal(t, f.reviewID, got.Review.ID)
		assert.Equal(t, "Reviewer", got.Review.User.Name)
	})

	t.Run("removed_content_is_nil", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		r := f.report(t, f.userID, f.placeID, &f.reviewID)
		_, err := svc.Remove(t.Context(), r.ID, f.adminID)
		require.NoError(t, err)

		got, err := svc.Fetch(t.Context(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReportStatusRemoved, got.Status)
		assert.NotNil(t, got.Place, "the place stays when only its review is removed")
		assert.Nil(t, got.Review)
	})
}

func TestService_Remove_Wired(t *testing.T) {
	t.Run("place", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		placeReport := f.report(t, f.userID, f.placeID, nil)
		reviewReport := f.report(t, f.adminID, f.placeID, &f.reviewID)

		got, err := svc.Remove(t.Context(), placeReport.ID, f.adminID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReportStatusRemoved, got.Status)

		assert.False(t, exists(t, `SELECT 1 FROM PLACES WHERE id = $1`, f.placeID))
		assert.Equal(t, map[int64]domain.ReportStatus{
			placeReport.ID:  domain.ReportStatusRemoved,
			reviewReport.ID: domain.ReportStatusRemoved,
		}, statuses(t))
	})

	t.Run("review_takes_its_rating_off_the_place", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		placeReport := f.report(t, f.userID, f.placeID, nil)
		reviewReport := f.report(t, f.userID, f.placeID, &f.reviewID)

		_, err := svc.Remove(t.Context(), reviewReport.ID, f.adminID)
		require.NoError(t, err)

		assert.False(t, exists(t, `SELECT 1 FROM PLACE_REVIEWS WHERE id = $1`, f.reviewID))
		assert.Equal(t, 0, ratingCount(t, f.placeID))
		assert.Equal(t, domain.ReportStatusPending, statuses(t)[placeReport.ID])
	})

	t.Run("content_deleted_elsewhere", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		r := f.report(t, f.userID, f.placeID, &f.reviewID)
		_, err := testDB.Exec(`DELETE FROM PLACE_REVIEWS WHERE id = $1`, f.reviewID)
		require.NoError(t, err)

		_, err = svc.Remove(t.Context(), r.ID, f.adminID)
		require.NoError(t, err)
		assert.Equal(t, domain.ReportStatusRemoved, statuses(t)[r.ID])
	})

	t.Run("failed_delete_keeps_the_report_pending", func(t *testing.T) {
		f := newFixture(t)
		failing := mock.NewMockPlaceService(gomock.NewController(t))
		boom := errors.New("boom")
		failing.EXPECT().Delete(gomock.Any(), f.placeID).Return(boom)
		svc := reports.NewService(f.repo, database.NewTransactor(testDB), failing, nil)
		r := f.report(t, f.userID, f.placeID, nil)

		_, err := svc.Remove(t.Context(), r.ID, f.adminID)
		assert.ErrorIs(t, err, boom)
		assert.Equal(t, domain.ReportStatusPending, statuses(t)[r.ID])
	})

	t.Run("concurrent_removes_take_the_rating_off_once", func(t *testing.T) {
		f := newFixture(t)
		svc := newWiredService(f)
		r := f.report(t, f.userID, f.placeID, &f.reviewID)

		const admins = 5
		var wg sync.WaitGroup
		errs := make([]error, admins)
		for i := range admins {
			wg.Go(func() { _, errs[i] = svc.Remove(t.Context(), r.ID, f.adminID) })
		}
		wg.Wait()

		var succeeded int
		for _, err := range errs {
			if err == nil {
				succeeded++
				continue
			}
			assert.ErrorIs(t, err, domain.ErrReportAlreadyResolved)
		}
		assert.Equal(t, 1, succeeded)
		assert.Equal(t, 0, ratingCount(t, f.placeID))
	})
}
