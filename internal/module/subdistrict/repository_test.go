//go:build integration

package subdistrict_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"rantau-go/internal/module/subdistrict"
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
func newRepository(t *testing.T) subdistrict.Repository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return subdistrict.NewPostgresRepository(testDB)
}

// newSubDistrict returns a sub-district that is valid to create.
func newSubDistrict(name, slug string) *subdistrict.SubDistrict {
	return &subdistrict.SubDistrict{Name: name, Slug: slug}
}

func mustCreate(t *testing.T, repo subdistrict.Repository, sd *subdistrict.SubDistrict) *subdistrict.SubDistrict {
	t.Helper()

	created, err := repo.Create(context.Background(), sd)
	require.NoError(t, err)

	return created
}

func TestRepository_Create(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the sub-district and returns it", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(ctx, &subdistrict.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           ptr("Banjir 2020"),
			SafetyCrimeInformation: ptr("Aman"),
			EstimatedDailySpending: ptr(75000.5),
		})

		require.NoError(t, err)
		assert.NotZero(t, got.ID)
		assert.Equal(t, "Menteng", got.Name)
		assert.Equal(t, "menteng", got.Slug)
		assert.Equal(t, ptr("Banjir 2020"), got.FloodHistory)
		assert.Equal(t, ptr("Aman"), got.SafetyCrimeInformation)
		assert.Equal(t, ptr(75000.5), got.EstimatedDailySpending)
		assert.Nil(t, got.ImageKey)
		assert.Nil(t, got.StaticMapImageKey)

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
				assert.ErrorIs(t, err, subdistrict.ErrSubDistrictExists)
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
		assert.Equal(t, []subdistrict.SubDistrict{*zebra, *alpha}, got)
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

		assert.ErrorIs(t, err, subdistrict.ErrSubDistrictNotFound)
		assert.Nil(t, got)
	})
}

func TestRepository_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("stores every field and returns the sub-district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Old", "old"))

		got, err := repo.Update(ctx, &subdistrict.SubDistrict{
			ID:                     created.ID,
			Name:                   "New",
			Slug:                   "new",
			FloodHistory:           ptr("Banjir"),
			SafetyCrimeInformation: ptr("Aman"),
			EstimatedDailySpending: ptr(50000.0),
		})

		require.NoError(t, err)
		assert.Equal(t, created.ID, got.ID)
		assert.Equal(t, "New", got.Name)
		assert.Equal(t, "new", got.Slug)
		assert.Equal(t, ptr("Banjir"), got.FloodHistory)
		assert.Equal(t, ptr("Aman"), got.SafetyCrimeInformation)
		assert.Equal(t, ptr(50000.0), got.EstimatedDailySpending)

		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, got, stored)
	})

	t.Run("clears optional fields that are nil", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, &subdistrict.SubDistrict{
			Name: "Menteng", Slug: "menteng", FloodHistory: ptr("Banjir"), EstimatedDailySpending: ptr(1.0),
		})

		got, err := repo.Update(ctx, &subdistrict.SubDistrict{ID: created.ID, Name: "Menteng", Slug: "menteng"})

		require.NoError(t, err)
		assert.Nil(t, got.FloodHistory)
		assert.Nil(t, got.EstimatedDailySpending)
	})

	t.Run("keeps the images", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")
		require.NoError(t, err)
		_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotStaticMap, "map.png")
		require.NoError(t, err)

		got, err := repo.Update(ctx, &subdistrict.SubDistrict{ID: created.ID, Name: "Renamed", Slug: "renamed"})

		require.NoError(t, err)
		assert.Equal(t, ptr("photo.jpg"), got.ImageKey)
		assert.Equal(t, ptr("map.png"), got.StaticMapImageKey)
	})

	t.Run("accepts the sub-district's own name and slug, even in another case", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.Update(ctx, &subdistrict.SubDistrict{ID: created.ID, Name: "MENTENG", Slug: "menteng"})

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

				got, err := repo.Update(ctx, &subdistrict.SubDistrict{ID: created.ID, Name: c.name, Slug: c.slug})

				assert.ErrorIs(t, err, subdistrict.ErrSubDistrictExists)
				assert.Nil(t, got)
				stored, err := repo.GetSubDistrict(ctx, created.ID)
				require.NoError(t, err)
				assert.Equal(t, created, stored)
			})
		}
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Update(ctx, &subdistrict.SubDistrict{ID: 999, Name: "Menteng", Slug: "menteng"})

		assert.ErrorIs(t, err, subdistrict.ErrSubDistrictNotFound)
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
		assert.Equal(t, []subdistrict.SubDistrict{*kept}, got)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "no images, nothing queued")
	})

	t.Run("queues both images for deletion", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")
		require.NoError(t, err)
		_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotStaticMap, "map.png")
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"map.png", "photo.jpg"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("queues only the image it has", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotStaticMap, "map.png")
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"map.png"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("tolerates images that are already queued or share a key", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "same.png")
		require.NoError(t, err)
		_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotStaticMap, "same.png")
		require.NoError(t, err)
		_, err = testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('same.png')`)
		require.NoError(t, err)

		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.Equal(t, []string{"same.png"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("returns ErrSubDistrictInUse while places reference it, and keeps everything", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")
		require.NoError(t, err)
		pgtest.InsertPlace(t, testDB, created.ID, pgtest.CategoryID(t, testDB, "laundry"), "Laundry")

		err = repo.Delete(ctx, created.ID)

		assert.ErrorIs(t, err, subdistrict.ErrSubDistrictInUse)
		_, err = repo.GetSubDistrict(ctx, created.ID)
		assert.NoError(t, err)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "the image is still in use")
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(ctx, 999), subdistrict.ErrSubDistrictNotFound)
	})

	t.Run("returns ErrSubDistrictNotFound the second time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		require.NoError(t, repo.Delete(ctx, created.ID))

		assert.ErrorIs(t, repo.Delete(ctx, created.ID), subdistrict.ErrSubDistrictNotFound)
	})
}

func TestRepository_SetImageKey(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the key in the slot and leaves the other slot alone", func(t *testing.T) {
		tests := []struct {
			slot       subdistrict.ImageSlot
			wantImage  *string
			wantMap    *string
			wantQueued string
		}{
			{subdistrict.SlotImage, ptr("new.jpg"), ptr("map.png"), "photo.jpg"},
			{subdistrict.SlotStaticMap, ptr("photo.jpg"), ptr("new.jpg"), "map.png"},
		}
		for _, tt := range tests {
			t.Run(string(tt.slot), func(t *testing.T) {
				repo := newRepository(t)
				created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
				_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")
				require.NoError(t, err)
				_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotStaticMap, "map.png")
				require.NoError(t, err)

				got, err := repo.SetImageKey(ctx, created.ID, tt.slot, "new.jpg")

				require.NoError(t, err)
				assert.Equal(t, created.ID, got.ID)
				assert.Equal(t, tt.wantImage, got.ImageKey)
				assert.Equal(t, tt.wantMap, got.StaticMapImageKey)
				assert.Equal(t, []string{tt.wantQueued}, pgtest.QueuedKeys(t, testDB), "only the replaced image is queued")

				stored, err := repo.GetSubDistrict(ctx, created.ID)
				require.NoError(t, err)
				assert.Equal(t, got, stored)
			})
		}
	})

	t.Run("queues nothing for a first image", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")

		require.NoError(t, err)
		assert.Equal(t, ptr("photo.jpg"), got.ImageKey)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("does not queue the key it keeps", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")
		require.NoError(t, err)

		_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "photo.jpg")

		require.NoError(t, err)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("tolerates a replaced image that is already queued", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		_, err := repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "old.jpg")
		require.NoError(t, err)
		_, err = testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('old.jpg')`)
		require.NoError(t, err)

		_, err = repo.SetImageKey(ctx, created.ID, subdistrict.SlotImage, "new.jpg")

		require.NoError(t, err)
		assert.Equal(t, []string{"old.jpg"}, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("returns ErrSubDistrictNotFound for an unknown id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.SetImageKey(ctx, 999, subdistrict.SlotImage, "photo.jpg")

		assert.ErrorIs(t, err, subdistrict.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		assert.Empty(t, pgtest.QueuedKeys(t, testDB))
	})

	t.Run("rejects an unknown slot and changes nothing", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.SetImageKey(ctx, created.ID, "banner", "banner.jpg")

		assert.Error(t, err)
		assert.NotErrorIs(t, err, subdistrict.ErrSubDistrictNotFound)
		assert.Nil(t, got)
		stored, err := repo.GetSubDistrict(ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created, stored)
	})
}
