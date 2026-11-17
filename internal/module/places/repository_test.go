//go:build integration

package places_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/places"
	"rantau-go/internal/pkg/postgis"
	"rantau-go/internal/storage"
	"rantau-go/internal/testutil/pgtest"
	"rantau-go/internal/testutil/redistest"
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
func newRepository(t *testing.T) places.Repository {
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
func newPlace(subDistrictID, categoryID int64, name string) *places.Place {
	return &places.Place{
		SubDistrictID: subDistrictID,
		CategoryID:    categoryID,
		Name:          name,
		Attributes:    json.RawMessage(`{}`),
	}
}

func mustCreate(t *testing.T, repo places.Repository, p *places.Place) *places.Place {
	t.Helper()

	created, err := repo.Create(context.Background(), p)
	require.NoError(t, err)

	return created
}

func names(places []places.Place) []string {
	out := make([]string, 0, len(places))
	for _, p := range places {
		out = append(out, p.Name)
	}

	return out
}

// queuedKeys returns the storage keys waiting for deletion.
func queuedKeys(t *testing.T) []string {
	t.Helper()
	return pgtest.QueuedKeys(t, testDB)
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the place and returns it with its category", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		got, err := repo.Create(ctx, &places.Place{
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
		require.NoError(t, err)

		assert.NotZero(t, got.ID)
		assert.Equal(t, subDistrictID, got.SubDistrictID)
		assert.Equal(t, laundryID, got.CategoryID)
		assert.Equal(t, "Laundry Bersih", got.Name)
		assert.Equal(t, new("Jl. Sudirman 1"), got.Address)
		assert.Equal(t, new("08:00-20:00"), got.OperatingHours)
		assert.Equal(t, new("Rp5.000-Rp10.000"), got.EstimatedPriceRange)
		assert.Equal(t, new("Cuci kilat"), got.ShortDescription)
		assert.JSONEq(t, `{"service_type": "Swalayan"}`, string(got.Attributes))
		require.NotNil(t, got.Location)
		assert.InDelta(t, 106.82, got.Location.X(), 1e-9)
		assert.InDelta(t, -6.2, got.Location.Y(), 1e-9)
		assert.Equal(t, 0, got.RatingCount)
		assert.Equal(t, 0, got.RatingSum)
		assert.NotNil(t, got.Medias, "a new place has an empty media list")
		assert.Empty(t, got.Medias)

		assert.Equal(t, laundryID, got.Category.ID)
		assert.Equal(t, "laundry", got.Category.Slug)
		assert.Equal(t, "L", got.Category.Code)
		assert.Equal(t, "Laundry", got.Category.Label)
		assert.Equal(t, "sign", got.Category.Color)

		stored, err := repo.GetPlace(ctx, got.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("leaves optional fields empty", func(t *testing.T) {
		repo := newRepository(t)

		got := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		assert.Nil(t, got.Address)
		assert.Nil(t, got.Location)
		assert.Nil(t, got.OperatingHours)
		assert.Nil(t, got.EstimatedPriceRange)
		assert.Nil(t, got.ShortDescription)
	})

	t.Run("gives each place its own id", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")

		first := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "First"))
		second := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Second"))

		assert.Less(t, first.ID, second.ID)
	})

	t.Run("rejects an unknown sub-district", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, newPlace(999, categoryID(t, "laundry"), "Laundry"))

		assert.ErrorIs(t, err, places.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})

	t.Run("rejects an unknown category", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, newPlace(insertSubDistrict(t, "Setiabudi"), 999, "Laundry"))

		assert.ErrorIs(t, err, places.ErrCategoryNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_GetPlace(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the place with its category", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "tailor"), "Penjahit"))

		got, err := repo.GetPlace(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, created, got)
		assert.Equal(t, "tailor", got.Category.Slug)
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetPlace(ctx, 999)

		assert.ErrorIs(t, err, places.ErrPlaceNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_GetMany(t *testing.T) {
	ctx := context.Background()

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
		filter    places.PlaceFilter
		want      []string
		wantTotal int
	}{
		{
			name:      "no filter returns every place by id",
			filter:    places.PlaceFilter{},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "sub-district",
			filter:    places.PlaceFilter{SubDistrictID: menteng},
			want:      []string{"Apotek Sehat", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "one category",
			filter:    places.PlaceFilter{CategoryIDs: []int64{tailor}},
			want:      []string{"Penjahit Maju", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "any of several categories",
			filter:    places.PlaceFilter{CategoryIDs: []int64{laundry, pharmacy}},
			want:      []string{"Laundry Bersih", "Apotek Sehat"},
			wantTotal: 2,
		},
		{
			name:      "query matches the name ignoring case",
			filter:    places.PlaceFilter{Query: "LAUNDRY"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query matches the address",
			filter:    places.PlaceFilter{Query: "thamrin"},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "query matches the short description",
			filter:    places.PlaceFilter{Query: "kilat"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query matches the category name",
			filter:    places.PlaceFilter{Query: "pharm"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query treats % literally",
			filter:    places.PlaceFilter{Query: "%"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query treats _ literally",
			filter:    places.PlaceFilter{Query: "_"},
			want:      []string{"Kios_Jahit"},
			wantTotal: 1,
		},
		{
			name:      "query with no match",
			filter:    places.PlaceFilter{Query: "nothing like this"},
			want:      []string{},
			wantTotal: 0,
		},
		{
			name:      "bounds keep places inside the box and drop those without a location",
			filter:    places.PlaceFilter{Bounds: &places.BoundingBox{MinLng: 106.81, MinLat: -6.22, MaxLng: 106.85, MaxLat: -6.19}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds include places on the edge",
			filter:    places.PlaceFilter{Bounds: &places.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.84, MaxLat: -6.20}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds exclude places just past the edge",
			filter:    places.PlaceFilter{Bounds: &places.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.839999, MaxLat: -6.20}},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "bounds exclude places outside the box",
			filter:    places.PlaceFilter{Bounds: &places.BoundingBox{MinLng: 106.85, MinLat: -6.40, MaxLng: 107, MaxLat: -6.25}},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name: "filters combine",
			filter: places.PlaceFilter{
				SubDistrictID: setiabudi,
				CategoryIDs:   []int64{tailor, pharmacy},
				Query:         "penjahit",
				Bounds:        &places.BoundingBox{MinLng: 106, MinLat: -7, MaxLng: 107, MaxLat: -6},
			},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "limit cuts the page and total counts every match",
			filter:    places.PlaceFilter{Limit: 2},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 4,
		},
		{
			name:      "limit equal to the matches reports them all",
			filter:    places.PlaceFilter{Limit: 4},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "limit above the matches reports them all",
			filter:    places.PlaceFilter{Limit: 10},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "offset skips the lowest ids and total counts every match",
			filter:    places.PlaceFilter{Limit: 2, Offset: 1},
			want:      []string{"Penjahit Maju", "Apotek Sehat"},
			wantTotal: 4,
		},
		{
			name:      "last page reports every match",
			filter:    places.PlaceFilter{Limit: 2, Offset: 3},
			want:      []string{"Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "offset past the matches returns no places but still the total",
			filter:    places.PlaceFilter{Limit: 2, Offset: 10},
			want:      []string{},
			wantTotal: 4,
		},
		{
			name:      "offset applies after the other filters",
			filter:    places.PlaceFilter{SubDistrictID: menteng, Limit: 1, Offset: 1},
			want:      []string{"Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "limit applies after the other filters",
			filter:    places.PlaceFilter{SubDistrictID: menteng, Limit: 1},
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

			got, total, err := repo.GetMany(ctx, tt.filter)

			require.NoError(t, err)
			assert.Equal(t, tt.want, names(got))
			assert.Equal(t, tt.wantTotal, total)
		})
	}

	t.Run("returns the category of each place", func(t *testing.T) {
		got, _, err := repo.GetMany(ctx, places.PlaceFilter{Limit: 1})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "laundry", got[0].Category.Slug)
		assert.Equal(t, "Laundry", got[0].Category.Label)
	})

	t.Run("rejects an unknown sub-district", func(t *testing.T) {
		got, total, err := repo.GetMany(ctx, places.PlaceFilter{SubDistrictID: 999})

		assert.ErrorIs(t, err, places.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		assert.Zero(t, total)
	})

	t.Run("rejects an unknown category among known ones", func(t *testing.T) {
		got, total, err := repo.GetMany(ctx, places.PlaceFilter{CategoryIDs: []int64{laundry, 999}})

		assert.ErrorIs(t, err, places.ErrCategoryNotFound)
		assert.Nil(t, got)
		assert.Zero(t, total)
	})

	t.Run("returns an empty, non-nil slice when there are no places", func(t *testing.T) {
		empty := newRepository(t)

		got, total, err := empty.GetMany(ctx, places.PlaceFilter{Limit: 10})

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
		assert.Zero(t, total)
	})
}

func TestRepository_GetCategories(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the categories the migrations define, with their fields in order", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetCategories(ctx)

		require.NoError(t, err)
		require.NotEmpty(t, got)
		for i := 1; i < len(got); i++ {
			assert.Less(t, got[i-1].ID, got[i].ID, "categories are ordered by id")
		}

		var laundry *places.Category
		for i := range got {
			require.NotNil(t, got[i].Fields, "%s has non-nil fields", got[i].Slug)
			if got[i].Slug == "laundry" {
				laundry = &got[i]
			}
		}
		require.NotNil(t, laundry)

		keys := make([]string, 0, len(laundry.Fields))
		for i, f := range laundry.Fields {
			assert.Equal(t, laundry.ID, f.CategoryID)
			assert.Equal(t, int32(i+1), f.SortOrder)
			keys = append(keys, f.Key)
		}
		assert.Equal(t, []string{"service_type", "turnaround_hours", "express_available", "minimum_weight_kg"}, keys)
	})
}

func TestRepository_GetCategory(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the category with its fields", func(t *testing.T) {
		repo := newRepository(t)
		id := categoryID(t, "tailor")

		got, err := repo.GetCategory(ctx, id)

		require.NoError(t, err)
		assert.Equal(t, id, got.ID)
		assert.Equal(t, "Tailor", got.Name)
		assert.Equal(t, "J", got.Code)
		assert.Equal(t, "Penjahit", got.Label)
		assert.Equal(t, "plum", got.Color)
		keys := make([]string, 0, len(got.Fields))
		for _, f := range got.Fields {
			keys = append(keys, f.Key)
		}
		assert.Equal(t, []string{"specialty", "turnaround_days", "same_day_service"}, keys)
	})

	t.Run("decodes the options of a select field", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetCategory(ctx, categoryID(t, "laundry"))

		require.NoError(t, err)
		serviceType := got.Fields[0]
		assert.Equal(t, "select", serviceType.DataType)
		assert.True(t, serviceType.Required)
		assert.Equal(t, places.FieldOptions{"Swalayan", "Titip cuci", "Swalayan & titip cuci"}, serviceType.Options)
		assert.Nil(t, got.Fields[1].Options, "other types have no options")
	})

	t.Run("returns empty, non-nil fields for a category without any", func(t *testing.T) {
		repo := newRepository(t)
		var id int64
		require.NoError(t, testDB.Get(&id, `INSERT INTO CATEGORIES (name, slug, code, label, color) VALUES ('Bare', 'bare', 'Z', 'Bare', 'cat') RETURNING id`))
		t.Cleanup(func() { _, _ = testDB.Exec(`DELETE FROM CATEGORIES WHERE id = $1`, id) })

		got, err := repo.GetCategory(ctx, id)

		require.NoError(t, err)
		assert.NotNil(t, got.Fields)
		assert.Empty(t, got.Fields)
	})

	t.Run("returns ErrCategoryNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetCategory(ctx, 999)

		assert.ErrorIs(t, err, places.ErrCategoryNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("stores every field and returns the place with its new category", func(t *testing.T) {
		repo := newRepository(t)
		setiabudi := insertSubDistrict(t, "Setiabudi")
		menteng := insertSubDistrict(t, "Menteng")
		created := mustCreate(t, repo, newPlace(setiabudi, categoryID(t, "laundry"), "Old"))
		tailorID := categoryID(t, "tailor")

		got, err := repo.Update(ctx, &places.Place{
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
		require.NoError(t, err)

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, menteng, got.SubDistrictID)
		assert.Equal(t, tailorID, got.CategoryID)
		assert.Equal(t, "New", got.Name)
		assert.Equal(t, new("Jl. Baru"), got.Address)
		assert.Equal(t, new("24 jam"), got.OperatingHours)
		assert.Equal(t, new("Rp1"), got.EstimatedPriceRange)
		assert.Equal(t, new("Baru"), got.ShortDescription)
		assert.JSONEq(t, `{"specialty": "Permak"}`, string(got.Attributes))
		require.NotNil(t, got.Location)
		assert.InDelta(t, 106.9, got.Location.X(), 1e-9)
		assert.Equal(t, "tailor", got.Category.Slug)

		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("clears optional fields that are nil", func(t *testing.T) {
		repo := newRepository(t)
		p := newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry")
		p.Address = new("Jl. Lama")
		p.Location = postgis.MustPoint(106.8, -6.2)
		created := mustCreate(t, repo, p)

		got, err := repo.Update(ctx, newPlaceWithID(created, "Laundry"))

		require.NoError(t, err)
		assert.Nil(t, got.Address)
		assert.Nil(t, got.Location)
	})

	t.Run("keeps the ratings and the medias", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1.jpg")
		_, err := testDB.Exec(`UPDATE PLACES SET rating_count = 2, rating_sum = 9 WHERE id = $1`, created.ID)
		require.NoError(t, err)

		got, err := repo.Update(ctx, newPlaceWithID(created, "Renamed"))

		require.NoError(t, err)
		assert.Equal(t, "Renamed", got.Name)
		assert.Equal(t, 2, got.RatingCount)
		assert.Equal(t, 9, got.RatingSum)
		assert.Equal(t, []storage.Media{m}, got.Medias)
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)
		p := newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry")
		p.ID = 999

		got, err := repo.Update(ctx, p)

		assert.ErrorIs(t, err, places.ErrPlaceNotFound)
		assert.Nil(t, got)
	})

	t.Run("rejects an unknown sub-district and keeps the place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		update := newPlaceWithID(created, "Renamed")
		update.SubDistrictID = 999

		got, err := repo.Update(ctx, update)

		assert.ErrorIs(t, err, places.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Laundry", stored.Name)
	})

	t.Run("rejects an unknown category and keeps the place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		update := newPlaceWithID(created, "Renamed")
		update.CategoryID = 999

		got, err := repo.Update(ctx, update)

		assert.ErrorIs(t, err, places.ErrCategoryNotFound)
		assert.Nil(t, got)
		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Laundry", stored.Name)
	})
}

// newPlaceWithID returns a copy of p's identity and ownership with the given
// name and no optional fields, to use as an update.
func newPlaceWithID(p *places.Place, name string) *places.Place {
	update := newPlace(p.SubDistrictID, p.CategoryID, name)
	update.ID = p.ID

	return update
}

func TestRepository_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		require.NoError(t, repo.Delete(ctx, created.ID))

		_, err := repo.GetPlace(ctx, created.ID)
		assert.ErrorIs(t, err, places.ErrPlaceNotFound)
		assert.Empty(t, queuedKeys(t), "a place without medias queues nothing")
	})

	t.Run("removes the medias and queues their images for deletion", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		created := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Laundry"))
		kept := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Kept"))
		mustAddMedia(t, repo, created.ID, "places/1/a.jpg")
		mustAddMedia(t, repo, created.ID, "places/1/b.jpg")
		keptMedia := mustAddMedia(t, repo, kept.ID, "places/2/a.jpg")

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.ElementsMatch(t, []string{"places/1/a.jpg", "places/1/b.jpg"}, queuedKeys(t))
		assert.Equal(t, 1, storedMediaCount(t), "only the other place's media is left")
		stored, err := repo.GetPlace(ctx, kept.ID)
		require.NoError(t, err)
		assert.Equal(t, []storage.Media{keptMedia}, stored.Medias)
	})

	t.Run("tolerates an image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1.jpg")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/1.jpg')`)
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"places/1.jpg"}, queuedKeys(t))
	})

	t.Run("leaves other places alone", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		doomed := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Doomed"))
		kept := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Kept"))

		require.NoError(t, repo.Delete(ctx, doomed.ID))

		got, total, err := repo.GetMany(ctx, places.PlaceFilter{Limit: 10})
		require.NoError(t, err)
		assert.Equal(t, []string{kept.Name}, names(got))
		assert.Equal(t, 1, total)
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(ctx, 999), places.ErrPlaceNotFound)
	})

	t.Run("returns ErrPlaceNotFound the second time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.ErrorIs(t, repo.Delete(ctx, created.ID), places.ErrPlaceNotFound)
	})
}

// newMedia returns a promoted image to store under key.
func newMedia(key string) storage.Media {
	return storage.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

// mustAddMedia stores an image under key for the place and returns it.
func mustAddMedia(t *testing.T, repo places.Repository, placeID int64, key string) storage.Media {
	t.Helper()

	got, err := repo.AddMedia(context.Background(), placeID, newMedia(key))
	require.NoError(t, err)

	for _, m := range got.Medias {
		if m.Key == key {
			return m
		}
	}
	require.FailNow(t, "the added media is missing", "key %q", key)
	return storage.Media{}
}

// mediaKeys returns the keys of medias, in order.
func mediaKeys(medias []storage.Media) []string {
	out := make([]string, 0, len(medias))
	for _, m := range medias {
		out = append(out, m.Key)
	}

	return out
}

// mediaPositions returns the positions of medias, in order.
func mediaPositions(medias []storage.Media) []int {
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
	require.NoError(t, testDB.Get(&count, `SELECT COUNT(*) FROM MEDIAS`))
	return count
}

func TestRepository_AddMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the media at the first position and returns the place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/a.webp"))

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "laundry", got.Category.Slug)
		require.Len(t, got.Medias, 1)
		assert.NotZero(t, got.Medias[0].ID)
		assert.Equal(t, "places/1/a.webp", got.Medias[0].Key)
		assert.Equal(t, "image/webp", got.Medias[0].ContentType)
		assert.Equal(t, int64(2048), got.Medias[0].Size)
		assert.Equal(t, 1, got.Medias[0].Position)
		assert.Empty(t, queuedKeys(t), "adding a media replaces nothing")

		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("adds each media after the last, in position order", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		mustAddMedia(t, repo, created.ID, "places/1/b.webp")

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/c.webp"))

		require.NoError(t, err)
		assert.Equal(t, []string{"places/1/a.webp", "places/1/b.webp", "places/1/c.webp"}, mediaKeys(got.Medias))
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias))
	})

	t.Run("appends after the remaining medias once one is deleted", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		removed := mustAddMedia(t, repo, created.ID, "places/1/b.webp")
		mustAddMedia(t, repo, created.ID, "places/1/c.webp")
		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/d.webp"))

		require.NoError(t, err)
		assert.Equal(t, []string{"places/1/a.webp", "places/1/c.webp", "places/1/d.webp"}, mediaKeys(got.Medias))
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias))
	})

	t.Run("returns ErrMediaLimitReached once the place has the most medias", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		for i := range storage.MaxMedias {
			mustAddMedia(t, repo, created.ID, fmt.Sprintf("places/1/%d.webp", i))
		}

		got, err := repo.AddMedia(ctx, created.ID, newMedia("places/1/extra.webp"))

		assert.ErrorIs(t, err, storage.ErrMediaLimitReached)
		assert.Nil(t, got)
		assert.Equal(t, storage.MaxMedias, storedMediaCount(t), "the extra media must not be stored")
	})

	t.Run("keeps the limit under concurrent adds", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		const attempts = storage.MaxMedias + 4

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
			case err == nil:
				added++
			case errors.Is(err, storage.ErrMediaLimitReached):
				limited++
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}
		assert.Equal(t, storage.MaxMedias, added)
		assert.Equal(t, attempts-storage.MaxMedias, limited)

		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, mediaPositions(stored.Medias))
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.AddMedia(ctx, 999, newMedia("places/1/a.webp"))

		assert.ErrorIs(t, err, places.ErrPlaceNotFound)
		assert.Nil(t, got)
		assert.Zero(t, storedMediaCount(t))
	})
}

func TestRepository_DeleteMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the media and queues its image for deletion", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		removed := mustAddMedia(t, repo, created.ID, "places/1/b.webp")
		mustAddMedia(t, repo, created.ID, "places/1/c.webp")

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"places/1/a.webp", "places/1/c.webp"}, mediaKeys(stored.Medias))
		assert.Equal(t, []int{1, 2}, mediaPositions(stored.Medias), "the later medias shift down")
		assert.Equal(t, 2, storedMediaCount(t))
		assert.Equal(t, []string{"places/1/b.webp"}, queuedKeys(t))
	})

	t.Run("tolerates an image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		removed := mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/1/a.webp')`)
		require.NoError(t, err)

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		assert.Equal(t, []string{"places/1/a.webp"}, queuedKeys(t))
	})

	t.Run("returns ErrMediaNotFound for a media of another place", func(t *testing.T) {
		repo := newRepository(t)
		subDistrictID := insertSubDistrict(t, "Setiabudi")
		laundryID := categoryID(t, "laundry")
		owner := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Owner"))
		other := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Other"))
		m := mustAddMedia(t, repo, owner.ID, "places/1/a.webp")

		err := repo.DeleteMedia(ctx, other.ID, m.ID)

		assert.ErrorIs(t, err, storage.ErrMediaNotFound)
		assert.Equal(t, 1, storedMediaCount(t), "the media must be kept")
		assert.Empty(t, queuedKeys(t))
	})

	t.Run("returns ErrMediaNotFound for an unknown media or place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1/a.webp")

		assert.ErrorIs(t, repo.DeleteMedia(ctx, created.ID, 999), storage.ErrMediaNotFound)
		assert.ErrorIs(t, repo.DeleteMedia(ctx, 999, m.ID), storage.ErrMediaNotFound)
		assert.Equal(t, 1, storedMediaCount(t))
	})

	t.Run("returns ErrMediaNotFound the second time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		m := mustAddMedia(t, repo, created.ID, "places/1/a.webp")
		require.NoError(t, repo.DeleteMedia(ctx, created.ID, m.ID))

		assert.ErrorIs(t, repo.DeleteMedia(ctx, created.ID, m.ID), storage.ErrMediaNotFound)
	})
}

func TestRepository_GetMany_Medias(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	subDistrictID := insertSubDistrict(t, "Setiabudi")
	laundryID := categoryID(t, "laundry")
	first := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "First"))
	second := mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Second"))
	mustCreate(t, repo, newPlace(subDistrictID, laundryID, "Without medias"))
	mustAddMedia(t, repo, first.ID, "places/1/a.webp")
	mustAddMedia(t, repo, second.ID, "places/2/a.webp")
	mustAddMedia(t, repo, first.ID, "places/1/b.webp")

	got, _, err := repo.GetMany(ctx, places.PlaceFilter{Limit: 10})

	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"places/1/a.webp", "places/1/b.webp"}, mediaKeys(got[0].Medias))
	assert.Equal(t, []string{"places/2/a.webp"}, mediaKeys(got[1].Medias))
	assert.NotNil(t, got[2].Medias, "a place without medias has an empty list")
	assert.Empty(t, got[2].Medias)
}
