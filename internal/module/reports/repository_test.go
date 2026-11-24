//go:build integration

package reports_test

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
	"sakukota-go/internal/module/reports"
	"sakukota-go/internal/testutil/pgtest"
)

// testDB is shared by the tests in this package. pgtest.Reset empties it
// before each test.
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

// fixture is a database with a place, a review of it, a reporting user, and
// an admin.
type fixture struct {
	repo     *reports.PostgresRepository
	placeID  int64
	reviewID int64
	userID   int64
	adminID  int64
}

// newFixture returns a repository over an emptied database and the rows to
// report.
func newFixture(t *testing.T) fixture {
	t.Helper()
	pgtest.Reset(t, testDB)

	placeID := insertPlace(t, "Warung")
	return fixture{
		repo:     reports.NewPostgresRepository(testDB),
		placeID:  placeID,
		reviewID: insertReview(t, placeID, pgtest.InsertUser(t, testDB, "Reviewer")),
		userID:   pgtest.InsertUser(t, testDB, "Ana"),
		adminID:  pgtest.InsertUser(t, testDB, "Admin"),
	}
}

// insertPlace inserts a place in a new sub-district and returns its id.
func insertPlace(t *testing.T, name string) int64 {
	t.Helper()
	subDistrictID := pgtest.InsertSubDistrict(t, testDB, "SD "+name)
	return pgtest.InsertPlace(t, testDB, subDistrictID, pgtest.CategoryID(t, testDB, "laundry"), name)
}

// insertReview inserts the user's 4-star review of the place, adds it to the
// place's rating totals, and returns its id.
func insertReview(t *testing.T, placeID, userID int64) int64 {
	t.Helper()
	var id int64
	err := testDB.Get(&id, `INSERT INTO PLACE_REVIEWS (place_id, user_id, rating) VALUES ($1, $2, 4) RETURNING id`, placeID, userID)
	require.NoError(t, err, "insert review")
	_, err = testDB.Exec(`UPDATE PLACES SET rating_count = rating_count + 1, rating_sum = rating_sum + 4 WHERE id = $1`, placeID)
	require.NoError(t, err, "add review rating")
	return id
}

// report files the user's spam report of the place, or of its review when
// reviewID is not nil.
func (f fixture) report(t *testing.T, userID, placeID int64, reviewID *int64) *domain.Report {
	t.Helper()
	created, err := f.repo.Create(t.Context(), &domain.Report{UserID: userID, PlaceID: placeID, ReviewID: reviewID, Reason: domain.ReportReasonSpam})
	require.NoError(t, err, "Create(place %d, review %v)", placeID, reviewID)
	return created
}

// statuses returns the status of every report by id.
func statuses(t *testing.T) map[int64]domain.ReportStatus {
	t.Helper()
	var rows []struct {
		ID     int64               `db:"id"`
		Status domain.ReportStatus `db:"status"`
	}
	require.NoError(t, testDB.Select(&rows, `SELECT id, status FROM REPORTS`))

	out := make(map[int64]domain.ReportStatus, len(rows))
	for _, r := range rows {
		out[r.ID] = r.Status
	}
	return out
}

func ids(reports []domain.Report) []int64 {
	out := make([]int64, 0, len(reports))
	for _, r := range reports {
		out = append(out, r.ID)
	}
	return out
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Run("stores_a_pending_place_report_with_its_author", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.Create(t.Context(), &domain.Report{UserID: f.userID, PlaceID: f.placeID, Reason: domain.ReportReasonFake, Note: new("Tutup")})
		require.NoError(t, err)

		assert.NotZero(t, got.ID)
		assert.Equal(t, f.placeID, got.PlaceID)
		assert.Nil(t, got.ReviewID)
		assert.Equal(t, domain.ReportReasonFake, got.Reason)
		assert.Equal(t, new("Tutup"), got.Note)
		assert.Equal(t, domain.ReportStatusPending, got.Status)
		assert.Nil(t, got.ResolvedBy)
		assert.Nil(t, got.ResolvedAt)
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.Equal(t, domain.Author{ID: f.userID, Name: "Ana"}, got.User)
	})

	t.Run("stores_a_review_report", func(t *testing.T) {
		f := newFixture(t)

		got := f.report(t, f.userID, f.placeID, &f.reviewID)
		assert.Equal(t, f.placeID, got.PlaceID)
		assert.Equal(t, &f.reviewID, got.ReviewID)
	})

	t.Run("unknown_targets", func(t *testing.T) {
		f := newFixture(t)
		otherPlaceID := insertPlace(t, "Other")
		unknown := int64(9999)

		tests := []struct {
			name     string
			placeID  int64
			reviewID *int64
			want     error
		}{
			{name: "unknown_place", placeID: 9999, want: domain.ErrPlaceNotFound},
			{name: "unknown_review", placeID: f.placeID, reviewID: &unknown, want: domain.ErrReviewNotFound},
			{name: "review_of_another_place", placeID: otherPlaceID, reviewID: &f.reviewID, want: domain.ErrReviewNotFound},
			{name: "review_of_unknown_place", placeID: 9999, reviewID: &f.reviewID, want: domain.ErrReviewNotFound},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := f.repo.Create(t.Context(), &domain.Report{UserID: f.userID, PlaceID: tc.placeID, ReviewID: tc.reviewID, Reason: domain.ReportReasonSpam})
				assert.ErrorIs(t, err, tc.want)
			})
		}
		assert.Empty(t, statuses(t))
	})

	t.Run("one_pending_report_per_user_and_content", func(t *testing.T) {
		f := newFixture(t)
		placeReport := f.report(t, f.userID, f.placeID, nil)
		f.report(t, f.userID, f.placeID, &f.reviewID)

		for _, reviewID := range []*int64{nil, &f.reviewID} {
			_, err := f.repo.Create(t.Context(), &domain.Report{UserID: f.userID, PlaceID: f.placeID, ReviewID: reviewID, Reason: domain.ReportReasonOther})
			assert.ErrorIs(t, err, domain.ErrAlreadyReported, "review %v", reviewID)
		}

		// Another user may report the same content.
		f.report(t, f.adminID, f.placeID, nil)

		// Once resolved, the content can be reported again.
		_, err := f.repo.Resolve(t.Context(), placeReport.ID, f.adminID, domain.ReportStatusDismissed)
		require.NoError(t, err)
		f.report(t, f.userID, f.placeID, nil)
	})

	t.Run("pending_limit", func(t *testing.T) {
		f := newFixture(t)
		placeIDs := make([]int64, reports.MaxPendingPerUser+1)
		for i := range placeIDs {
			placeIDs[i] = insertPlace(t, fmt.Sprintf("Place%d", i))
		}

		for _, id := range placeIDs[:reports.MaxPendingPerUser] {
			f.report(t, f.userID, id, nil)
		}
		_, err := f.repo.Create(t.Context(), &domain.Report{UserID: f.userID, PlaceID: placeIDs[reports.MaxPendingPerUser], Reason: domain.ReportReasonSpam})
		assert.ErrorIs(t, err, domain.ErrReportLimitReached)

		// A resolved report frees a slot.
		first, err := f.repo.List(t.Context(), reports.Filter{})
		require.NoError(t, err)
		_, err = f.repo.Resolve(t.Context(), first[0].ID, f.adminID, domain.ReportStatusDismissed)
		require.NoError(t, err)
		f.report(t, f.userID, placeIDs[reports.MaxPendingPerUser], nil)
	})

	t.Run("concurrent_reports_respect_the_limit_and_uniqueness", func(t *testing.T) {
		f := newFixture(t)
		const attempts = 10
		placeIDs := make([]int64, attempts)
		for i := range placeIDs {
			placeIDs[i] = insertPlace(t, fmt.Sprintf("Place%d", i))
		}

		var wg sync.WaitGroup
		limitErrs := make([]error, attempts)
		dupErrs := make([]error, attempts)
		for i := range attempts {
			wg.Go(func() {
				_, limitErrs[i] = f.repo.Create(t.Context(), &domain.Report{UserID: f.userID, PlaceID: placeIDs[i], Reason: domain.ReportReasonSpam})
			})
			wg.Go(func() {
				_, dupErrs[i] = f.repo.Create(t.Context(), &domain.Report{UserID: f.adminID, PlaceID: f.placeID, Reason: domain.ReportReasonSpam})
			})
		}
		wg.Wait()

		count := func(errs []error, want error) (ok, refused int) {
			for _, err := range errs {
				switch {
				case err == nil:
					ok++
				case assert.ErrorIs(t, err, want):
					refused++
				}
			}
			return ok, refused
		}
		ok, refused := count(limitErrs, domain.ErrReportLimitReached)
		assert.Equal(t, [2]int{reports.MaxPendingPerUser, attempts - reports.MaxPendingPerUser}, [2]int{ok, refused})
		ok, refused = count(dupErrs, domain.ErrAlreadyReported)
		assert.Equal(t, [2]int{1, attempts - 1}, [2]int{ok, refused})
	})
}

func TestPostgresRepository_List(t *testing.T) {
	f := newFixture(t)
	first := f.report(t, f.userID, f.placeID, nil)
	second := f.report(t, f.userID, f.placeID, &f.reviewID)
	third := f.report(t, f.adminID, f.placeID, nil)
	_, err := testDB.Exec(`UPDATE REPORTS SET created_at = '2026-01-01' WHERE id = $1`, third.ID)
	require.NoError(t, err)
	_, err = f.repo.Resolve(t.Context(), second.ID, f.adminID, domain.ReportStatusDismissed)
	require.NoError(t, err)

	tests := []struct {
		status domain.ReportStatus
		want   []int64
	}{
		{status: "", want: []int64{third.ID, first.ID, second.ID}},
		{status: domain.ReportStatusPending, want: []int64{third.ID, first.ID}},
		{status: domain.ReportStatusDismissed, want: []int64{second.ID}},
		{status: domain.ReportStatusRemoved, want: []int64{}},
	}
	for _, tc := range tests {
		t.Run("status_"+string(tc.status), func(t *testing.T) {
			got, err := f.repo.List(t.Context(), reports.Filter{Status: tc.status})
			require.NoError(t, err)
			assert.Equal(t, tc.want, ids(got))
		})
	}
}

func TestPostgresRepository_Fetch(t *testing.T) {
	f := newFixture(t)
	created := f.report(t, f.userID, f.placeID, &f.reviewID)

	got, err := f.repo.Fetch(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, got)

	_, err = f.repo.Fetch(t.Context(), created.ID+1)
	assert.ErrorIs(t, err, domain.ErrReportNotFound)
}

func TestPostgresRepository_Resolve(t *testing.T) {
	t.Run("marks_the_report_with_status_and_admin", func(t *testing.T) {
		f := newFixture(t)
		r := f.report(t, f.userID, f.placeID, nil)

		got, err := f.repo.Resolve(t.Context(), r.ID, f.adminID, domain.ReportStatusDismissed)
		require.NoError(t, err)
		assert.Equal(t, domain.ReportStatusDismissed, got.Status)
		assert.Equal(t, &f.adminID, got.ResolvedBy)
		require.NotNil(t, got.ResolvedAt)
		assert.WithinDuration(t, time.Now(), *got.ResolvedAt, time.Minute)
		assert.Equal(t, domain.Author{ID: f.userID, Name: "Ana"}, got.User)
	})

	t.Run("unknown_or_resolved", func(t *testing.T) {
		f := newFixture(t)
		r := f.report(t, f.userID, f.placeID, nil)
		_, err := f.repo.Resolve(t.Context(), r.ID, f.adminID, domain.ReportStatusRemoved)
		require.NoError(t, err)

		_, err = f.repo.Resolve(t.Context(), r.ID, f.adminID, domain.ReportStatusDismissed)
		assert.ErrorIs(t, err, domain.ErrReportAlreadyResolved)
		_, err = f.repo.Resolve(t.Context(), 9999, f.adminID, domain.ReportStatusDismissed)
		assert.ErrorIs(t, err, domain.ErrReportNotFound)
		assert.Equal(t, domain.ReportStatusRemoved, statuses(t)[r.ID])
	})
}

func TestPostgresRepository_ResolveContent(t *testing.T) {
	t.Run("place_resolves_its_reports_and_its_reviews_reports", func(t *testing.T) {
		f := newFixture(t)
		otherPlaceID := insertPlace(t, "Other")
		placeReport := f.report(t, f.userID, f.placeID, nil)
		reviewReport := f.report(t, f.adminID, f.placeID, &f.reviewID)
		other := f.report(t, f.userID, otherPlaceID, nil)

		require.NoError(t, f.repo.ResolveContent(t.Context(), f.placeID, nil, f.adminID))
		assert.Equal(t, map[int64]domain.ReportStatus{
			placeReport.ID:  domain.ReportStatusRemoved,
			reviewReport.ID: domain.ReportStatusRemoved,
			other.ID:        domain.ReportStatusPending,
		}, statuses(t))
	})

	t.Run("review_resolves_only_its_reports", func(t *testing.T) {
		f := newFixture(t)
		otherReviewID := insertReview(t, f.placeID, f.adminID)
		placeReport := f.report(t, f.userID, f.placeID, nil)
		reviewReport := f.report(t, f.userID, f.placeID, &f.reviewID)
		otherReview := f.report(t, f.userID, f.placeID, &otherReviewID)

		require.NoError(t, f.repo.ResolveContent(t.Context(), f.placeID, &f.reviewID, f.adminID))
		assert.Equal(t, map[int64]domain.ReportStatus{
			placeReport.ID:  domain.ReportStatusPending,
			reviewReport.ID: domain.ReportStatusRemoved,
			otherReview.ID:  domain.ReportStatusPending,
		}, statuses(t))
	})

	t.Run("resolved_reports_keep_their_status", func(t *testing.T) {
		f := newFixture(t)
		r := f.report(t, f.userID, f.placeID, nil)
		_, err := f.repo.Resolve(t.Context(), r.ID, f.adminID, domain.ReportStatusDismissed)
		require.NoError(t, err)

		require.NoError(t, f.repo.ResolveContent(t.Context(), f.placeID, nil, f.adminID))
		assert.Equal(t, domain.ReportStatusDismissed, statuses(t)[r.ID])
	})
}

func TestPostgresRepository_HistoryOutlivesContent(t *testing.T) {
	f := newFixture(t)
	placeReport := f.report(t, f.userID, f.placeID, nil)
	reviewReport := f.report(t, f.userID, f.placeID, &f.reviewID)

	_, err := testDB.Exec(`DELETE FROM PLACES WHERE id = $1`, f.placeID)
	require.NoError(t, err)

	got, err := f.repo.List(t.Context(), reports.Filter{})
	require.NoError(t, err)
	assert.Equal(t, []int64{placeReport.ID, reviewReport.ID}, ids(got))
	assert.Equal(t, &f.reviewID, got[1].ReviewID)
}

func TestPostgresRepository_DeletingTheReporterDeletesTheirReports(t *testing.T) {
	f := newFixture(t)
	f.report(t, f.userID, f.placeID, nil)
	kept := f.report(t, f.adminID, f.placeID, nil)

	_, err := testDB.Exec(`DELETE FROM USERS WHERE id = $1`, f.userID)
	require.NoError(t, err)

	got, err := f.repo.List(t.Context(), reports.Filter{})
	require.NoError(t, err)
	assert.Equal(t, []int64{kept.ID}, ids(got))
}
