//go:build integration

package place_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/place"
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

// newRepository returns a repository over an emptied database.
func newRepository(t *testing.T) place.Repository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return place.NewPostgresRepository(testDB)
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
func newPlace(subDistrictID, categoryID int64, name string) *place.Place {
	return &place.Place{
		SubDistrictID: subDistrictID,
		CategoryID:    categoryID,
		Name:          name,
		Attributes:    json.RawMessage(`{}`),
	}
}

func mustCreate(t *testing.T, repo place.Repository, p *place.Place) *place.Place {
	t.Helper()

	created, err := repo.Create(context.Background(), p)
	require.NoError(t, err)

	return created
}

func names(places []place.Place) []string {
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

		got, err := repo.Create(ctx, &place.Place{
			SubDistrictID:       subDistrictID,
			CategoryID:          laundryID,
			Name:                "Laundry Bersih",
			Address:             ptr("Jl. Sudirman 1"),
			Location:            postgis.MustPoint(106.82, -6.2),
			OperatingHours:      ptr("08:00-20:00"),
			EstimatedPriceRange: ptr("Rp5.000-Rp10.000"),
			ShortDescription:    ptr("Cuci kilat"),
			Attributes:          json.RawMessage(`{"service_type": "Swalayan"}`),
		})
		require.NoError(t, err)

		assert.NotZero(t, got.ID)
		assert.Equal(t, subDistrictID, got.SubDistrictID)
		assert.Equal(t, laundryID, got.CategoryID)
		assert.Equal(t, "Laundry Bersih", got.Name)
		assert.Equal(t, ptr("Jl. Sudirman 1"), got.Address)
		assert.Equal(t, ptr("08:00-20:00"), got.OperatingHours)
		assert.Equal(t, ptr("Rp5.000-Rp10.000"), got.EstimatedPriceRange)
		assert.Equal(t, ptr("Cuci kilat"), got.ShortDescription)
		assert.JSONEq(t, `{"service_type": "Swalayan"}`, string(got.Attributes))
		require.NotNil(t, got.Location)
		assert.InDelta(t, 106.82, got.Location.X(), 1e-9)
		assert.InDelta(t, -6.2, got.Location.Y(), 1e-9)
		assert.Equal(t, 0, got.RatingCount)
		assert.Equal(t, 0, got.RatingSum)
		assert.Nil(t, got.ImageKey)

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

		assert.ErrorIs(t, err, place.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})

	t.Run("rejects an unknown category", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, newPlace(insertSubDistrict(t, "Setiabudi"), 999, "Laundry"))

		assert.ErrorIs(t, err, place.ErrCategoryNotFound)
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

		assert.ErrorIs(t, err, place.ErrPlaceNotFound)
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
	laundryPlace.Address = ptr("Jl. Sudirman 1")
	laundryPlace.ShortDescription = ptr("Cuci kilat")
	laundryPlace.Location = postgis.MustPoint(106.82, -6.20)
	mustCreate(t, repo, laundryPlace)

	tailorPlace := newPlace(setiabudi, tailor, "Penjahit Maju")
	tailorPlace.Address = ptr("Jl. Thamrin 2")
	tailorPlace.Location = postgis.MustPoint(106.84, -6.21)
	mustCreate(t, repo, tailorPlace)

	pharmacyPlace := newPlace(menteng, pharmacy, "Apotek Sehat")
	pharmacyPlace.ShortDescription = ptr("Diskon 100% obat")
	pharmacyPlace.Location = postgis.MustPoint(106.90, -6.30)
	mustCreate(t, repo, pharmacyPlace)

	mustCreate(t, repo, newPlace(menteng, tailor, "Kios_Jahit"))

	tests := []struct {
		name      string
		filter    place.PlaceFilter
		want      []string
		wantTotal int
	}{
		{
			name:      "no filter returns every place by id",
			filter:    place.PlaceFilter{},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "sub-district",
			filter:    place.PlaceFilter{SubDistrictID: menteng},
			want:      []string{"Apotek Sehat", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "one category",
			filter:    place.PlaceFilter{CategoryIDs: []int64{tailor}},
			want:      []string{"Penjahit Maju", "Kios_Jahit"},
			wantTotal: 2,
		},
		{
			name:      "any of several categories",
			filter:    place.PlaceFilter{CategoryIDs: []int64{laundry, pharmacy}},
			want:      []string{"Laundry Bersih", "Apotek Sehat"},
			wantTotal: 2,
		},
		{
			name:      "query matches the name ignoring case",
			filter:    place.PlaceFilter{Query: "LAUNDRY"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query matches the address",
			filter:    place.PlaceFilter{Query: "thamrin"},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "query matches the short description",
			filter:    place.PlaceFilter{Query: "kilat"},
			want:      []string{"Laundry Bersih"},
			wantTotal: 1,
		},
		{
			name:      "query matches the category name",
			filter:    place.PlaceFilter{Query: "pharm"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query treats % literally",
			filter:    place.PlaceFilter{Query: "%"},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name:      "query treats _ literally",
			filter:    place.PlaceFilter{Query: "_"},
			want:      []string{"Kios_Jahit"},
			wantTotal: 1,
		},
		{
			name:      "query with no match",
			filter:    place.PlaceFilter{Query: "nothing like this"},
			want:      []string{},
			wantTotal: 0,
		},
		{
			name:      "bounds keep places inside the box and drop those without a location",
			filter:    place.PlaceFilter{Bounds: &place.BoundingBox{MinLng: 106.81, MinLat: -6.22, MaxLng: 106.85, MaxLat: -6.19}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds include places on the edge",
			filter:    place.PlaceFilter{Bounds: &place.BoundingBox{MinLng: 106.82, MinLat: -6.21, MaxLng: 106.84, MaxLat: -6.20}},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 2,
		},
		{
			name:      "bounds exclude places outside the box",
			filter:    place.PlaceFilter{Bounds: &place.BoundingBox{MinLng: 106.85, MinLat: -6.40, MaxLng: 107, MaxLat: -6.25}},
			want:      []string{"Apotek Sehat"},
			wantTotal: 1,
		},
		{
			name: "filters combine",
			filter: place.PlaceFilter{
				SubDistrictID: setiabudi,
				CategoryIDs:   []int64{tailor, pharmacy},
				Query:         "penjahit",
				Bounds:        &place.BoundingBox{MinLng: 106, MinLat: -7, MaxLng: 107, MaxLat: -6},
			},
			want:      []string{"Penjahit Maju"},
			wantTotal: 1,
		},
		{
			name:      "limit cuts the page and total counts every match",
			filter:    place.PlaceFilter{Limit: 2},
			want:      []string{"Laundry Bersih", "Penjahit Maju"},
			wantTotal: 4,
		},
		{
			name:      "limit equal to the matches reports them all",
			filter:    place.PlaceFilter{Limit: 4},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "limit above the matches reports them all",
			filter:    place.PlaceFilter{Limit: 10},
			want:      []string{"Laundry Bersih", "Penjahit Maju", "Apotek Sehat", "Kios_Jahit"},
			wantTotal: 4,
		},
		{
			name:      "limit applies after the other filters",
			filter:    place.PlaceFilter{SubDistrictID: menteng, Limit: 1},
			want:      []string{"Apotek Sehat"},
			wantTotal: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total, err := repo.GetMany(ctx, tt.filter)

			require.NoError(t, err)
			assert.Equal(t, tt.want, names(got))
			assert.Equal(t, tt.wantTotal, total)
		})
	}

	t.Run("returns the category of each place", func(t *testing.T) {
		got, _, err := repo.GetMany(ctx, place.PlaceFilter{Limit: 1})

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "laundry", got[0].Category.Slug)
		assert.Equal(t, "Laundry", got[0].Category.Label)
	})

	t.Run("rejects an unknown sub-district", func(t *testing.T) {
		got, total, err := repo.GetMany(ctx, place.PlaceFilter{SubDistrictID: 999})

		assert.ErrorIs(t, err, place.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		assert.Zero(t, total)
	})

	t.Run("rejects an unknown category among known ones", func(t *testing.T) {
		got, total, err := repo.GetMany(ctx, place.PlaceFilter{CategoryIDs: []int64{laundry, 999}})

		assert.ErrorIs(t, err, place.ErrCategoryNotFound)
		assert.Nil(t, got)
		assert.Zero(t, total)
	})

	t.Run("returns an empty, non-nil slice when there are no places", func(t *testing.T) {
		empty := newRepository(t)

		got, total, err := empty.GetMany(ctx, place.PlaceFilter{})

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

		var laundry *place.Category
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
		assert.Equal(t, place.FieldOptions{"Swalayan", "Titip cuci", "Swalayan & titip cuci"}, serviceType.Options)
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

		assert.ErrorIs(t, err, place.ErrCategoryNotFound)
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

		got, err := repo.Update(ctx, &place.Place{
			ID:                  created.ID,
			SubDistrictID:       menteng,
			CategoryID:          tailorID,
			Name:                "New",
			Address:             ptr("Jl. Baru"),
			Location:            postgis.MustPoint(106.9, -6.3),
			OperatingHours:      ptr("24 jam"),
			EstimatedPriceRange: ptr("Rp1"),
			ShortDescription:    ptr("Baru"),
			Attributes:          json.RawMessage(`{"specialty": "Permak"}`),
		})
		require.NoError(t, err)

		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, menteng, got.SubDistrictID)
		assert.Equal(t, tailorID, got.CategoryID)
		assert.Equal(t, "New", got.Name)
		assert.Equal(t, ptr("Jl. Baru"), got.Address)
		assert.Equal(t, ptr("24 jam"), got.OperatingHours)
		assert.Equal(t, ptr("Rp1"), got.EstimatedPriceRange)
		assert.Equal(t, ptr("Baru"), got.ShortDescription)
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
		p.Address = ptr("Jl. Lama")
		p.Location = postgis.MustPoint(106.8, -6.2)
		created := mustCreate(t, repo, p)

		got, err := repo.Update(ctx, newPlaceWithID(created, "Laundry"))

		require.NoError(t, err)
		assert.Nil(t, got.Address)
		assert.Nil(t, got.Location)
	})

	t.Run("keeps the ratings and the image", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")
		require.NoError(t, err)
		_, err = testDB.Exec(`UPDATE PLACES SET rating_count = 2, rating_sum = 9 WHERE id = $1`, created.ID)
		require.NoError(t, err)

		got, err := repo.Update(ctx, newPlaceWithID(created, "Renamed"))

		require.NoError(t, err)
		assert.Equal(t, "Renamed", got.Name)
		assert.Equal(t, 2, got.RatingCount)
		assert.Equal(t, 9, got.RatingSum)
		assert.Equal(t, ptr("places/1.jpg"), got.ImageKey)
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)
		p := newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry")
		p.ID = 999

		got, err := repo.Update(ctx, p)

		assert.ErrorIs(t, err, place.ErrPlaceNotFound)
		assert.Nil(t, got)
	})

	t.Run("rejects an unknown sub-district and keeps the place", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		update := newPlaceWithID(created, "Renamed")
		update.SubDistrictID = 999

		got, err := repo.Update(ctx, update)

		assert.ErrorIs(t, err, place.ErrSubDistrictNotFound)
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

		assert.ErrorIs(t, err, place.ErrCategoryNotFound)
		assert.Nil(t, got)
		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "Laundry", stored.Name)
	})
}

// newPlaceWithID returns a copy of p's identity and ownership with the given
// name and no optional fields, to use as an update.
func newPlaceWithID(p *place.Place, name string) *place.Place {
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
		assert.ErrorIs(t, err, place.ErrPlaceNotFound)
		assert.Empty(t, queuedKeys(t), "a place without an image queues nothing")
	})

	t.Run("queues the image for deletion", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"places/1.jpg"}, queuedKeys(t))
	})

	t.Run("tolerates an image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")
		require.NoError(t, err)
		_, err = testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/1.jpg')`)
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

		got, total, err := repo.GetMany(ctx, place.PlaceFilter{})
		require.NoError(t, err)
		assert.Equal(t, []string{kept.Name}, names(got))
		assert.Equal(t, 1, total)
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(ctx, 999), place.ErrPlaceNotFound)
	})

	t.Run("returns ErrPlaceNotFound the second time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.ErrorIs(t, repo.Delete(ctx, created.ID), place.ErrPlaceNotFound)
	})
}

func TestRepository_SetImageKey(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the key and returns the place with its category", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))

		got, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, ptr("places/1.jpg"), got.ImageKey)
		assert.Equal(t, "laundry", got.Category.Slug)
		assert.Empty(t, queuedKeys(t), "a first image replaces nothing")

		stored, err := repo.GetPlace(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, ptr("places/1.jpg"), stored.ImageKey)
	})

	t.Run("queues the image it replaces", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/old.jpg")
		require.NoError(t, err)

		got, err := repo.SetImageKey(ctx, created.ID, "places/new.jpg")

		require.NoError(t, err)
		assert.Equal(t, ptr("places/new.jpg"), got.ImageKey)
		assert.Equal(t, []string{"places/old.jpg"}, queuedKeys(t))
	})

	t.Run("does not queue the key it keeps", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")
		require.NoError(t, err)

		got, err := repo.SetImageKey(ctx, created.ID, "places/1.jpg")

		require.NoError(t, err)
		assert.Equal(t, ptr("places/1.jpg"), got.ImageKey)
		assert.Empty(t, queuedKeys(t))
	})

	t.Run("tolerates a replaced image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newPlace(insertSubDistrict(t, "Setiabudi"), categoryID(t, "laundry"), "Laundry"))
		_, err := repo.SetImageKey(ctx, created.ID, "places/old.jpg")
		require.NoError(t, err)
		_, err = testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('places/old.jpg')`)
		require.NoError(t, err)

		_, err = repo.SetImageKey(ctx, created.ID, "places/new.jpg")

		require.NoError(t, err)
		assert.Equal(t, []string{"places/old.jpg"}, queuedKeys(t))
	})

	t.Run("returns ErrPlaceNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.SetImageKey(ctx, 999, "places/1.jpg")

		assert.ErrorIs(t, err, place.ErrPlaceNotFound)
		assert.Nil(t, got)
		assert.Empty(t, queuedKeys(t))
	})
}
