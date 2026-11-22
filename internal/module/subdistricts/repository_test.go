//go:build integration

package subdistricts_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/subdistricts"
	"rantau-go/internal/storage"
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
func newRepository(t *testing.T) subdistricts.Repository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return subdistricts.NewPostgresRepository(testDB)
}

// newSubDistrict returns a sub-district that is valid to create.
func newSubDistrict(name, slug string) *subdistricts.SubDistrict {
	return &subdistricts.SubDistrict{Name: name, Slug: slug}
}

func mustCreate(t *testing.T, repo subdistricts.Repository, sd *subdistricts.SubDistrict) *subdistricts.SubDistrict {
	t.Helper()

	created, err := repo.Create(context.Background(), sd)
	require.NoError(t, err)

	return created
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the sub-district and returns it", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, &subdistricts.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir 2020"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(75000.5),
		})

		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, "Menteng", got.Name)
		assert.Equal(t, "menteng", got.Slug)
		assert.Equal(t, new("Banjir 2020"), got.FloodHistory)
		assert.Equal(t, new("Aman"), got.SafetyCrimeInformation)
		assert.Equal(t, new(75000.5), got.EstimatedDailySpending)
		assert.NotNil(t, got.Medias, "a new sub-district has an empty media list")
		assert.Empty(t, got.Medias)
		assert.Nil(t, got.Boundary)

		stored, err := repo.GetSubDistrict(ctx, got.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("leaves optional fields empty", func(t *testing.T) {
		repo := newRepository(t)

		got := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		assert.Nil(t, got.FloodHistory)
		assert.Nil(t, got.SafetyCrimeInformation)
		assert.Nil(t, got.EstimatedDailySpending)
	})

	conflicts := []struct {
		desc       string
		name       string
		slug       string
		wantExists bool
	}{
		{"the same name", "Menteng", "other", true},
		{"the same name in other case", "MENTENG", "other", true},
		{"the same slug", "Other", "menteng", true},
		{"a different name and slug", "Other", "other", false},
	}
	for _, c := range conflicts {
		t.Run("with "+c.desc, func(t *testing.T) {
			repo := newRepository(t)
			mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

			got, err := repo.Create(ctx, newSubDistrict(c.name, c.slug))

			if c.wantExists {
				assert.ErrorIs(t, err, subdistricts.ErrSubDistrictExists)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRepository_GetMany(t *testing.T) {
	ctx := context.Background()

	t.Run("returns every sub-district in id order", func(t *testing.T) {
		repo := newRepository(t)
		zebra := mustCreate(t, repo, newSubDistrict("Zebra", "zebra"))
		alpha := mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))

		got, err := repo.GetMany(ctx)

		require.NoError(t, err)
		assert.Equal(t, []subdistricts.SubDistrict{*zebra, *alpha}, got)
	})

	t.Run("returns an empty, non-nil slice when there are none", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetMany(ctx)

		require.NoError(t, err)
		assert.NotNil(t, got)
		assert.Empty(t, got)
	})
}

func TestRepository_GetSubDistrict(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the sub-district", func(t *testing.T) {
		repo := newRepository(t)
		mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.GetSubDistrict(ctx, created.ID)

		require.NoError(t, err)
		assert.Equal(t, created, got)
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.GetSubDistrict(ctx, 999)

		assert.ErrorIs(t, err, subdistricts.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("stores every field and returns the sub-district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Old", "old"))

		got, err := repo.Update(ctx, &subdistricts.SubDistrict{
			ID:                     created.ID,
			Name:                   "New",
			Slug:                   "new",
			FloodHistory:           new("Banjir"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(50000.0),
		})

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "New", got.Name)
		assert.Equal(t, "new", got.Slug)
		assert.Equal(t, new("Banjir"), got.FloodHistory)
		assert.Equal(t, new("Aman"), got.SafetyCrimeInformation)
		assert.Equal(t, new(50000.0), got.EstimatedDailySpending)

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("clears optional fields that are nil", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, &subdistricts.SubDistrict{
			Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir"), EstimatedDailySpending: new(1.0),
		})

		got, err := repo.Update(ctx, &subdistricts.SubDistrict{ID: created.ID, Name: "Menteng", Slug: "menteng"})

		require.NoError(t, err)
		assert.Nil(t, got.FloodHistory)
		assert.Nil(t, got.EstimatedDailySpending)
	})

	t.Run("keeps the images and the boundary", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "photo.jpg")
		mustSetBoundary(t, repo, created.ID)

		got, err := repo.Update(ctx, &subdistricts.SubDistrict{ID: created.ID, Name: "Renamed", Slug: "renamed"})

		require.NoError(t, err)
		assert.Equal(t, []storage.Media{m}, got.Medias)
		assert.Equal(t, mustBoundary(t, squareJSON), got.Boundary)
	})

	t.Run("accepts the sub-district's own name and slug, even in another case", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.Update(ctx, &subdistricts.SubDistrict{ID: created.ID, Name: "MENTENG", Slug: "menteng"})

		require.NoError(t, err)
		assert.Equal(t, "MENTENG", got.Name)
	})

	t.Run("returns ErrSubDistrictExists for another sub-district's name or slug and keeps the row", func(t *testing.T) {
		for _, c := range []struct{ name, slug string }{
			{"alpha", "new"},
			{"New", "alpha"},
		} {
			t.Run(c.name+"/"+c.slug, func(t *testing.T) {
				repo := newRepository(t)
				mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))
				created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

				got, err := repo.Update(ctx, &subdistricts.SubDistrict{ID: created.ID, Name: c.name, Slug: c.slug})

				assert.ErrorIs(t, err, subdistricts.ErrSubDistrictExists)
				assert.Nil(t, got)
				stored, err := repo.GetSubDistrict(ctx, created.ID)
				require.NoError(t, err)
				assert.Equal(t, created, stored)
			})
		}
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Update(ctx, &subdistricts.SubDistrict{ID: 999, Name: "Menteng", Slug: "menteng"})

		assert.ErrorIs(t, err, subdistricts.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the sub-district and leaves the others", func(t *testing.T) {
		repo := newRepository(t)
		doomed := mustCreate(t, repo, newSubDistrict("Doomed", "doomed"))
		kept := mustCreate(t, repo, newSubDistrict("Kept", "kept"))

		require.NoError(t, repo.Delete(ctx, doomed.ID))

		got, err := repo.GetMany(ctx)
		require.NoError(t, err)
		assert.Equal(t, []subdistricts.SubDistrict{*kept}, got)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "no images, nothing queued")
	})

	t.Run("removes the medias and queues their images", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		kept := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
		mustAddMedia(t, repo, created.ID, "a.jpg")
		mustAddMedia(t, repo, created.ID, "b.jpg")
		mustSetBoundary(t, repo, created.ID)
		mustAddMedia(t, repo, kept.ID, "kept.jpg")

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"a.jpg", "b.jpg"}, pgtest.QueuedKeys(t, testDB))
		assert.Equal(t, []string{"kept.jpg"}, storedMediaKeys(t), "only the other sub-district's images are left")
	})

	t.Run("tolerates images that are already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "photo.png")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('photo.png')`)
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"photo.png"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("returns ErrSubDistrictInUse while places reference it, and keeps everything", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "photo.jpg")
		mustSetBoundary(t, repo, created.ID)
		pgtest.InsertPlace(t, testDB, created.ID, pgtest.CategoryID(t, testDB, "laundry"), "Laundry")

		err := repo.Delete(ctx, created.ID)

		assert.ErrorIs(t, err, subdistricts.ErrSubDistrictInUse)
		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, []storage.Media{m}, stored.Medias)
		assert.Equal(t, mustBoundary(t, squareJSON), stored.Boundary)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "the images are still in use")
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(ctx, 999), subdistricts.ErrSubDistrictNotFound)
	})

	t.Run("returns ErrSubDistrictNotFound the second time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.ErrorIs(t, repo.Delete(ctx, created.ID), subdistricts.ErrSubDistrictNotFound)
	})
}

// newMedia returns a promoted image to store under key.
func newMedia(key string) storage.Media {
	return storage.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

// mustAddMedia stores an image under key for the sub-district and returns it.
func mustAddMedia(t *testing.T, repo subdistricts.Repository, id int64, key string) storage.Media {
	t.Helper()

	got, err := repo.AddMedia(context.Background(), id, newMedia(key))
	require.NoError(t, err)

	for _, m := range got.Medias {
		if m.Key == key {
			return m
		}
	}
	require.FailNow(t, "the added media is missing", "key %q", key)
	return storage.Media{}
}

// mustSetBoundary stores squareJSON as the sub-district's boundary.
func mustSetBoundary(t *testing.T, repo subdistricts.Repository, id int64) {
	t.Helper()

	_, err := repo.SetBoundary(context.Background(), id, mustBoundary(t, squareJSON))
	require.NoError(t, err)
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

// storedMediaKeys returns the keys of every MEDIAS row, sorted.
func storedMediaKeys(t *testing.T) []string {
	t.Helper()

	keys := make([]string, 0)
	require.NoError(t, testDB.Select(&keys, `SELECT key FROM MEDIAS ORDER BY key`))
	return keys
}

func TestRepository_AddMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the media at the first position and returns the sub-district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.AddMedia(ctx, created.ID, newMedia("subdistricts/1/medias/a.webp"))

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		require.Len(t, got.Medias, 1)
		assert.NotZero(t, got.Medias[0].ID)
		assert.Equal(t, "subdistricts/1/medias/a.webp", got.Medias[0].Key)
		assert.Equal(t, "image/webp", got.Medias[0].ContentType)
		assert.Equal(t, int64(2048), got.Medias[0].Size)
		assert.Equal(t, 1, got.Medias[0].Position)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("appends after the remaining medias once one is deleted", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "a.webp")
		removed := mustAddMedia(t, repo, created.ID, "b.webp")
		mustAddMedia(t, repo, created.ID, "c.webp")
		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		got, err := repo.AddMedia(ctx, created.ID, newMedia("d.webp"))

		require.NoError(t, err)
		assert.Equal(t, []string{"a.webp", "c.webp", "d.webp"}, mediaKeys(got.Medias))
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias))
	})

	t.Run("returns ErrMediaLimitReached once the sub-district has the most medias", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		for i := range storage.MaxMedias {
			mustAddMedia(t, repo, created.ID, fmt.Sprintf("%d.webp", i))
		}

		got, err := repo.AddMedia(ctx, created.ID, newMedia("extra.webp"))

		assert.ErrorIs(t, err, storage.ErrMediaLimitReached)
		assert.Nil(t, got)
		assert.Len(t, storedMediaKeys(t), storage.MaxMedias, "the extra media must not be stored")
	})

	t.Run("keeps the limit under concurrent adds", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		const attempts = storage.MaxMedias + 4

		errs := make([]error, attempts)
		var wg sync.WaitGroup
		for i := range attempts {
			wg.Go(func() {
				_, errs[i] = repo.AddMedia(ctx, created.ID, newMedia(fmt.Sprintf("%d.webp", i)))
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

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, mediaPositions(stored.Medias))
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.AddMedia(ctx, 999, newMedia("a.webp"))

		assert.ErrorIs(t, err, subdistricts.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		assert.Empty(t, storedMediaKeys(t))
	})
}

func TestRepository_DeleteMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("removes the media and queues its image for deletion", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "a.webp")
		removed := mustAddMedia(t, repo, created.ID, "b.webp")
		mustAddMedia(t, repo, created.ID, "c.webp")

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, []string{"a.webp", "c.webp"}, mediaKeys(stored.Medias))
		assert.Equal(t, []int{1, 2}, mediaPositions(stored.Medias), "the later medias shift down")
		assert.Equal(t, []string{"a.webp", "c.webp"}, storedMediaKeys(t))
		assert.Equal(t, []string{"b.webp"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("tolerates an image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		removed := mustAddMedia(t, repo, created.ID, "a.webp")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('a.webp')`)
		require.NoError(t, err)

		require.NoError(t, repo.DeleteMedia(ctx, created.ID, removed.ID))

		assert.Equal(t, []string{"a.webp"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("returns ErrMediaNotFound for a media of another sub-district", func(t *testing.T) {
		repo := newRepository(t)
		owner := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		other := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
		m := mustAddMedia(t, repo, owner.ID, "a.webp")

		err := repo.DeleteMedia(ctx, other.ID, m.ID)

		assert.ErrorIs(t, err, storage.ErrMediaNotFound)
		assert.Equal(t, []string{"a.webp"}, storedMediaKeys(t), "the media must be kept")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("returns ErrMediaNotFound for an unknown media or sub-district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "a.webp")

		assert.ErrorIs(t, repo.DeleteMedia(ctx, created.ID, 999), storage.ErrMediaNotFound)
		assert.ErrorIs(t, repo.DeleteMedia(ctx, 999, m.ID), storage.ErrMediaNotFound)
		assert.Equal(t, []string{"a.webp"}, storedMediaKeys(t))
	})
}

func TestRepository_SetBoundary(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the boundary and leaves the rest alone", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, &subdistricts.SubDistrict{Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir")})
		m := mustAddMedia(t, repo, created.ID, "photo.webp")

		got, err := repo.SetBoundary(ctx, created.ID, mustBoundary(t, squareJSON))

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, new("Banjir"), got.FloodHistory)
		assert.Equal(t, []storage.Media{m}, got.Medias)
		assert.Equal(t, mustBoundary(t, squareJSON), got.Boundary)

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("replaces the previous boundary", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustSetBoundary(t, repo, created.ID)
		two := mustBoundary(t, `{"type":"MultiPolygon","coordinates":[`+
			`[[[106.8,-6.2],[106.85,-6.2],[106.85,-6.25],[106.8,-6.2]]],`+
			`[[[106.9,-6.2],[106.95,-6.2],[106.95,-6.25],[106.9,-6.2]]]]}`)

		got, err := repo.SetBoundary(ctx, created.ID, two)

		require.NoError(t, err)
		assert.Equal(t, two, got.Boundary)
	})

	t.Run("keeps a hole in a polygon", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		holed := mustBoundary(t, `{"type":"Polygon","coordinates":[`+
			`[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3],[106.8,-6.2]],`+
			`[[106.84,-6.24],[106.86,-6.24],[106.86,-6.26],[106.84,-6.24]]]}`)

		got, err := repo.SetBoundary(ctx, created.ID, holed)

		require.NoError(t, err)
		assert.Equal(t, holed, got.Boundary)
	})

	t.Run("returns ErrInvalidBoundary for an area that crosses itself and keeps the old one", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustSetBoundary(t, repo, created.ID)
		bowTie := mustBoundary(t, `{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.3],[106.9,-6.2],[106.8,-6.3],[106.8,-6.2]]]}`)

		got, err := repo.SetBoundary(ctx, created.ID, bowTie)

		assert.ErrorIs(t, err, subdistricts.ErrInvalidBoundary)
		assert.Nil(t, got)
		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, mustBoundary(t, squareJSON), stored.Boundary)
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.SetBoundary(ctx, 999, mustBoundary(t, squareJSON))

		assert.ErrorIs(t, err, subdistricts.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_GetMany_Medias(t *testing.T) {
	ctx := context.Background()
	repo := newRepository(t)
	first := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
	second := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
	mustCreate(t, repo, newSubDistrict("Tebet", "tebet"))
	mustAddMedia(t, repo, first.ID, "1/a.webp")
	mustAddMedia(t, repo, second.ID, "2/a.webp")
	mustAddMedia(t, repo, first.ID, "1/b.webp")
	mustSetBoundary(t, repo, second.ID)

	got, err := repo.GetMany(ctx)

	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, []string{"1/a.webp", "1/b.webp"}, mediaKeys(got[0].Medias))
	assert.Nil(t, got[0].Boundary)
	assert.Equal(t, []string{"2/a.webp"}, mediaKeys(got[1].Medias))
	assert.Equal(t, mustBoundary(t, squareJSON), got[1].Boundary)
	assert.NotNil(t, got[2].Medias, "a sub-district without medias has an empty list")
	assert.Empty(t, got[2].Medias)
}
