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

	"rantau-go/internal/database"
	"rantau-go/internal/module/places"
	"rantau-go/internal/module/suggestions"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/testutil/pgtest"
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
	repo          suggestions.Repository
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
func (f fixture) newSuggestion(name string) *suggestions.Suggestion {
	return &suggestions.Suggestion{
		UserID:        f.userID,
		SubDistrictID: f.subDistrictID,
		CategoryID:    f.categoryID,
		Name:          name,
		Location:      postgis.MustPoint(106.82, -6.2),
		Attributes:    json.RawMessage(`{}`),
	}
}

func (f fixture) create(t *testing.T, name string) *suggestions.Suggestion {
	t.Helper()

	created, err := f.repo.Create(context.Background(), f.newSuggestion(name))
	require.NoError(t, err)

	return created
}

// setCreatedAt moves the suggestion's creation time, to order suggestions
// without waiting.
func setCreatedAt(t *testing.T, id int64, createdAt string) {
	t.Helper()

	_, err := testDB.Exec(`UPDATE PLACE_SUGGESTIONS SET created_at = $2 WHERE id = $1`, id, createdAt)
	require.NoError(t, err)
}

func ids(suggestions []suggestions.Suggestion) []int64 {
	out := make([]int64, 0, len(suggestions))
	for _, s := range suggestions {
		out = append(out, s.ID)
	}

	return out
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the suggestion as pending and returns it with its author", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry Bersih")
		s.Address = new("Jl. Sudirman 1")
		s.OperatingHours = new("08:00-20:00")
		s.EstimatedPriceRange = new("Rp5.000")
		s.ShortDescription = new("Cuci kilat")
		s.Attributes = json.RawMessage(`{"service_type": "Swalayan"}`)
		s.Note = new("Baru buka")

		got, err := f.repo.Create(ctx, s)

		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, f.userID, got.UserID)
		assert.Equal(t, f.subDistrictID, got.SubDistrictID)
		assert.Equal(t, f.categoryID, got.CategoryID)
		assert.Equal(t, "Laundry Bersih", got.Name)
		assert.Equal(t, new("Jl. Sudirman 1"), got.Address)
		assert.Equal(t, new("08:00-20:00"), got.OperatingHours)
		assert.Equal(t, new("Rp5.000"), got.EstimatedPriceRange)
		assert.Equal(t, new("Cuci kilat"), got.ShortDescription)
		assert.JSONEq(t, `{"service_type": "Swalayan"}`, string(got.Attributes))
		assert.Equal(t, new("Baru buka"), got.Note)
		require.NotNil(t, got.Location)
		assert.InDelta(t, 106.82, got.Location.X(), 1e-9)
		assert.InDelta(t, -6.2, got.Location.Y(), 1e-9)
		assert.Equal(t, suggestions.StatusPending, got.Status)
		assert.Nil(t, got.RejectionReason)
		assert.Nil(t, got.PlaceID)
		assert.Nil(t, got.ReviewedBy)
		assert.Nil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), got.CreatedAt, time.Minute)
		assert.Equal(t, suggestions.Author{ID: f.userID, Name: "Ana"}, got.User)

		stored, err := f.repo.GetSuggestion(ctx, got.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("leaves optional fields empty", func(t *testing.T) {
		f := newFixture(t)

		got := f.create(t, "Laundry")

		assert.Nil(t, got.Address)
		assert.Nil(t, got.OperatingHours)
		assert.Nil(t, got.EstimatedPriceRange)
		assert.Nil(t, got.ShortDescription)
		assert.Nil(t, got.Note)
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown sub-district", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.SubDistrictID = 999

		got, err := f.repo.Create(ctx, s)

		assert.ErrorIs(t, err, places.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrCategoryNotFound for an unknown category", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.CategoryID = 999

		got, err := f.repo.Create(ctx, s)

		assert.ErrorIs(t, err, places.ErrCategoryNotFound)
		assert.Nil(t, got)
	})

	t.Run("fails for an unknown user", func(t *testing.T) {
		f := newFixture(t)
		s := f.newSuggestion("Laundry")
		s.UserID = 999

		got, err := f.repo.Create(ctx, s)

		assert.True(t, database.IsCode(err, database.ForeignKeyViolation), "got %v", err)
		assert.Nil(t, got)
	})

	t.Run("allows up to the pending limit and then returns ErrTooManyPending", func(t *testing.T) {
		f := newFixture(t)
		for i := range suggestions.MaxPendingPerUser {
			f.create(t, fmt.Sprintf("Laundry %d", i))
		}

		got, err := f.repo.Create(ctx, f.newSuggestion("One too many"))

		assert.ErrorIs(t, err, suggestions.ErrTooManyPending)
		assert.Nil(t, got)
		all, err := f.repo.GetByUser(ctx, f.userID)
		require.NoError(t, err)
		assert.Len(t, all, suggestions.MaxPendingPerUser, "the rejected one was not stored")
	})

	t.Run("counts only the user's own pending suggestions", func(t *testing.T) {
		f := newFixture(t)
		for i := range suggestions.MaxPendingPerUser {
			f.create(t, fmt.Sprintf("Laundry %d", i))
		}
		other := f.newSuggestion("Other user's")
		other.UserID = f.reviewerID

		_, err := f.repo.Create(ctx, other)

		assert.NoError(t, err)
	})

	t.Run("frees a slot when a pending suggestion is reviewed", func(t *testing.T) {
		for name, review := range map[string]func(f fixture, id int64) error{
			"approved": func(f fixture, id int64) error {
				placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")
				_, err := f.repo.MarkApproved(ctx, id, placeID, f.reviewerID)
				return err
			},
			"rejected": func(f fixture, id int64) error {
				_, err := f.repo.Reject(ctx, id, f.reviewerID, nil)
				return err
			},
		} {
			t.Run(name, func(t *testing.T) {
				f := newFixture(t)
				first := f.create(t, "Laundry 0")
				for i := 1; i < suggestions.MaxPendingPerUser; i++ {
					f.create(t, fmt.Sprintf("Laundry %d", i))
				}
				require.NoError(t, review(f, first.ID))

				_, err := f.repo.Create(ctx, f.newSuggestion("Replacement"))

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
				_, errs[i] = f.repo.Create(ctx, f.newSuggestion(fmt.Sprintf("Laundry %d", i)))
			}()
		}
		wg.Wait()

		var stored, tooMany int
		for _, err := range errs {
			if err == nil {
				stored++
				continue
			}
			require.ErrorIs(t, err, suggestions.ErrTooManyPending)
			tooMany++
		}
		assert.Equal(t, suggestions.MaxPendingPerUser, stored)
		assert.Equal(t, attempts-suggestions.MaxPendingPerUser, tooMany)
	})
}

func TestRepository_GetSuggestion(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the suggestion with its author", func(t *testing.T) {
		f := newFixture(t)
		f.create(t, "First")
		created := f.create(t, "Second")

		got, err := f.repo.GetSuggestion(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, created, got)
		assert.Equal(t, "Ana", got.User.Name)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.GetSuggestion(ctx, 999)

		assert.ErrorIs(t, err, suggestions.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_GetByUser(t *testing.T) {
	ctx := context.Background()

	t.Run("returns only the user's suggestions, newest first", func(t *testing.T) {
		f := newFixture(t)
		oldest := f.create(t, "Oldest")
		newest := f.create(t, "Newest")
		other := f.newSuggestion("Other user's")
		other.UserID = f.reviewerID
		_, err := f.repo.Create(ctx, other)
		require.NoError(t, err)
		setCreatedAt(t, oldest.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, newest.ID, "2026-01-02T00:00:00Z")

		got, err := f.repo.GetByUser(ctx, f.userID)

		require.NoError(t, err)
		assert.Equal(t, []int64{newest.ID, oldest.ID}, ids(got))
		assert.Equal(t, "Ana", got[0].User.Name)
	})

	t.Run("lists the later of two simultaneous suggestions first", func(t *testing.T) {
		f := newFixture(t)
		first := f.create(t, "First")
		second := f.create(t, "Second")
		setCreatedAt(t, first.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, second.ID, "2026-01-01T00:00:00Z")

		got, err := f.repo.GetByUser(ctx, f.userID)

		require.NoError(t, err)
		assert.Equal(t, []int64{second.ID, first.ID}, ids(got))
	})

	t.Run("includes reviewed suggestions", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		_, err := f.repo.Reject(ctx, created.ID, f.reviewerID, nil)
		require.NoError(t, err)

		got, err := f.repo.GetByUser(ctx, f.userID)

		require.NoError(t, err)
		assert.Equal(t, []int64{created.ID}, ids(got))
	})

	t.Run("returns an empty, non-nil slice for a user without suggestions", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.GetByUser(ctx, f.reviewerID)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestRepository_GetMany(t *testing.T) {
	ctx := context.Background()

	// Three suggestions, oldest first: one pending, one approved, one rejected.
	setup := func(t *testing.T) (f fixture, pendingID, approvedID, rejectedID int64) {
		f = newFixture(t)
		pendingID = f.create(t, "Pending").ID
		approvedID = f.create(t, "Approved").ID
		rejectedID = f.create(t, "Rejected").ID

		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Approved")
		_, err := f.repo.MarkApproved(ctx, approvedID, placeID, f.reviewerID)
		require.NoError(t, err)
		_, err = f.repo.Reject(ctx, rejectedID, f.reviewerID, nil)
		require.NoError(t, err)

		setCreatedAt(t, pendingID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, approvedID, "2026-01-02T00:00:00Z")
		setCreatedAt(t, rejectedID, "2026-01-03T00:00:00Z")

		return f, pendingID, approvedID, rejectedID
	}

	t.Run("returns every suggestion, oldest first, without a filter", func(t *testing.T) {
		f, pendingID, approvedID, rejectedID := setup(t)

		got, err := f.repo.GetMany(ctx, suggestions.SuggestionFilter{})

		require.NoError(t, err)
		assert.Equal(t, []int64{pendingID, approvedID, rejectedID}, ids(got))
		assert.Equal(t, "Ana", got[0].User.Name)
	})

	t.Run("keeps only the suggestions in the status", func(t *testing.T) {
		f, pendingID, approvedID, rejectedID := setup(t)

		tests := []struct {
			status suggestions.Status
			want   []int64
		}{
			{suggestions.StatusPending, []int64{pendingID}},
			{suggestions.StatusApproved, []int64{approvedID}},
			{suggestions.StatusRejected, []int64{rejectedID}},
		}
		for _, tt := range tests {
			got, err := f.repo.GetMany(ctx, suggestions.SuggestionFilter{Status: tt.status})

			require.NoError(t, err)
			assert.Equal(t, tt.want, ids(got), string(tt.status))
		}
	})

	t.Run("lists the earlier of two simultaneous suggestions first", func(t *testing.T) {
		f := newFixture(t)
		first := f.create(t, "First")
		second := f.create(t, "Second")
		setCreatedAt(t, first.ID, "2026-01-01T00:00:00Z")
		setCreatedAt(t, second.ID, "2026-01-01T00:00:00Z")

		got, err := f.repo.GetMany(ctx, suggestions.SuggestionFilter{})

		require.NoError(t, err)
		assert.Equal(t, []int64{first.ID, second.ID}, ids(got))
	})

	t.Run("returns an empty, non-nil slice when nothing matches", func(t *testing.T) {
		f := newFixture(t)
		f.create(t, "Pending")

		got, err := f.repo.GetMany(ctx, suggestions.SuggestionFilter{Status: suggestions.StatusRejected})

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestRepository_MarkApproved(t *testing.T) {
	ctx := context.Background()

	t.Run("marks a pending suggestion approved as the place", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")

		got, err := f.repo.MarkApproved(ctx, created.ID, placeID, f.reviewerID)

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, suggestions.StatusApproved, got.Status)
		assert.Equal(t, new(placeID), got.PlaceID)
		assert.Equal(t, new(f.reviewerID), got.ReviewedBy)
		require.NotNil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), *got.ReviewedAt, time.Minute)
		assert.Nil(t, got.RejectionReason)
		assert.Equal(t, suggestions.Author{ID: f.userID, Name: "Ana"}, got.User, "the author stays the submitter")
		assert.Equal(t, created.Name, got.Name)

		stored, err := f.repo.GetSuggestion(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.MarkApproved(ctx, 999, 1, f.reviewerID)

		assert.ErrorIs(t, err, suggestions.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrAlreadyReviewed for a reviewed suggestion and keeps its outcome", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		rejected, err := f.repo.Reject(ctx, created.ID, f.reviewerID, new("Duplikat"))
		require.NoError(t, err)
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")

		got, err := f.repo.MarkApproved(ctx, created.ID, placeID, f.reviewerID)

		assert.ErrorIs(t, err, suggestions.ErrAlreadyReviewed)
		assert.Nil(t, got)
		stored, err := f.repo.GetSuggestion(ctx, created.ID)
		require.NoError(t, err)
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
				_, errs[i] = f.repo.MarkApproved(ctx, created.ID, placeID, f.reviewerID)
			}()
		}
		wg.Wait()

		var approved int
		for _, err := range errs {
			if err == nil {
				approved++
				continue
			}
			assert.ErrorIs(t, err, suggestions.ErrAlreadyReviewed)
		}
		assert.Equal(t, 1, approved)
	})
}

func TestRepository_Reject(t *testing.T) {
	ctx := context.Background()

	t.Run("marks a pending suggestion rejected with the reason", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")

		got, err := f.repo.Reject(ctx, created.ID, f.reviewerID, new("Sudah ada"))

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, suggestions.StatusRejected, got.Status)
		assert.Equal(t, new("Sudah ada"), got.RejectionReason)
		assert.Equal(t, new(f.reviewerID), got.ReviewedBy)
		require.NotNil(t, got.ReviewedAt)
		assert.WithinDuration(t, time.Now(), *got.ReviewedAt, time.Minute)
		assert.Nil(t, got.PlaceID)
		assert.Equal(t, suggestions.Author{ID: f.userID, Name: "Ana"}, got.User)

		stored, err := f.repo.GetSuggestion(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("accepts a rejection without a reason", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")

		got, err := f.repo.Reject(ctx, created.ID, f.reviewerID, nil)

		require.NoError(t, err)
		assert.Equal(t, suggestions.StatusRejected, got.Status)
		assert.Nil(t, got.RejectionReason)
	})

	t.Run("returns ErrSuggestionNotFound for an unknown id", func(t *testing.T) {
		f := newFixture(t)

		got, err := f.repo.Reject(ctx, 999, f.reviewerID, nil)

		assert.ErrorIs(t, err, suggestions.ErrSuggestionNotFound)
		assert.Nil(t, got)
	})

	t.Run("returns ErrAlreadyReviewed for a reviewed suggestion and keeps its outcome", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		placeID := pgtest.InsertPlace(t, testDB, f.subDistrictID, f.categoryID, "Laundry")
		approved, err := f.repo.MarkApproved(ctx, created.ID, placeID, f.reviewerID)
		require.NoError(t, err)

		got, err := f.repo.Reject(ctx, created.ID, f.reviewerID, new("Too late"))

		assert.ErrorIs(t, err, suggestions.ErrAlreadyReviewed)
		assert.Nil(t, got)
		stored, err := f.repo.GetSuggestion(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, approved, stored)
	})
}
