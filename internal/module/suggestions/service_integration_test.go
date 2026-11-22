//go:build integration

package suggestions_test

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/database"
	"rantau-go/internal/domain"
	"rantau-go/internal/module/places"
	"rantau-go/internal/module/suggestions"
	"rantau-go/internal/module/suggestions/mock"
	"rantau-go/internal/pkg/postgis"
)

// newPlaceService returns the place service as production wires it, without media storage.
func newPlaceService() places.Service {
	return places.NewService(places.NewPostgresRepository(testDB), nil)
}

// laundryPlace is a valid place in the fixture's sub-district and laundry category.
func (f fixture) laundryPlace(name string) *domain.Place {
	return &domain.Place{
		SubDistrictID: f.subDistrictID,
		CategoryID:    f.categoryID,
		Name:          name,
		Location:      postgis.MustPoint(106.82, -6.2),
		Attributes:    json.RawMessage(`{"service_type":"Swalayan","turnaround_hours":{"min":24,"max":48}}`),
	}
}

// placeCount returns how many places are stored.
func placeCount(t *testing.T) int {
	t.Helper()

	var n int
	require.NoError(t, testDB.Get(&n, `SELECT COUNT(*) FROM PLACES`), "count places")
	return n
}

func TestService_Approve_Transaction(t *testing.T) {
	t.Run("creates_place_and_approves", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		service := suggestions.NewService(f.repo, database.NewTransactor(testDB), newPlaceService())

		got, err := service.Approve(t.Context(), created.ID, f.reviewerID, f.laundryPlace("Laundry Kilat"))

		require.NoError(t, err)
		assert.Equal(t, domain.StatusApproved, got.Status)
		require.NotNil(t, got.PlaceID)
		place, err := newPlaceService().Fetch(t.Context(), *got.PlaceID)
		require.NoError(t, err, "Fetch(%d)", *got.PlaceID)
		assert.Equal(t, "Laundry Kilat", place.Name)
	})

	t.Run("concurrent_approvals_create_one_place", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		service := suggestions.NewService(f.repo, database.NewTransactor(testDB), newPlaceService())
		const admins = 8

		var wg sync.WaitGroup
		errs := make([]error, admins)
		for i := range admins {
			wg.Go(func() {
				_, errs[i] = service.Approve(t.Context(), created.ID, f.reviewerID, f.laundryPlace("Laundry Kilat"))
			})
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
		assert.Equal(t, 1, placeCount(t), "places")
	})

	t.Run("failed_mark_discards_place", func(t *testing.T) {
		f := newFixture(t)
		created := f.create(t, "Laundry")
		boom := errors.New("boom")
		repo := mock.NewMockRepository(gomock.NewController(t))
		repo.EXPECT().Fetch(gomock.Any(), created.ID).Return(created, nil)
		repo.EXPECT().MarkApproved(gomock.Any(), created.ID, gomock.Any(), f.reviewerID).Return(nil, boom)
		service := suggestions.NewService(repo, database.NewTransactor(testDB), newPlaceService())

		_, err := service.Approve(t.Context(), created.ID, f.reviewerID, f.laundryPlace("Laundry Kilat"))

		assert.ErrorIs(t, err, boom)
		assert.Zero(t, placeCount(t), "the place outlived the rollback")
	})
}
