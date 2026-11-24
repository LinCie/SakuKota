//go:build integration

package places_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/module/places/mock"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/testutil/redistest"
)

// newCached returns a cached repository over repo and Redis, where repo expects
// exactly locationCalls ListLocations calls (the cache misses).
func newCached(t *testing.T, repo places.Repository, locationCalls int) *places.CachedRepository {
	t.Helper()

	spy := mock.NewMockRepository(gomock.NewController(t))
	spy.EXPECT().ListLocations(gomock.Any(), gomock.Any()).DoAndReturn(repo.ListLocations).Times(locationCalls)
	spy.EXPECT().ListByIDs(gomock.Any(), gomock.Any()).DoAndReturn(repo.ListByIDs).AnyTimes()
	spy.EXPECT().List(gomock.Any(), gomock.Any()).DoAndReturn(repo.List).AnyTimes()

	return places.NewCachedRepository(spy, testRedis)
}

// TestCachedRepository_List checks that, cold and warm, the cached repository answers
// like the database, and only the cold call reads the index.
func TestCachedRepository_List(t *testing.T) {
	ctx := t.Context()
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
	filters := map[string]places.Filter{
		"sub_district":         {SubDistrictID: setiabudi},
		"categories":           {SubDistrictID: setiabudi, CategoryIDs: []int64{tailor, laundry}},
		"query":                {SubDistrictID: setiabudi, Query: "laundry"},
		"bounds":               {SubDistrictID: setiabudi, Bounds: box},
		"bounds_category":      {SubDistrictID: setiabudi, Bounds: box, CategoryIDs: []int64{laundry}},
		"bounds_distance":      {SubDistrictID: setiabudi, Bounds: nearBox},
		"bounds_limit":         {SubDistrictID: setiabudi, Bounds: nearBox, Limit: 1},
		"bounds_offset":        {SubDistrictID: setiabudi, Bounds: nearBox, Limit: 1, Offset: 1},
		"spread":               {SubDistrictID: setiabudi, Bounds: wideBox, Limit: 3},
		"spread_offset":        {SubDistrictID: setiabudi, Bounds: wideBox, Limit: 3, Offset: 3},
		"limit":                {SubDistrictID: setiabudi, Limit: 2},
		"offset":               {SubDistrictID: setiabudi, Limit: 2, Offset: 2},
		"offset_past_end":      {SubDistrictID: setiabudi, Limit: 2, Offset: 20},
		"no_match":             {SubDistrictID: setiabudi, Query: "nothing like this"},
		"without_sub_district": {Limit: 3},
	}

	for name, filter := range filters {
		t.Run(name, func(t *testing.T) {
			// The service always sets a limit, so the repository requires one.
			if filter.Limit == 0 {
				filter.Limit = 100
			}

			want, wantTotal, err := repo.List(ctx, filter)
			require.NoError(t, err, "List(%+v)", filter)

			// The cold call misses and reads the index once, the warm call hits.
			// Without a sub-district nothing is cached, so the index is never read.
			redistest.Reset(t, testRedis)
			locationCalls := 1
			if filter.SubDistrictID == 0 {
				locationCalls = 0
			}
			cached := newCached(t, repo, locationCalls)

			for _, pass := range []string{"cold", "warm"} {
				got, total, err := cached.List(ctx, filter)
				require.NoError(t, err, "%s List(%+v)", pass, filter)

				assert.Equal(t, want, got, "%s List(%+v)", pass, filter)
				assert.Equal(t, wantTotal, total, "%s List(%+v) total", pass, filter)
			}
		})
	}

	t.Run("unknown_sub_district_not_cached", func(t *testing.T) {
		redistest.Reset(t, testRedis)
		cached := newCached(t, repo, 2)

		for range 2 {
			_, _, err := cached.List(ctx, places.Filter{SubDistrictID: 999999})
			assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "List(sub-district 999999)")
		}
	})
}
