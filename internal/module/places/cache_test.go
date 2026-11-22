//go:build integration

package places_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"rantau-go/internal/module/places"
	"rantau-go/internal/module/places/mock"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/testutil/redistest"
)

// newCached returns a cached repository over repo and Redis, where repo expects exactly
// locationCalls GetLocations calls (the cache misses).
func newCached(t *testing.T, repo places.Repository, locationCalls int) places.Repository {
	t.Helper()

	spy := mock.NewMockRepository(gomock.NewController(t))
	spy.EXPECT().GetLocations(gomock.Any(), gomock.Any()).DoAndReturn(repo.GetLocations).Times(locationCalls)
	spy.EXPECT().GetByIDs(gomock.Any(), gomock.Any()).DoAndReturn(repo.GetByIDs).AnyTimes()
	spy.EXPECT().GetMany(gomock.Any(), gomock.Any()).DoAndReturn(repo.GetMany).AnyTimes()

	return places.NewCachedRepository(spy, testRedis)
}

// TestCachedRepository_GetMany checks that, cold and warm, the cached repository answers
// like the database, and only the cold call reads the index.
func TestCachedRepository_GetMany(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)

	setiabudi := insertSubDistrict(t, "Setiabudi")
	menteng := insertSubDistrict(t, "Menteng")
	laundry, tailor := categoryID(t, "laundry"), categoryID(t, "tailor")

	create := func(subDistrictID, categoryID int64, name string, location *postgis.Point) {
		p := newPlace(subDistrictID, categoryID, name)
		p.Location = location
		mustCreate(t, repo, p)
	}
	create(setiabudi, laundry, "Laundry A", postgis.MustPoint(106.82, -6.20))
	create(setiabudi, tailor, "Penjahit B", postgis.MustPoint(106.84, -6.21))
	create(setiabudi, laundry, "Laundry C", nil)
	create(setiabudi, laundry, "Laundry D", postgis.MustPoint(106.90, -6.30))
	create(menteng, tailor, "Penjahit E", postgis.MustPoint(106.83, -6.20))
	// A crowd of both categories, so cells hold several places and take turns.
	create(setiabudi, tailor, "Penjahit F", postgis.MustPoint(106.821, -6.201))
	create(setiabudi, laundry, "Laundry G", postgis.MustPoint(106.822, -6.202))
	create(setiabudi, tailor, "Penjahit H", postgis.MustPoint(106.823, -6.201))
	create(setiabudi, laundry, "Laundry I", postgis.MustPoint(106.88, -6.25))

	box := &places.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.84, MaxLat: -6.20}
	// The centre of nearBox is on Penjahit B, so it comes before Laundry A despite its higher id.
	nearBox := &places.BoundingBox{MinLng: 106.81, MinLat: -6.23, MaxLng: 106.87, MaxLat: -6.19}
	wideBox := &places.BoundingBox{MinLng: 106.80, MinLat: -6.32, MaxLng: 106.92, MaxLat: -6.18}
	filters := map[string]places.PlaceFilter{
		"sub-district":         {SubDistrictID: setiabudi},
		"categories":           {SubDistrictID: setiabudi, CategoryIDs: []int64{tailor, laundry}},
		"query":                {SubDistrictID: setiabudi, Query: "laundry"},
		"bounds":               {SubDistrictID: setiabudi, Bounds: box},
		"bounds and category":  {SubDistrictID: setiabudi, Bounds: box, CategoryIDs: []int64{laundry}},
		"bounds by distance":   {SubDistrictID: setiabudi, Bounds: nearBox},
		"bounds and limit":     {SubDistrictID: setiabudi, Bounds: nearBox, Limit: 1},
		"bounds and offset":    {SubDistrictID: setiabudi, Bounds: nearBox, Limit: 1, Offset: 1},
		"spread":               {SubDistrictID: setiabudi, Bounds: wideBox, Limit: 3},
		"spread and offset":    {SubDistrictID: setiabudi, Bounds: wideBox, Limit: 3, Offset: 3},
		"limit":                {SubDistrictID: setiabudi, Limit: 2},
		"offset":               {SubDistrictID: setiabudi, Limit: 2, Offset: 2},
		"offset past the end":  {SubDistrictID: setiabudi, Limit: 2, Offset: 20},
		"no match":             {SubDistrictID: setiabudi, Query: "nothing like this"},
		"without sub-district": {Limit: 3},
	}

	for name, filter := range filters {
		t.Run(name, func(t *testing.T) {
			// The service always sets a limit, so the repository requires one.
			if filter.Limit == 0 {
				filter.Limit = 100
			}

			want, wantTotal, err := repo.GetMany(ctx, filter)
			require.NoError(t, err)

			// The cold call misses and reads the index once, the warm call hits.
			// Without a sub-district nothing is cached, so the index is never read.
			redistest.Reset(t, testRedis)
			locationCalls := 1
			if filter.SubDistrictID == 0 {
				locationCalls = 0
			}
			cached := newCached(t, repo, locationCalls)

			for _, pass := range []string{"cold", "warm"} {
				got, total, err := cached.GetMany(ctx, filter)
				require.NoError(t, err, pass)

				assert.Equal(t, want, got, pass)
				assert.Equal(t, wantTotal, total, pass)
			}
		})
	}

	t.Run("unknown sub-district is an error that is not cached", func(t *testing.T) {
		redistest.Reset(t, testRedis)
		cached := newCached(t, repo, 2)

		for range 2 {
			_, _, err := cached.GetMany(ctx, places.PlaceFilter{SubDistrictID: 999999})
			assert.ErrorIs(t, err, places.ErrSubDistrictNotFound)
		}
	})
}
