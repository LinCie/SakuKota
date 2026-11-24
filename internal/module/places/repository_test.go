//go:build integration

package places_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sakukota-go/internal/database"
	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/places"
	"sakukota-go/internal/pkg/postgis"
	"sakukota-go/internal/testutil/pgtest"
	"sakukota-go/internal/testutil/redistest"
)

// testDB and testRedis are shared by the tests in this package. pgtest.Reset and
// redistest.Reset empty them before each test.
var (
	testDB    *sqlx.DB
	testRedis *redis.Client
)

func TestMain(m *testing.M) {
	db, stopDB, err := pgtest.Start(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	testDB = db

	rdb, stopRedis, err := redistest.Start(context.Background())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		stopDB()
		os.Exit(1)
	}
	testRedis = rdb

	code := m.Run()
	stopRedis()
	stopDB()
	os.Exit(code)
}

// newRepository returns a repository over an emptied database.
func newRepository(t *testing.T) *places.PostgresRepository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return places.NewPostgresRepository(testDB)
}

func insertSubDistrict(t *testing.T, name string) int64 {
	t.Helper()
	return pgtest.InsertSubDistrict(t, testDB, name)
}

// categoryID returns the id of a category the migrations define.
func categoryID(t *testing.T, slug string) int64 {
	t.Helper()
	return pgtest.CategoryID(t, testDB, slug)
}

// newPlace returns a place that is valid to create.
func newPlace(subDistrictID, categoryID int64, name string) *domain.Place {
	return &domain.Place{
		SubDistrictID: subDistrictID,
		CategoryID:    categoryID,
		Name:          name,
		Attributes:    json.RawMessage(`{}`),
	}
}

func mustCreate(t *testing.T, repo places.Repository, p *domain.Place) *domain.Place {
	t.Helper()

	created, err := repo.Create(t.Context(), p)
	require.NoError(t, err, "Create(%q)", p.Name)

	return created
}

func names(places []domain.Place) []string {
	out := make([]string, 0, len(places))
	for _, p := range places {
		out = append(out, p.Name)
	}

	return out
}

// insertReviewMedia inserts the user's review of the place with one media stored under key.
func insertReviewMedia(t *testing.T, placeID, userID int64, key string) {
	t.Helper()

	_, err := testDB.Exec(
		`WITH review AS (
			INSERT INTO PLACE_REVIEWS (place_id, user_id, rating) VALUES ($1, $2, 5) RETURNING id
		), media AS (
			INSERT INTO MEDIAS (key, content_type, size_bytes) VALUES ($3, 'image/jpeg', 1) RETURNING id
		)
		INSERT INTO PLACE_REVIEW_MEDIAS (review_id, media_id, position)
		SELECT review.id, media.id, 1 FROM review, media`,
		placeID, userID, key,
	)
	require.NoError(t, err, "insert review media %q", key)
}

// queuedKeys returns the storage keys waiting for deletion.
func queuedKeys(t *testing.T) []string {
	t.Helper()
	return pgtest.QueuedKeys(t, testDB)
}

// near reports whether p is within 1e-9 degrees of the coordinates.
func near(p *postgis.Point, longitude, latitude float64) bool {
	return p != nil && math.Abs(p.X()-longitude) <= 1e-9 && math.Abs(p.Y()-latitude) <= 1e-9
}

func TestRepository_Create(t *testing.T) {
	ctx := t.Context()

	t.Run("stores_place", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		got, err := repo.Create(ctx, &domain.Place{
			SubDistrictID:       subDistrictID,
			CategoryID:          laundryID,
			Name:                "Laundry Bersih",
			Address:             new("Jl. Sudirman 1"),
			Location:            postgis.MustPoint(106.82, -6.2),
			OperatingHours:      new("08:00-20:00"),
			EstimatedPriceRange: new("Rp5.000-Rp10.000"),
			ShortDescription:    new("Cuci kilat"),
			Attributes:          json.RawMessage(`{"service_type": "Swalayan"}`),
		})
		require.NoError(t, err, "Create()")

		assert.NotZero(t, got.ID, "Create().ID")
		wantFields := domain.Place{
			SubDistrictID:       subDistrictID,
			CategoryID:          laundryID,
			Name:                "Laundry Bersih",
			Address:             new("Jl. Sudirman 1"),
			Location:            got.Location,
			OperatingHours:      new("08:00-20:00"),
			EstimatedPriceRange: new("Rp5.000-Rp10.000"),
			ShortDescription:    new("Cuci kilat"),
			Attributes:          got.Attributes,
			Medias:              []domain.Media{},
			Category:            domain.Category{ID: laundryID, Name: got.Category.Name, Slug: "laundry", Code: "L", Label: "Laundry", Color: "sign"},
		}
		wantFields.ID = got.ID
		assert.Equal(t, wantFields, *got, "Create()")
		assert.JSONEq(t, `{"service_type": "Swalayan"}`, string(got.Attributes), "Create().Attributes")
		assert.True(t, near(got.Location, 106.82, -6.2), "Create().Location = %v, want (106.82, -6.2)", got.Location)

		stored, err := repo.Fetch(ctx, got.ID)
		require.NoError(t, err, "Fetch(%d)", got.ID)
		assert.Equal(t, got, stored, "Fetch(%d) differs from the created place", got.ID)
	})

	t.Run("optional_fields_empty", func(t *testing.T) {
		repo := newRepository(t)

		got := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		assert.True(t, got.Address == nil && got.Location == nil && got.OperatingHours == nil && got.EstimatedPriceRange == nil && got.ShortDescription == nil,
			"Create() = %+v, want every optional field nil", got)
	})

	t.Run("own_ids", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		first := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "First"))
		second := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Second"))

		assert.Less(t, first.ID, second.ID, "Create() ids, want increasing")
	})

	t.Run("unknown_sub_district", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, newPlace(999, categoryID(t, "laundry"), "Laundry"))

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "Create(sub-district 999)")
		assert.Nil(t, got, "Create(sub-district 999)")
	})

	t.Run("unknown_category", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, newPlace(insertSubDistrict(t, "Setiabudi"), 999, "Laundry"))

		assert.ErrorIs(t, err, domain.ErrCategoryNotFound, "Create(category 999)")
		assert.Nil(t, got, "Create(category 999)")
	})
}

func TestRepository_CreateMany(t *testing.T) {
	ctx := t.Context()

	t.Run("stores_places_in_order", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		first := newPlace(subDistrictID, laundryID, "First")
		first.Location = postgis.MustPoint(106.82, -6.2)
		got, err := repo.CreateMany(ctx, []domain.Place{*first, *newPlace(subDistrictID, laundryID, "Second")})
		require.NoError(t, err, "CreateMany()")

		require.Len(t, got, 2, "CreateMany()")
		assert.Equal(t, []string{"First", "Second"}, names(got), "CreateMany() names")
		assert.Less(t, got[0].ID, got[1].ID, "CreateMany() ids, want increasing")
		assert.True(t, near(got[0].Location, 106.82, -6.2), "CreateMany()[0].Location = %v, want (106.82, -6.2)", got[0].Location)
		for _, p := range got {
			stored, err := repo.Fetch(ctx, p.ID)
			require.NoError(t, err, "Fetch(%d)", p.ID)
			assert.Equal(t, &p, stored, "Fetch(%d) differs from the created place", p.ID)
		}
	})

	t.Run("failure_stores_nothing", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		got, err := repo.CreateMany(ctx, []domain.Place{
			*newPlace(subDistrictID, laundryID, "First"),
			*newPlace(999, laundryID, "Second"),
		})

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "CreateMany(sub-district 999)")
		var batchErr *domain.PlaceBatchError
		require.ErrorAs(t, err, &batchErr, "CreateMany(sub-district 999)")
		assert.Equal(t, 2, batchErr.Position, "CreateMany(sub-district 999) error position")
		assert.Nil(t, got, "CreateMany(sub-district 999)")

		_, total, err := repo.List(ctx, places.Filter{Limit: 10})
		require.NoError(t, err, "List()")
		assert.Zero(t, total, "places stored after a failed CreateMany")
	})
}

func TestRepository_Fetch(t *testing.T) {
	ctx := t.Context()

	t.Run("with_category", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "tailor"), "Penjahit"))

		got, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)

		assert.Equal(t, created, got, "Fetch(%d)", created.ID)
		assert.Equal(t, "tailor", got.Category.Slug, "Fetch(%d).Category.Slug", created.ID)
	})

	t.Run("unknown_id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Fetch(ctx, 999)

		assert.ErrorIs(t, err, domain.ErrPlaceNotFound, "Fetch(999)")
		assert.Nil(t, got, "Fetch(999)")
	})
}

func TestRepository_List(t *testing.T) {
	ctx := t.Context()

	// Four places, created in this order so ids follow it:
	//   Laundry Bersih   laundry   setiabudi  inside the Setiabudi box
	//   Penjahit Maju    tailor    setiabudi  inside the Setiabudi box
	//   Apotek Sehat     pharmacy  menteng    far from the box
	//   Kios_Jahit       tailor    menteng    no location
	repo := newRepository(t)
	setiabudi := insertSubDistrict(t, "Setiabudi")
	menteng := insertSubDistrict(t, "Menteng")
	laundry, tailor, pharmacy := categoryID(t, "laundry"), categoryID(t, "tailor"), categoryID(t, "pharmacy")

	laundryPlace := newPlace(setiabudi, laundry, "Laundry Bersih")
	laundryPlace.Address = new("Jl. Sudirman 1")
	laundryPlace.ShortDescription = new("Cuci kilat")
	laundryPlace.Location = postgis.MustPoint(106.82, -6.20)
	mustCreate(t, repo, laundryPlace)

	tailorPlace := newPlace(setiabudi, tailor, "Penjahit Maju")
	tailorPlace.Address = new("Jl. Thamrin 2")
	tailorPlace.Location = postgis.MustPoint(106.84, -6.21)
	mustCreate(t, repo, tailorPlace)

	pharmacyPlace := newPlace(menteng, pharmacy, "Apotek Sehat")
	pharmacyPlace.ShortDescription = new("Diskon 100% obat")
	pharmacyPlace.Location = postgis.MustPoint(106.90, -6.30)
	mustCreate(t, repo, pharmacyPlace)

	mustCreate(t, repo, newPlace(menteng, tailor, "Kios_Jahit"))

	tests := []struct {
		name      string
		filter    places.Filter
		want      []string
		wantTotal int
	}{
		{
			name:      "no_filter",
			filter:    places.Filter{},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "sub_district",
			filter:    places.Filter{SubDistrictID: menteng},
			want:      []string{"Apotek Sehat", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "one_category",
			filter:    places.Filter{CategoryIDs: []int64{tailor}},
			want:      []string{"Penjahit Maju", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "any_category",
			filter:    places.Filter{CategoryIDs: []int64{laundry, pharmacy}},
			want:      []string{"Laundry Bersih", "Apotek Sehat"},
			wantTotal: 2,
		},
		{
			name:      "query_name",
			filter:    places.Filter{Query: "LAUNDRY"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query_address",
			filter:    places.Filter{Query: "thamrin"},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "query_description",
			filter:    places.Filter{Query: "kilat"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query_category_name",
			filter:    places.Filter{Query: "pharm"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query_percent_literal",
			filter:    places.Filter{Query: "%"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query_underscore_literal",
			filter:    places.Filter{Query: "_"},
			want:      []string{"Kios_Jahit"},
			wantTotal: 1,
		},
		{
			name:      "query_no_match",
			filter:    places.Filter{Query: "nothing like this"},
			want:      []string{},
			wantTotal: 0,
		},
		{
			name:      "bounds_inside",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.81, MinLat: -6.22, MaxLng: 106.85, MaxLat: -6.19}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds_edge",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.84, MaxLat: -6.20}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds_just_past_edge",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.839999, MaxLat: -6.20}},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "bounds_outside",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.85, MinLat: -6.40, MaxLng: 107, MaxLat: -6.25}},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "bounds_by_distance",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.81, MinLat: -6.23, MaxLng: 106.87, MaxLat: -6.19}},
			want:      []string{"Penjahit Maju", "Laundry Bersih"},
			wantTotal: 2,
		},
		// With a limit of 1 both places share one 0.0625° grid cell, (1709, -100).
		// Its categories take turns from (index - 1709 - -100) mod 2, where tailor is
		// index 0 and laundry index 1 (by id), so laundry goes first.
		{
			name:      "limit_sizes_grid",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.81, MinLat: -6.23, MaxLng: 106.87, MaxLat: -6.19}, Limit: 1},
			want:      []string{"Laundry Bersih"},
			wantTotal: 2,
		},
		{
			name:      "bounds_offset",
			filter:    places.Filter{Bounds: &places.BoundingBox{MinLng: 106.81, MinLat: -6.23, MaxLng: 106.87, MaxLat: -6.19}, Limit: 1, Offset: 1},
			want:      []string{"Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name: "filters_combine",
			filter: places.Filter{
				SubDistrictID: setiabudi,
				CategoryIDs:   []int64{tailor, pharmacy},
				Query:         "penjahit",
				Bounds:        &places.BoundingBox{MinLng: 106, MinLat: -7, MaxLng: 107, MaxLat: -6},
			},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "limit_cuts_page",
			filter:    places.Filter{Limit: 2},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 4,
		},
		{
			name:      "limit_equals_matches",
			filter:    places.Filter{Limit: 4},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "limit_above_matches",
			filter:    places.Filter{Limit: 10},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "offset_skips",
			filter:    places.Filter{Limit: 2, Offset: 1},
			want:      []string{"Penjahit Maju", "Apotek Sehat"},
			wantTotal: 4,
		},
		{
			name:      "last_page",
			filter:    places.Filter{Limit: 2, Offset: 3},
			want:      []string{"Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "offset_past_matches",
			filter:    places.Filter{Limit: 2, Offset: 10},
			want:      []string{},
			wantTotal: 4,
		},
		{
			name:      "offset_after_filters",
			filter:    places.Filter{SubDistrictID: menteng, Limit: 1, Offset: 1},
			want:      []string{"Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "limit_after_filters",
			filter:    places.Filter{SubDistrictID: menteng, Limit: 1},
			want:      []string{"Apotek Sehat"},
			wantTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The service always sets a limit, so the repository requires one.
			if tt.filter.Limit == 0 {
				tt.filter.Limit = 100
			}

			got, total, err := repo.List(ctx, tt.filter)
			require.NoError(t, err, "List(%+v)", tt.filter)

			assert.Equal(t, tt.want, names(got), "List(%+v) names", tt.filter)
			assert.Equal(t, tt.wantTotal, total, "List(%+v) total", tt.filter)
		})
	}

	t.Run("category_of_each", func(t *testing.T) {
		got, _, err := repo.List(ctx, places.Filter{Limit: 1})
		require.NoError(t, err, "List(limit 1)")
		require.Len(t, got, 1, "List(limit 1)")
		assert.Equal(t, "laundry", got[0].Category.Slug, "List(limit 1)[0].Category.Slug")
		assert.Equal(t, "Laundry", got[0].Category.Label, "List(limit 1)[0].Category.Label")
	})

	t.Run("unknown_sub_district", func(t *testing.T) {
		got, total, err := repo.List(ctx, places.Filter{SubDistrictID: 999})

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "List(sub-district 999)")
		assert.Nil(t, got, "List(sub-district 999)")
		assert.Zero(t, total, "List(sub-district 999) total")
	})

	t.Run("unknown_category", func(t *testing.T) {
		got, total, err := repo.List(ctx, places.Filter{CategoryIDs: []int64{laundry, 999}})

		assert.ErrorIs(t, err, domain.ErrCategoryNotFound, "List(categories %d, 999)", laundry)
		assert.Nil(t, got, "List(categories %d, 999)", laundry)
		assert.Zero(t, total, "List(categories %d, 999) total", laundry)
	})

	t.Run("empty_non_nil", func(t *testing.T) {
		empty := newRepository(t)

		got, total, err := empty.List(ctx, places.Filter{Limit: 10})
		require.NoError(t, err, "List(limit 10)")

		assert.Equal(t, []domain.Place{}, got, "List(limit 10) must be an empty, non-nil slice")
		assert.Zero(t, total, "List(limit 10) total")
	})
}

func TestRepository_List_Spread(t *testing.T) {
	repo := newRepository(t)
	setiabudi := insertSubDistrict(t, "Setiabudi")
	laundry := categoryID(t, "laundry")

	create := func(name string, longitude, latitude float64) {
		p := newPlace(setiabudi, laundry, name)
		p.Location = postgis.MustPoint(longitude, latitude)
		mustCreate(t, repo, p)
	}
	// The box is 0.125° square, so a limit of 4 splits it into four 0.0625° cells.
	// Four places crowd the centre, all in the south-west cell.
	create("Tengah 1", 106.81, -6.19)
	create("Tengah 2", 106.809, -6.19)
	create("Tengah 3", 106.811, -6.19)
	create("Tengah 4", 106.808, -6.192)
	create("Barat Laut", 106.76, -6.13)
	create("Timur Laut", 106.87, -6.13)
	box := &places.BoundingBox{MinLng: 106.75, MinLat: -6.25, MaxLng: 106.875, MaxLat: -6.125}

	got, total, err := repo.List(t.Context(), places.Filter{Bounds: box, Limit: 4})
	require.NoError(t, err, "List(bounds, limit 4)")

	// One place from each cell (the one nearest the cell's centre), nearest the
	// box's centre first, then the crowded cell's next, instead of the four
	// places nearest the box's centre.
	want := []string{"Tengah 4", "Barat Laut", "Timur Laut", "Tengah 2"}
	assert.Equal(t, want, names(got), "List(bounds, limit 4) names")
	assert.Equal(t, 6, total, "List(bounds, limit 4) total")
}

func TestRepository_ListCategories(t *testing.T) {
	t.Run("migrated_categories", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.ListCategories(t.Context())
		require.NoError(t, err, "ListCategories()")
		require.NotEmpty(t, got, "ListCategories() must return the categories the migrations define")
		for i := 1; i < len(got); i++ {
			assert.Less(t, got[i-1].ID, got[i].ID, "ListCategories() ids, want ordered by id")
		}

		var laundry *domain.Category
		for i := range got {
			assert.NotNil(t, got[i].Fields, "ListCategories() category %q fields", got[i].Slug)
			if got[i].Slug == "laundry" {
				laundry = &got[i]
			}
		}
		require.NotNil(t, laundry, "ListCategories() must have a laundry category")

		keys := make([]string, 0, len(laundry.Fields))
		for i, f := range laundry.Fields {
			assert.Equal(t, laundry.ID, f.CategoryID, "laundry field %q CategoryID", f.Key)
			assert.Equal(t, int32(i+1), f.SortOrder, "laundry field %q SortOrder", f.Key)
			keys = append(keys, f.Key)
		}
		wantKeys := []string{"service_type", "turnaround_hours", "express_available", "minimum_weight_kg"}
		assert.Equal(t, wantKeys, keys, "laundry field keys")
	})
}

func TestRepository_FetchCategory(t *testing.T) {
	ctx := t.Context()

	t.Run("with_fields", func(t *testing.T) {
		repo := newRepository(t)
		id := categoryID(t, "tailor")

		got, err := repo.FetchCategory(ctx, id)
		require.NoError(t, err, "FetchCategory(%d)", id)

		assert.Equal(t, id, got.ID, "FetchCategory(%d).ID", id)
		assert.Equal(t, "Tailor", got.Name, "FetchCategory(%d).Name", id)
		assert.Equal(t, "J", got.Code, "FetchCategory(%d).Code", id)
		assert.Equal(t, "Penjahit", got.Label, "FetchCategory(%d).Label", id)
		assert.Equal(t, "plum", got.Color, "FetchCategory(%d).Color", id)
		keys := make([]string, 0, len(got.Fields))
		for _, f := range got.Fields {
			keys = append(keys, f.Key)
		}
		wantKeys := []string{"specialty", "turnaround_days", "same_day_service"}
		assert.Equal(t, wantKeys, keys, "FetchCategory(%d) field keys", id)
	})

	t.Run("select_options", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.FetchCategory(ctx, categoryID(t, "laundry"))
		require.NoError(t, err, "FetchCategory(laundry)")

		serviceType := got.Fields[0]
		assert.Equal(t, domain.FieldTypeSelect, serviceType.DataType, "FetchCategory(laundry).Fields[0].DataType")
		assert.True(t, serviceType.Required, "FetchCategory(laundry).Fields[0].Required")
		wantOptions := domain.FieldOptions{"Swalayan", "Titip cuci", "Swalayan & titip cuci"}
		assert.Equal(t, wantOptions, serviceType.Options, "FetchCategory(laundry).Fields[0].Options")
		assert.Nil(t, got.Fields[1].Options, "FetchCategory(laundry).Fields[1].Options (other types have no options)")
	})

	t.Run("no_fields", func(t *testing.T) {
		repo := newRepository(t)
		var id int64
		err := testDB.Get(&id, `INSERT INTO CATEGORIES (name, slug, code, label, color) VALUES ('Bare', 'bare', 'Z', 'Bare', 'cat') RETURNING id`)
		require.NoError(t, err, "insert category")
		t.Cleanup(func() { _, _ = testDB.Exec(`DELETE FROM CATEGORIES WHERE id = $1`, id) }) // best-effort cleanup

		got, err := repo.FetchCategory(ctx, id)
		require.NoError(t, err, "FetchCategory(%d)", id)

		assert.Equal(t, []domain.CategoryField{}, got.Fields, "FetchCategory(%d).Fields must be an empty, non-nil slice", id)
	})

	t.Run("unknown_id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.FetchCategory(ctx, 999)

		assert.ErrorIs(t, err, domain.ErrCategoryNotFound, "FetchCategory(999)")
		assert.Nil(t, got, "FetchCategory(999)")
	})
}

func TestRepository_Update(t *testing.T) {
	ctx := t.Context()

	t.Run("stores_every_field", func(t *testing.T) {
		repo := newRepository(t)
		setiabudi := insertSubDistrict(t, "Setiabudi")
		menteng := insertSubDistrict(t, "Menteng")
		created := mustCreate(t, repo, newPlace(setiabudi, categoryID(t, "laundry"), "Old"))
		tailorID := categoryID(t, "tailor")

		got, err := repo.Update(ctx, &domain.Place{
			ID:                  created.ID,
			SubDistrictID:       menteng,
			CategoryID:          tailorID,
			Name:                "New",
			Address:             new("Jl. Baru"),
			Location:            postgis.MustPoint(106.9, -6.3),
			OperatingHours:      new("24 jam"),
			EstimatedPriceRange: new("Rp1"),
			ShortDescription:    new("Baru"),
			Attributes:          json.RawMessage(`{"specialty": "Permak"}`),
		})
		require.NoError(t, err, "Update()")

		assert.Equal(t, created.ID, got.ID, "Update().ID")
		assert.Equal(t, menteng, got.SubDistrictID, "Update().SubDistrictID")
		assert.Equal(t, tailorID, got.CategoryID, "Update().CategoryID")
		assert.Equal(t, "New", got.Name, "Update().Name")
		wantTexts := []*string{new("Jl. Baru"), new("24 jam"), new("Rp1"), new("Baru")}
		gotTexts := []*string{got.Address, got.OperatingHours, got.EstimatedPriceRange, got.ShortDescription}
		assert.Equal(t, wantTexts, gotTexts, "Update() text fields")
		assert.JSONEq(t, `{"specialty": "Permak"}`, string(got.Attributes), "Update().Attributes")
		require.NotNil(t, got.Location, "Update().Location")
		assert.InDelta(t, 106.9, got.Location.X(), 1e-9, "Update().Location longitude")
		assert.Equal(t, "tailor", got.Category.Slug, "Update().Category.Slug")

		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, got, stored, "Fetch(%d) differs from the updated place", created.ID)
	})

	t.Run("clears_nil_fields", func(t *testing.T) {
		repo := newRepository(t)
		p := newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry")
		p.Address = new("Jl. Lama")
		p.Location = postgis.MustPoint(106.8, -6.2)
		created := mustCreate(t, repo, p)

		got, err := repo.Update(ctx, newPlaceWithID(created, "Laundry"))
		require.NoError(t, err, "Update()")

		assert.Nil(t, got.Address, "Update().Address")
		assert.Nil(t, got.Location, "Update().Location")
	})

	t.Run("keeps_ratings_medias", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1.jpg")
		_, err := testDB.Exec(`UPDATE PLACES SET rating_count = 2, rating_sum = 9 WHERE id = $1`, created.ID)
		require.NoError(t, err, "set ratings")

		got, err := repo.Update(ctx, newPlaceWithID(created, "Renamed"))
		require.NoError(t, err, "Update()")

		assert.Equal(t, "Renamed", got.Name, "Update().Name")
		assert.Equal(t, 2, got.RatingCount, "Update().RatingCount")
		assert.Equal(t, 9, got.RatingSum, "Update().RatingSum")
		assert.Equal(t, []domain.Media{m}, got.Medias, "Update().Medias")
	})

	t.Run("unknown_id", func(t *testing.T) {
		repo := newRepository(t)
		p := newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry")
		p.ID = 999

		got, err := repo.Update(ctx, p)

		assert.ErrorIs(t, err, domain.ErrPlaceNotFound, "Update(ID 999)")
		assert.Nil(t, got, "Update(ID 999)")
	})

	t.Run("unknown_sub_district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		update := newPlaceWithID(created, "Renamed")
		update.SubDistrictID = 999

		got, err := repo.Update(ctx, update)

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "Update(sub-district 999)")
		assert.Nil(t, got, "Update(sub-district 999)")
		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, "Laundry", stored.Name, "Fetch(%d).Name after a refused update", created.ID)
	})

	t.Run("unknown_category", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		update := newPlaceWithID(created, "Renamed")
		update.CategoryID = 999

		got, err := repo.Update(ctx, update)

		assert.ErrorIs(t, err, domain.ErrCategoryNotFound, "Update(category 999)")
		assert.Nil(t, got, "Update(category 999)")
		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, "Laundry", stored.Name, "Fetch(%d).Name after a refused update", created.ID)
	})
}

// newPlaceWithID returns a copy of p's identity and ownership with the given
// name and no optional fields, to use as an update.
func newPlaceWithID(p *domain.Place, name string) *domain.Place {
	update := newPlace(p.SubDistrictID, p.CategoryID, name)
	update.ID = p.ID

	return update
}

// ratingTotals returns the stored place's rating count and sum.
func ratingTotals(t *testing.T, repo places.Repository, id int64) [2]int {
	t.Helper()

	p, err := repo.Fetch(t.Context(), id)
	require.NoError(t, err, "Fetch(%d)", id)
	return [2]int{p.RatingCount, p.RatingSum}
}

func TestRepository_AddRating(t *testing.T) {
	t.Run("adds_each_rating_to_totals", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		place := mustCreate(t, repo, newPlace(subDistrictID, categoryID(t, "laundry"), "Laundry"))
		other := mustCreate(t, repo, newPlace(subDistrictID, categoryID(t, "laundry"), "Other"))

		for _, rating := range []int{5, 3, 1} {
			require.NoError(t, repo.AddRating(t.Context(), place.ID, rating), "AddRating(%d)", rating)
		}

		assert.Equal(t, [2]int{3, 9}, ratingTotals(t, repo, place.ID))
		// Other places are untouched.
		assert.Equal(t, [2]int{0, 0}, ratingTotals(t, repo, other.ID))
	})

	t.Run("concurrent_ratings_lose_none", func(t *testing.T) {
		repo := newRepository(t)
		place := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		const raters = 20

		var wg sync.WaitGroup
		errs := make([]error, raters)
		for i := range raters {
			wg.Go(func() {
				errs[i] = repo.AddRating(t.Context(), place.ID, 4)
			})
		}
		wg.Wait()

		for i, err := range errs {
			require.NoError(t, err, "AddRating() by rater %d", i)
		}
		assert.Equal(t, [2]int{raters, raters * 4}, ratingTotals(t, repo, place.ID))
	})

	t.Run("unknown_place", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.AddRating(t.Context(), 999, 4), domain.ErrPlaceNotFound)
	})

	t.Run("joins_the_transaction_in_ctx", func(t *testing.T) {
		repo := newRepository(t)
		place := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		errBoom := errors.New("boom")

		err := database.NewTransactor(testDB).WithinTx(t.Context(), func(ctx context.Context) error {
			if err := repo.AddRating(ctx, place.ID, 4); err != nil {
				return err
			}
			return errBoom
		})

		assert.ErrorIs(t, err, errBoom)
		assert.Equal(t, [2]int{0, 0}, ratingTotals(t, repo, place.ID), "the rating outlived the rollback")
	})
}

func TestRepository_RemoveRating(t *testing.T) {
	t.Run("takes_the_rating_off_totals", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		place := mustCreate(t, repo, newPlace(subDistrictID, categoryID(t, "laundry"), "Laundry"))
		other := mustCreate(t, repo, newPlace(subDistrictID, categoryID(t, "laundry"), "Other"))
		for _, rating := range []int{5, 3} {
			require.NoError(t, repo.AddRating(t.Context(), place.ID, rating), "AddRating(%d)", rating)
		}
		require.NoError(t, repo.AddRating(t.Context(), other.ID, 2), "AddRating(other)")

		require.NoError(t, repo.RemoveRating(t.Context(), place.ID, 5))

		assert.Equal(t, [2]int{1, 3}, ratingTotals(t, repo, place.ID))
		// Other places are untouched.
		assert.Equal(t, [2]int{1, 2}, ratingTotals(t, repo, other.ID))
	})

	t.Run("unknown_place", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.RemoveRating(t.Context(), 999, 4), domain.ErrPlaceNotFound)
	})
}

func TestRepository_Delete(t *testing.T) {
	ctx := t.Context()

	t.Run("removes_place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)

		_, err := repo.Fetch(ctx, created.ID)
		assert.ErrorIs(t, err, domain.ErrPlaceNotFound, "Fetch(%d) after Delete", created.ID)
		assert.Empty(t, queuedKeys(t), "queued keys (a place without medias queues nothing)")
	})

	t.Run("joins_the_callers_transaction", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		boom := errors.New("boom")

		err := database.NewTransactor(testDB).WithinTx(ctx, func(ctx context.Context) error {
			require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)
			return boom
		})
		require.ErrorIs(t, err, boom)

		_, err = repo.Fetch(ctx, created.ID)
		assert.NoError(t, err, "Fetch(%d) after the transaction rolled back", created.ID)
	})

	t.Run("queues_images", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		created := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Laundry"))
		kept := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Kept"))
		mustAddMedia(t, repo, created.ID, "places/1/a.jpg")
		mustAddMedia(t, repo, created.ID, "places/1/b.jpg")
		keptMedia := mustAddMedia(t, repo, kept.ID, "places/2/a.jpg")

		require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)

		keys := queuedKeys(t)
		slices.Sort(keys)
		assert.Equal(t, []string{"places/1/a.jpg", "places/1/b.jpg"}, keys, "queued keys")
		assert.Equal(t, 1, storedMediaCount(t), "stored medias (only the other place's media)")
		stored, err := repo.Fetch(ctx, kept.ID)
		require.NoError(t, err, "Fetch(%d)", kept.ID)
		assert.Equal(t, []domain.Media{keptMedia}, stored.Medias, "Fetch(%d).Medias", kept.ID)
	})

	t.Run("queues_review_images", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		created := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Laundry"))
		kept := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Kept"))
		userID := pgtest.InsertUser(t, testDB, "Ana")
		insertReviewMedia(t, created.ID, userID, "reviews/1/1/a.jpg")
		insertReviewMedia(t, kept.ID, userID, "reviews/2/1/a.jpg")

		require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)

		assert.Equal(t, []string{"reviews/1/1/a.jpg"}, queuedKeys(t), "queued keys")
		assert.Equal(t, 1, storedMediaCount(t), "stored medias (only the other place's review media)")
	})

	t.Run("already_queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1.jpg")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/1.jpg')`)
		require.NoError(t, err, "queue key")

		require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)

		assert.Equal(t, []string{"places/1.jpg"}, queuedKeys(t), "queued keys")
	})

	t.Run("leaves_others", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		doomed := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Doomed"))
		kept := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Kept"))

		require.NoError(t, repo.Delete(ctx, doomed.ID), "Delete(%d)", doomed.ID)

		got, total, err := repo.List(ctx, places.Filter{Limit: 10})
		require.NoError(t, err, "List()")
		assert.Equal(t, []string{kept.Name}, names(got), "List() names")
		assert.Equal(t, 1, total, "List() total")
	})

	t.Run("unknown_id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(ctx, 999), domain.ErrPlaceNotFound, "Delete(999)")
	})

	t.Run("second_time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		require.NoError(t, repo.Delete(ctx, created.ID), "Delete(%d)", created.ID)

		assert.ErrorIs(t, repo.Delete(ctx, created.ID), domain.ErrPlaceNotFound, "second Delete(%d)", created.ID)
	})
}

// newMedia returns a promoted image to store under key.
func newMedia(key string) domain.Media {
	return domain.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

// mustAddMedia stores an image under key for the place and returns it.
func mustAddMedia(t *testing.T, repo places.Repository, placeID int64, key string) domain.Media {
	t.Helper()

	got, err := repo.AddMedia(t.Context(), placeID, newMedia(key))
	require.NoError(t, err, "AddMedia(%d, %q)", placeID, key)

	for _, m := range got.Medias {
		if m.Key == key {
			return m
		}
	}
	require.Failf(t, "media missing", "AddMedia(%d, %q) returned a place without the added media", placeID, key)

	return domain.Media{}
}

// mediaKeys returns the keys of medias, in order.
func mediaKeys(medias []domain.Media) []string {
	out := make([]string, 0, len(medias))
	for _, m := range medias {
		out = append(out, m.Key)
	}

	return out
}

// mediaPositions returns the positions of medias, in order.
func mediaPositions(medias []domain.Media) []int {
	out := make([]int, 0, len(medias))
	for _, m := range medias {
		out = append(out, m.Position)
	}

	return out
}

// storedMediaCount returns the number of MEDIAS rows.
func storedMediaCount(t *testing.T) int {
	t.Helper()

	var count int
	require.NoError(t, testDB.Get(&count, `SELECT COUNT(*) FROM MEDIAS`), "count medias")

	return count
}

func TestRepository_AddMedia(t *testing.T) {
	ctx := t.Context()

	t.Run("first_position", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/a.webp"))
		require.NoError(t, err, "AddMedia()")

		assert.Equal(t, created.ID, got.ID, "AddMedia() place ID")
		assert.Equal(t, "laundry", got.Category.Slug, "AddMedia() place category")
		require.Len(t, got.Medias, 1, "AddMedia().Medias")
		m := got.Medias[0]
		assert.NotZero(t, m.ID, "AddMedia().Medias[0].ID")
		want := domain.Media{ID: m.ID, Key: "places/1/a.webp", ContentType: "image/webp", Size: 2048, Position: 1}
		assert.Equal(t, want, m, "AddMedia().Medias[0]")
		assert.Empty(t, queuedKeys(t), "queued keys (adding a media replaces nothing)")

		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, got, stored, "Fetch(%d) differs from the returned place", created.ID)
	})

	t.Run("after_last", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		mustAddMedia(t, repo, created.ID, "places/1/b.webp")

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/c.webp"))
		require.NoError(t, err, "AddMedia()")

		assert.Equal(t, []string{"places/1/a.webp", "places/1/b.webp", "places/1/c.webp"}, mediaKeys(got.Medias), "AddMedia() keys")
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias), "AddMedia() positions")
	})

	t.Run("after_delete", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		removed := mustAddMedia(t, repo, created.ID, "places/1/b.webp")
		mustAddMedia(t, repo, created.ID, "places/1/c.webp")
		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID), "DeleteMedia(%d, %d)", created.ID, removed.ID)

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/d.webp"))
		require.NoError(t, err, "AddMedia()")

		assert.Equal(t, []string{"places/1/a.webp", "places/1/c.webp", "places/1/d.webp"}, mediaKeys(got.Medias), "AddMedia() keys")
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias), "AddMedia() positions")
	})

	t.Run("limit_reached", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		for i := range domain.MaxMedias {
			mustAddMedia(t, repo, created.ID, fmt.Sprintf("places/1/%d.webp", i))
		}

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/extra.webp"))

		assert.ErrorIs(t, err, domain.ErrMediaLimitReached, "AddMedia(11th media)")
		assert.Nil(t, got, "AddMedia(11th media)")
		assert.Equal(t, domain.MaxMedias, storedMediaCount(t), "stored medias (the extra media must not be stored)")
	})

	t.Run("concurrent_limit", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		const attempts = domain.MaxMedias + 4

		errs := make([]error, attempts)
		var wg sync.WaitGroup
		for i := range attempts {
			wg.Go(func() {
				_, errs[i] = repo.AddMedia(ctx, created.ID, newMedia(fmt.Sprintf("places/1/%d.webp", i)))
			})
		}
		wg.Wait()

		added, limited := 0, 0
		for _, err := range errs {
			switch {
			case err == nil: // if NO error
				added++
			case errors.Is(err, domain.ErrMediaLimitReached):
				limited++
			default:
				assert.Fail(t, "unexpected error", "AddMedia() unexpected error: %v", err)
			}
		}
		assert.Equal(t, domain.MaxMedias, added, "concurrent AddMedia() added")
		assert.Equal(t, attempts-domain.MaxMedias, limited, "concurrent AddMedia() limited")

		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, mediaPositions(stored.Medias), "Fetch(%d) positions", created.ID)
	})

	t.Run("unknown_id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.AddMedia(ctx, 999, newMedia("places/1/a.webp"))

		assert.ErrorIs(t, err, domain.ErrPlaceNotFound, "AddMedia(place 999)")
		assert.Nil(t, got, "AddMedia(place 999)")
		assert.Zero(t, storedMediaCount(t), "stored medias")
	})
}

func TestRepository_DeleteMedia(t *testing.T) {
	ctx := t.Context()

	t.Run("queues_image", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		removed := mustAddMedia(t, repo, created.ID, "places/1/b.webp")
		mustAddMedia(t, repo, created.ID, "places/1/c.webp")

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID), "DeleteMedia(%d, %d)", created.ID, removed.ID)

		stored, err := repo.Fetch(ctx, created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)
		assert.Equal(t, []string{"places/1/a.webp", "places/1/c.webp"}, mediaKeys(stored.Medias), "Fetch(%d) keys", created.ID)
		// The later medias shift down.
		assert.Equal(t, []int{1, 2}, mediaPositions(stored.Medias), "Fetch(%d) positions", created.ID)
		assert.Equal(t, 2, storedMediaCount(t), "stored medias")
		assert.Equal(t, []string{"places/1/b.webp"}, queuedKeys(t), "queued keys")
	})

	t.Run("already_queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		removed := mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/1/a.webp')`)
		require.NoError(t, err, "queue key")

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID), "DeleteMedia(%d, %d)", created.ID, removed.ID)

		assert.Equal(t, []string{"places/1/a.webp"}, queuedKeys(t), "queued keys")
	})

	t.Run("other_place", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		owner := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Owner"))
		other := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Other"))
		m := mustAddMedia(t, repo, owner.ID, "places/1/a.webp")

		err := repo.DeleteMedia(ctx, other.ID, m.ID)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound, "DeleteMedia(%d, %d)", other.ID, m.ID)
		assert.Equal(t, 1, storedMediaCount(t), "stored medias (the media must be kept)")
		assert.Empty(t, queuedKeys(t), "queued keys")
	})

	t.Run("unknown_media_or_place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1/a.webp")

		assert.ErrorIs(t, repo.DeleteMedia(ctx, created.ID, 999), domain.ErrMediaNotFound, "DeleteMedia(%d, 999)", created.ID)
		assert.ErrorIs(t, repo.DeleteMedia(ctx, 999, m.ID), domain.ErrMediaNotFound, "DeleteMedia(999, %d)", m.ID)
		assert.Equal(t, 1, storedMediaCount(t), "stored medias")
	})

	t.Run("second_time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		require.NoError(t, repo.DeleteMedia(ctx, created.ID, m.ID), "DeleteMedia(%d, %d)", created.ID, m.ID)

		assert.ErrorIs(t, repo.DeleteMedia(ctx, created.ID, m.ID), domain.ErrMediaNotFound, "second DeleteMedia(%d, %d)", created.ID, m.ID)
	})
}

func TestRepository_List_Medias(t *testing.T) {
	repo := newRepository(t)
	subDistrictID := insertSubDistrict(t, "Setiabudi")
	laundryID := categoryID(t, "laundry")
	first := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "First"))
	second := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Second"))
	mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Without medias"))
	mustAddMedia(t, repo, first.ID, "places/1/a.webp")
	mustAddMedia(t, repo, second.ID, "places/2/a.webp")
	mustAddMedia(t, repo, first.ID, "places/1/b.webp")

	got, _, err := repo.List(t.Context(), places.Filter{Limit: 10})
	require.NoError(t, err, "List(limit 10)")

	require.Len(t, got, 3, "List(limit 10)")
	assert.Equal(t, []string{"places/1/a.webp", "places/1/b.webp"}, mediaKeys(got[0].Medias), "List()[0] media keys")
	assert.Equal(t, []string{"places/2/a.webp"}, mediaKeys(got[1].Medias), "List()[1] media keys")
	assert.Equal(t, []domain.Media{}, got[2].Medias, "List()[2].Medias must be an empty, non-nil list")
}
