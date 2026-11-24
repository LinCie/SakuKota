//go:build integration

package suggestions_test

import (
	"context"
	"encoding/json"
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
	"sakukota-go/internal/module/suggestions"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/testutil/pgtest"
)

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

// fixture is a database with the rows a suggestion refers to: a sub-district,
// a category, a submitting user, and an admin.
type fixture struct {
	repo          *suggestions.PostgresRepository
	subDistrictID int64
	categoryID    int64
	userID        int64
	reviewerID    int64
}

// newFixture returns a repository over an emptied database and the rows to
// submit suggestions against.
func newFixture(t *testing.T) fixture {
	t.Helper()
	pgtest.Reset(t, testDB)

	return fixture{
		repo:          suggestions.NewPostgresRepository(testDB),
		subDistrictID: pgtest.InsertSubDistrict(t, testDB, "Setiabudi"),
		categoryID:    pgtest.CategoryID(t, testDB, "laundry"),
		userID:        pgtest.InsertUser(t, testDB, "Ana"),
		reviewerID:    pgtest.InsertUser(t, testDB, "Admin"),
	}
}

// newSuggestion returns a suggestion by the fixture's user that is valid to create.
func (f fixture) newSuggestion(name string) *domain.Suggestion {
	return &domain.Suggestion{
		UserID:        f.userID,
		SubDistrictID: f.subDistrictID,
		CategoryID:    f.categoryID,
		Name:          name,
		Location:      postgis.MustPoint(106.82, -6.2),
		Attributes:    json.RawMessage(`{}`),
	}
}

func (f fixture) create(t *testing.T, name string) *domain.Suggestion {
	t.Helper()

	created, err := f.repo.Create(t.Context(), f.newSuggestion(name))
	require.NoError(t, err, "Create(%q)", name)

	return created
}

// setCreatedAt moves the suggestion's creation time, to order suggestions
// without waiting.
func setCreatedAt(t *testing.T, id int64, createdAt string) {
	t.Helper()

	_, err := testDB.Exec(`UPDATE PLACE_SUGGESTIONS SET created_at = $2 WHERE id = $1`, id, createdAt)
	require.NoError(t, err)
}

func ids(suggestions []domain.Suggestion) []int64 {
	out := make([]int64, 0, len(suggestions))
	for _, s := range suggestions {
		out = append(out, s.ID)
	}

	return out
}

func TestPostgresRepository_Create(t *testing.T) {
	t.Run("stores the suggestion as pending and returns it with its author", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry Bersih")
		s.Address = new("Jl. Sudirman 1")
		s.OperatingHours = domain.OperatingHours{{Day: 1, Open: "08:00", Close: "20:00"}}
		s.PriceMin, s.PriceMax = new(5000), new(7000)
		s.ShortDescription = new("Cuci kilat")
		s.Attributes = json.RawMessage(`{"service_type": "Swalayan"}`)
		s.Note = new("Baru buka")

		got, err := f.repo.Create(t.Context(), s)
		require.NoError(t, err, "Create()")

		assert.NotZero(t, got.ID)
		assert.Equal(t, f.userID, got.UserID)
		assert.Equal(t, f.subDistrictID, got.SubDistrictID)
		assert.Equal(t, f.categoryID, got.CategoryID)
		assert.Equal(t, "Laundry Bersih", got.Name)
		assert.Equal(t, new("Jl. Sudirman 1"), got.Address)
		assert.Equal(t, domain.OperatingHours{{Day: 1, Open: "08:00", Close: "20:00"}}, got.OperatingHours)
		assert.Equal(t, new(5000), got.PriceMin)
		assert.Equal(t, new(7000), got.PriceMax)
		assert.Equal(t, new("Cuci kilat"), got.ShortDescription)
		assert.JSONEq(t, `{"service_type": "Swalayan"}`, string(got.Attributes))
		assert.Equal(t, new("Baru buka"), got.Note)
		require.NotNil(t, got.Location)
		assert.InDelta(t, 106.82, got.Location.X(), 1e-9)
		assert.InDelta(t, -6.2, got.Location.Y(), 1e-9)
		assert.Equal(t, domain.StatusPending, got.Status)
		assert.Nil(t, got.RejectionReason)
		assert.Nil(t, got.PlaceID)
		assert.Nil(t, got.ReviewedBy)
		assert.Nil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.Equal(t, domain.Author{ID: f.userID, Name: "Ana"}, got.User)

		stored, err := f.repo.Fetch(t.Context(), got.ID)
		require.NoError(t, err, "Fetch(%d)", got.ID)
		assert.Equal(t, got, stored)
	})

	t.Run("leaves optional fields empty", func(t *testing.T) {
		f := newFixture(t)

		got := f.create(t, "Laundry")

		assert.Nil(t, got.Address)
		assert.Nil(t, got.OperatingHours)
		assert.Nil(t, got.PriceMin)
		assert.Nil(t, got.PriceMax)
		assert.Nil(t, got.ShortDescription)
		assert.Nil(t, got.Note)
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown sub-district", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.SubDistrictID = 999

		got, err := f.repo.Create(t.Context(), s)

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrCategoryNotFound for an unknown category", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.CategoryID = 999

		got, err := f.repo.Create(t.Context(), s)

		assert.ErrorIs(t, err, domain.ErrCategoryNotFound)
		assert.Nil(t, got)
	})

	t.Run("fails for an unknown user", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.UserID = 999

		got, err := f.repo.Create(t.Context(), s)

		assert.True(t, database.IsCode(err, database.ForeignKeyViolation), "Create() error = %v, want a foreign key violation", err)
		assert.Nil(t, got)
	})

	t.Run("allows up to the pending limit and then returns ErrSuggestionLimitReached", func(t *testing.T) {
		f := newFixture(t)
		for i := range suggestions.MaxPendingPerUser {
			f.create(t, fmt.Sprintf("Laundry %d", i))
		}

		got, err := f.repo.Create(t.Context(), f.newSuggestion("One too many"))

		assert.ErrorIs(t, err, domain.ErrSuggestionLimitReached)
		assert.Nil(t, got)
		all, err := f.repo.ListByUser(t.Context(), f.userID)
		require.NoError(t, err, "ListByUser(%d)", f.userID)
		assert.Len(t, all, suggestions.MaxPendingPerUser, "the rejected one was not stored")
	})

	t.Run("counts only the user's own pending suggestions", func(t *testing.T) {
		f := newFixture(t)
		for i := range suggestions.MaxPendingPerUser {
			f.create(t, fmt.Sprintf("Laundry %d", i))
		}
		other := f.newSuggestion("Other user's")
		other.UserID = f.reviewerID

		_, err := f.repo.Create(t.Context(), other)
		assert.NoError(t, err)
	})

	t.Run("frees a slot when a pending suggestion is reviewed", func(t *testing.T) {
		reviews := map[string]func(t *testing.T, f fixture, id int64) error{
			"approved": func(t *testing.T, f fixture, id int64) error {
				placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")
				_, err := f.repo.MarkApproved(t.Context(), id, placeID, f.reviewerID)
				return err
			},
			"rejected": func(t *testing.T, f fixture, id int64) error {
				_, err := f.repo.Reject(t.Context(), id, f.reviewerID, nil)
				return err
			},
		}

		for name, review := range reviews {
			t.Run(name, func(t *testing.T) {
				f := newFixture(t)
				first := f.create(t, "Laundry 0")
				for i := 1; i < suggestions.MaxPendingPerUser; i++ {
					f.create(t, fmt.Sprintf("Laundry %d", i))
				}
				require.NoError(t, review(t, f, first.ID), "review suggestion %d", first.ID)

				_, err := f.repo.Create(t.Context(), f.newSuggestion("Replacement"))
				assert.NoError(t, err)
			})
		}
	})

	t.Run("holds the limit when the user submits at once", func(t *testing.T) {
		f := newFixture(t)
		const attempts = 12

		var wg sync.WaitGroup
		errs := make([]error, attempts)
		for i := range attempts {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.repo.Create(t.Context(), f.newSuggestion(fmt.Sprintf("Laundry %d", i)))
			}()
		}
		wg.Wait()

		var stored, tooMany int
		for _, err := range errs {
			if err == nil { // if NO error
				stored++
				continue
			}
			require.ErrorIs(t, err, domain.ErrSuggestionLimitReached)
			tooMany++
		}
		assert.Equal(t, suggestions.MaxPendingPerUser, stored, "stored suggestions")
		assert.Equal(t, attempts-suggestions.MaxPendingPerUser, tooMany, "refused suggestions")
	})
}

func TestPostgresRepository_Fetch(t *testing.T) {
	t.Run("returns the suggestion with its author", func(t *testing.T) {
		f := newFixture(t)
		f.create(t, "First")
		created := f.create(t, "Second")

		got, err := f.repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)

		assert.Equal(t, created, got)
		assert.Equal(t, "Ana", got.User.Name)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.Fetch(t.Context(), 999)

		assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})
}

func TestPostgresRepository_ListByUser(t *testing.T) {
	t.Run("returns only the user's suggestions, newest first", func(t *testing.T) {
		f := newFixture(t)
		oldest := f.create(t, "Oldest")
		newest := f.create(t, "Newest")
		other := f.newSuggestion("Other user's")
		other.UserID = f.reviewerID
		_, err := f.repo.Create(t.Context(), other)
		require.NoError(t, err)
		setCreatedAt(t, oldest.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, newest.ID, "2026-01-02T00:00:00Z")

		got, err := f.repo.ListByUser(t.Context(), f.userID)
		require.NoError(t, err, "ListByUser(%d)", f.userID)

		assert.Equal(t, []int64{newest.ID, oldest.ID}, ids(got))
		assert.Equal(t, "Ana", got[0].User.Name)
	})

	t.Run("lists the later of two simultaneous suggestions first", func(t *testing.T) {
		f := newFixture(t)
		first := f.create(t, "First")
		second := f.create(t, "Second")
		setCreatedAt(t, first.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, second.ID, "2026-01-01T00:00:00Z")

		got, err := f.repo.ListByUser(t.Context(), f.userID)
		require.NoError(t, err, "ListByUser(%d)", f.userID)

		assert.Equal(t, []int64{second.ID, first.ID}, ids(got))
	})

	t.Run("includes reviewed suggestions", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		_, err := f.repo.Reject(t.Context(), created.ID, f.reviewerID, nil)
		require.NoError(t, err)

		got, err := f.repo.ListByUser(t.Context(), f.userID)
		require.NoError(t, err, "ListByUser(%d)", f.userID)

		assert.Equal(t, []int64{created.ID}, ids(got))
	})

	t.Run("returns an empty, non-nil slice for a user without suggestions", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.ListByUser(t.Context(), f.reviewerID)
		require.NoError(t, err, "ListByUser(%d)", f.reviewerID)

		assert.Equal(t, []domain.Suggestion{}, got)
	})
}

func TestPostgresRepository_List(t *testing.T) {
	// Three suggestions, oldest first: one pending, one approved, one rejected.
	setup := func(t *testing.T) (f fixture, pendingID, approvedID, rejectedID int64) {
		f = newFixture(t)
		pendingID = f.create(t, "Pending").ID
		approvedID = f.create(t, "Approved").ID
		rejectedID = f.create(t, "Rejected").ID

		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Approved")
		_, err := f.repo.MarkApproved(t.Context(), approvedID, placeID, f.reviewerID)
		require.NoError(t, err)
		_, err = f.repo.Reject(t.Context(), rejectedID, f.reviewerID, nil)
		require.NoError(t, err)

		setCreatedAt(t, pendingID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, approvedID, "2026-01-02T00:00:00Z")
		setCreatedAt(t, rejectedID, "2026-01-03T00:00:00Z")

		return f, pendingID, approvedID, rejectedID
	}

	t.Run("returns every suggestion, oldest first, without a filter", func(t *testing.T) {
		f, pendingID, approvedID, rejectedID := setup(t)

		got, err := f.repo.List(t.Context(), suggestions.Filter{})
		require.NoError(t, err, "List()")

		assert.Equal(t, []int64{pendingID, approvedID, rejectedID}, ids(got))
		assert.Equal(t, "Ana", got[0].User.Name)
	})

	t.Run("keeps only the suggestions in the status", func(t *testing.T) {
		f, pendingID, approvedID, rejectedID := setup(t)

		tests := []struct {
			status domain.Status
			want   []int64
		}{
			{domain.StatusPending, []int64{pendingID}},
			{domain.StatusApproved, []int64{approvedID}},
			{domain.StatusRejected, []int64{rejectedID}},
		}
		for _, tc := range tests {
			got, err := f.repo.List(t.Context(), suggestions.Filter{Status: tc.status})
			require.NoError(t, err, "List(%q)", tc.status)

			assert.Equal(t, tc.want, ids(got))
		}
	})

	t.Run("lists the earlier of two simultaneous suggestions first", func(t *testing.T) {
		f := newFixture(t)
		first := f.create(t, "First")
		second := f.create(t, "Second")
		setCreatedAt(t, first.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, second.ID, "2026-01-01T00:00:00Z")

		got, err := f.repo.List(t.Context(), suggestions.Filter{})
		require.NoError(t, err, "List()")

		assert.Equal(t, []int64{first.ID, second.ID}, ids(got))
	})

	t.Run("returns an empty, non-nil slice when nothing matches", func(t *testing.T) {
		f := newFixture(t)
		f.create(t, "Pending")

		got, err := f.repo.List(t.Context(), suggestions.Filter{Status: domain.StatusRejected})
		require.NoError(t, err, "List()")

		assert.Equal(t, []domain.Suggestion{}, got)
	})
}

func TestPostgresRepository_MarkApproved(t *testing.T) {
	t.Run("marks a pending suggestion approved as the place", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")

		got, err := f.repo.MarkApproved(t.Context(), created.ID, placeID, f.reviewerID)
		require.NoError(t, err, "MarkApproved()")

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, domain.StatusApproved, got.Status)
		assert.Equal(t, new(placeID), got.PlaceID)
		assert.Equal(t, new(f.reviewerID), got.ReviewedBy)
		require.NotNil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), *got.ReviewedAt, time.Minute)
		assert.Nil(t, got.RejectionReason)
		assert.Equal(t, domain.Author{ID: f.userID, Name: "Ana"}, got.User)
		assert.Equal(t, created.Name, got.Name)

		stored, err := f.repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, got, stored)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.MarkApproved(t.Context(), 999, 1, f.reviewerID)

		assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrSuggestionAlreadyReviewed for a reviewed suggestion and keeps its outcome", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		rejected, err := f.repo.Reject(t.Context(), created.ID, f.reviewerID, new("Duplikat"))
		require.NoError(t, err, "Reject(%d)", created.ID)
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")

		got, err := f.repo.MarkApproved(t.Context(), created.ID, placeID, f.reviewerID)

		assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyReviewed)
		assert.Nil(t, got)
		stored, err := f.repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, rejected, stored)
	})

	t.Run("lets only one of several admins approve at once", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")
		const admins = 8

		var wg sync.WaitGroup
		errs := make([]error, admins)
		for i := range admins {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, errs[i] = f.repo.MarkApproved(t.Context(), created.ID, placeID, f.reviewerID)
			}()
		}
		wg.Wait()

		var approved int
		for _, err := range errs {
			if err == nil { // if NO error
				approved++
				continue
			}
			assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyReviewed)
		}
		assert.Equal(t, 1, approved, "approvals")
	})
}

func TestPostgresRepository_Reject(t *testing.T) {
	t.Run("marks a pending suggestion rejected with the reason", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")

		got, err := f.repo.Reject(t.Context(), created.ID, f.reviewerID, new("Sudah ada"))
		require.NoError(t, err, "Reject()")

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, domain.StatusRejected, got.Status)
		assert.Equal(t, new("Sudah ada"), got.RejectionReason)
		assert.Equal(t, new(f.reviewerID), got.ReviewedBy)
		require.NotNil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), *got.ReviewedAt, time.Minute)
		assert.Nil(t, got.PlaceID)
		assert.Equal(t, domain.Author{ID: f.userID, Name: "Ana"}, got.User)

		stored, err := f.repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, got, stored)
	})

	t.Run("accepts a rejection without a reason", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")

		got, err := f.repo.Reject(t.Context(), created.ID, f.reviewerID, nil)
		require.NoError(t, err, "Reject()")

		assert.Equal(t, domain.StatusRejected, got.Status)
		assert.Nil(t, got.RejectionReason)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.Reject(t.Context(), 999, f.reviewerID, nil)

		assert.ErrorIs(t, err, domain.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrSuggestionAlreadyReviewed for a reviewed suggestion and keeps its outcome", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")
		approved, err := f.repo.MarkApproved(t.Context(), created.ID, placeID, f.reviewerID)
		require.NoError(t, err, "MarkApproved(%d)", created.ID)

		got, err := f.repo.Reject(t.Context(), created.ID, f.reviewerID, new("Too late"))

		assert.ErrorIs(t, err, domain.ErrSuggestionAlreadyReviewed)
		assert.Nil(t, got)
		stored, err := f.repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, approved, stored)
	})
}
