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

	"sakukota-go/internal/domain"
	"sakukota-go/internal/module/subdistricts"
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

// newRepository returns a repository over an emptied database.
func newRepository(t *testing.T) *subdistricts.PostgresRepository {
	t.Helper()
	pgtest.Reset(t, testDB)
	return subdistricts.NewPostgresRepository(testDB)
}

// newSubDistrict returns a sub-district that is valid to create.
func newSubDistrict(name, slug string) *domain.SubDistrict {
	return &domain.SubDistrict{Name: name, Slug: slug}
}

func mustCreate(t *testing.T, repo *subdistricts.PostgresRepository, sd *domain.SubDistrict) *domain.SubDistrict {
	t.Helper()

	created, err := repo.Create(t.Context(), sd)
	require.NoError(t, err, "Create(%q)", sd.Name)

	return created
}

// mustFetch returns the stored sub-district with the id.
func mustFetch(t *testing.T, repo *subdistricts.PostgresRepository, id int64) *domain.SubDistrict {
	t.Helper()

	sd, err := repo.Fetch(t.Context(), id)
	require.NoError(t, err, "Fetch(%d)", id)

	return sd
}

func TestRepository_Create(t *testing.T) {
	t.Run("stores-and-returns", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Create(t.Context(), &domain.SubDistrict{
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir 2020"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(75000.5),
		})
		require.NoError(t, err, "Create()")

		assert.NotZero(t, got.ID, "Create().ID, want a generated ID")
		want := domain.SubDistrict{
			ID:                     got.ID,
			Name:                   "Menteng",
			Slug:                   "menteng",
			FloodHistory:           new("Banjir 2020"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(75000.5),
			Medias:                 []domain.Media{}, // a new sub-district has an empty, non-nil media list
		}
		assert.Equal(t, want, *got, "Create()")

		stored := mustFetch(t, repo, got.ID)
		assert.Equal(t, got, stored, "Fetch(%d) after Create", got.ID)
	})

	t.Run("optional-fields-empty", func(t *testing.T) {
		repo := newRepository(t)

		got := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		assert.Nil(t, got.FloodHistory, "Create().FloodHistory")
		assert.Nil(t, got.SafetyCrimeInformation, "Create().SafetyCrimeInformation")
		assert.Nil(t, got.EstimatedDailySpending, "Create().EstimatedDailySpending")
	})

	conflicts := []struct {
		name       string
		inName     string
		slug       string
		wantExists bool
	}{
		{"same-name", "Menteng", "other", true},
		{"same-name-other-case", "MENTENG", "other", true},
		{"same-slug", "Other", "menteng", true},
		{"different-name-and-slug", "Other", "other", false},
	}
	for _, c := range conflicts {
		t.Run("with-"+c.name, func(t *testing.T) {
			repo := newRepository(t)
			mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

			got, err := repo.Create(t.Context(), newSubDistrict(c.inName, c.slug))

			if !c.wantExists {
				assert.NoError(t, err, "Create(%q, %q)", c.inName, c.slug)
				return
			}
			assert.ErrorIs(t, err, domain.ErrSubDistrictExists, "Create(%q, %q)", c.inName, c.slug)
			assert.Nil(t, got, "Create(%q, %q)", c.inName, c.slug)
		})
	}
}

func TestRepository_List(t *testing.T) {
	t.Run("id-order", func(t *testing.T) {
		repo := newRepository(t)
		zebra := mustCreate(t, repo, newSubDistrict("Zebra", "zebra"))
		alpha := mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))

		got, err := repo.List(t.Context())
		require.NoError(t, err, "List()")

		assert.Equal(t, []domain.SubDistrict{*zebra, *alpha}, got, "List()")
	})

	t.Run("empty-non-nil", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.List(t.Context())
		require.NoError(t, err, "List()")

		assert.Equal(t, []domain.SubDistrict{}, got, "List(), want an empty, non-nil slice")
	})
}

func TestRepository_Fetch(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		repo := newRepository(t)
		mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.Fetch(t.Context(), created.ID)
		require.NoError(t, err, "Fetch(%d)", created.ID)

		assert.Equal(t, created, got, "Fetch(%d)", created.ID)
	})

	t.Run("unknown-id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Fetch(t.Context(), 999)

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "Fetch(999)")
		assert.Nil(t, got, "Fetch(999)")
	})
}

func TestRepository_Update(t *testing.T) {
	t.Run("stores-every-field", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Old", "old"))

		got, err := repo.Update(t.Context(), &domain.SubDistrict{
			ID:                     created.ID,
			Name:                   "New",
			Slug:                   "new",
			FloodHistory:           new("Banjir"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(50000.0),
		})
		require.NoError(t, err, "Update()")

		want := domain.SubDistrict{
			ID:                     created.ID,
			Name:                   "New",
			Slug:                   "new",
			FloodHistory:           new("Banjir"),
			SafetyCrimeInformation: new("Aman"),
			EstimatedDailySpending: new(50000.0),
			Medias:                 []domain.Media{},
		}
		assert.Equal(t, want, *got, "Update()")

		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, got, stored, "Fetch(%d) after Update", created.ID)
	})

	t.Run("clears-nil-fields", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, &domain.SubDistrict{
			Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir"), EstimatedDailySpending: new(1.0),
		})

		got, err := repo.Update(t.Context(), &domain.SubDistrict{ID: created.ID, Name: "Menteng", Slug: "menteng"})
		require.NoError(t, err, "Update()")

		assert.Nil(t, got.FloodHistory, "Update().FloodHistory")
		assert.Nil(t, got.EstimatedDailySpending, "Update().EstimatedDailySpending")
	})

	t.Run("keeps-medias-and-boundary", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "photo.jpg")
		mustSetBoundary(t, repo, created.ID)

		got, err := repo.Update(t.Context(), &domain.SubDistrict{ID: created.ID, Name: "Renamed", Slug: "renamed"})
		require.NoError(t, err, "Update()")

		assert.Equal(t, []domain.Media{m}, got.Medias, "Update().Medias")
		assert.Equal(t, mustBoundary(t, squareJSON), got.Boundary, "Update().Boundary")
	})

	t.Run("own-name-and-slug", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.Update(t.Context(), &domain.SubDistrict{ID: created.ID, Name: "MENTENG", Slug: "menteng"})
		require.NoError(t, err, "Update()")

		assert.Equal(t, "MENTENG", got.Name, "Update().Name")
	})

	t.Run("taken-name-or-slug", func(t *testing.T) {
		for _, c := range []struct{ name, slug string }{
			{"alpha", "new"},
			{"New", "alpha"},
		} {
			t.Run(c.name+"-"+c.slug, func(t *testing.T) {
				repo := newRepository(t)
				mustCreate(t, repo, newSubDistrict("Alpha", "alpha"))
				created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

				got, err := repo.Update(t.Context(), &domain.SubDistrict{ID: created.ID, Name: c.name, Slug: c.slug})

				assert.ErrorIs(t, err, domain.ErrSubDistrictExists, "Update(%q, %q)", c.name, c.slug)
				assert.Nil(t, got, "Update(%q, %q)", c.name, c.slug)
				stored := mustFetch(t, repo, created.ID)
				assert.Equal(t, created, stored, "Fetch(%d) after failed Update", created.ID)
			})
		}
	})

	t.Run("unknown-id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.Update(t.Context(), &domain.SubDistrict{ID: 999, Name: "Menteng", Slug: "menteng"})

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "Update(999)")
		assert.Nil(t, got, "Update(999)")
	})
}

func TestRepository_Delete(t *testing.T) {
	t.Run("leaves-the-others", func(t *testing.T) {
		repo := newRepository(t)
		doomed := mustCreate(t, repo, newSubDistrict("Doomed", "doomed"))
		kept := mustCreate(t, repo, newSubDistrict("Kept", "kept"))

		require.NoError(t, repo.Delete(t.Context(), doomed.ID), "Delete(%d)", doomed.ID)

		got, err := repo.List(t.Context())
		require.NoError(t, err, "List()")
		assert.Equal(t, []domain.SubDistrict{*kept}, got, "List() after Delete")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys, want none: no images, nothing queued")
	})

	t.Run("queues-images", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		kept := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
		mustAddMedia(t, repo, created.ID, "a.jpg")
		mustAddMedia(t, repo, created.ID, "b.jpg")
		mustSetBoundary(t, repo, created.ID)
		mustAddMedia(t, repo, kept.ID, "kept.jpg")

		require.NoError(t, repo.Delete(t.Context(), created.ID), "Delete(%d)", created.ID)

		assert.Equal(t, []string{"a.jpg", "b.jpg"}, pgtest.QueuedKeys(t, testDB), "queued keys")
		assert.Equal(t, []string{"kept.jpg"}, storedMediaKeys(t), "stored media keys, only the other sub-district's images are left")
	})

	t.Run("already-queued-images", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "photo.png")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('photo.png')`)
		require.NoError(t, err, "queue photo.png")

		require.NoError(t, repo.Delete(t.Context(), created.ID), "Delete(%d)", created.ID)

		assert.Equal(t, []string{"photo.png"}, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("in-use", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "photo.jpg")
		mustSetBoundary(t, repo, created.ID)
		pgtest.InsertPlace(t, testDB, created.ID, pgtest.CategoryID(t, testDB, "laundry"), "Laundry")

		err := repo.Delete(t.Context(), created.ID)

		assert.ErrorIs(t, err, domain.ErrSubDistrictInUse, "Delete(%d)", created.ID)
		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, []domain.Media{m}, stored.Medias, "Medias after failed Delete")
		assert.Equal(t, mustBoundary(t, squareJSON), stored.Boundary, "Boundary after failed Delete")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys, want none: the images are still in use")
	})

	t.Run("unknown-id", func(t *testing.T) {
		repo := newRepository(t)

		assert.ErrorIs(t, repo.Delete(t.Context(), 999), domain.ErrSubDistrictNotFound, "Delete(999)")
	})

	t.Run("second-time", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		require.NoError(t, repo.Delete(t.Context(), created.ID), "Delete(%d)", created.ID)

		assert.ErrorIs(t, repo.Delete(t.Context(), created.ID), domain.ErrSubDistrictNotFound, "second Delete(%d)", created.ID)
	})
}

// newMedia returns a promoted image to store under key.
func newMedia(key string) domain.Media {
	return domain.Media{Key: key, ContentType: "image/webp", Size: 2048}
}

// mustAddMedia stores an image under key for the sub-district and returns it.
func mustAddMedia(t *testing.T, repo *subdistricts.PostgresRepository, id int64, key string) domain.Media {
	t.Helper()

	got, err := repo.AddMedia(t.Context(), id, newMedia(key))
	require.NoError(t, err, "AddMedia(%d, %q)", id, key)

	for _, m := range got.Medias {
		if m.Key == key {
			return m
		}
	}
	require.Fail(t, "media not returned", "AddMedia(%d, %q) did not return the added media", id, key)
	return domain.Media{}
}

// mustSetBoundary stores squareJSON as the sub-district's boundary.
func mustSetBoundary(t *testing.T, repo *subdistricts.PostgresRepository, id int64) {
	t.Helper()

	_, err := repo.SetBoundary(t.Context(), id, mustBoundary(t, squareJSON))
	require.NoError(t, err, "SetBoundary(%d)", id)
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

// storedMediaKeys returns the keys of every MEDIAS row, sorted.
func storedMediaKeys(t *testing.T) []string {
	t.Helper()

	keys := make([]string, 0)
	require.NoError(t, testDB.Select(&keys, `SELECT key FROM MEDIAS ORDER BY key`), "select media keys")

	return keys
}

func TestRepository_AddMedia(t *testing.T) {
	t.Run("first-position", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))

		got, err := repo.AddMedia(t.Context(), created.ID, newMedia("subdistricts/1/medias/a.webp"))
		require.NoError(t, err, "AddMedia()")

		assert.Equal(t, created.ID, got.ID, "AddMedia().ID")
		require.Len(t, got.Medias, 1, "AddMedia().Medias = %+v, want 1 media", got.Medias)
		m := got.Medias[0]
		assert.NotZero(t, m.ID, "AddMedia().Medias[0].ID, want a generated ID")
		want := domain.Media{ID: m.ID, Key: "subdistricts/1/medias/a.webp", ContentType: "image/webp", Size: 2048, Position: 1}
		assert.Equal(t, want, m, "AddMedia().Medias[0]")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys, want none")

		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, got, stored, "Fetch(%d) after AddMedia", created.ID)
	})

	t.Run("appends-after-delete", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "a.webp")
		removed := mustAddMedia(t, repo, created.ID, "b.webp")
		mustAddMedia(t, repo, created.ID, "c.webp")
		require.NoError(t, repo.DeleteMedia(t.Context(), created.ID, removed.ID), "DeleteMedia()")

		got, err := repo.AddMedia(t.Context(), created.ID, newMedia("d.webp"))
		require.NoError(t, err, "AddMedia()")

		assert.Equal(t, []string{"a.webp", "c.webp", "d.webp"}, mediaKeys(got.Medias), "AddMedia() media keys")
		assert.Equal(t, []int{1, 2, 3}, mediaPositions(got.Medias), "AddMedia() media positions")
	})

	t.Run("limit-reached", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		for i := range domain.MaxMedias {
			mustAddMedia(t, repo, created.ID, fmt.Sprintf("%d.webp", i))
		}

		got, err := repo.AddMedia(t.Context(), created.ID, newMedia("extra.webp"))

		assert.ErrorIs(t, err, domain.ErrMediaLimitReached, "AddMedia()")
		assert.Nil(t, got, "AddMedia()")
		assert.Len(t, storedMediaKeys(t), domain.MaxMedias, "stored medias: the extra media must not be stored")
	})

	t.Run("concurrent-adds", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		const attempts = domain.MaxMedias + 4

		errs := make([]error, attempts)
		var wg sync.WaitGroup
		for i := range attempts {
			wg.Go(func() {
				_, errs[i] = repo.AddMedia(t.Context(), created.ID, newMedia(fmt.Sprintf("%d.webp", i)))
			})
		}
		wg.Wait()

		added, limited := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				added++
			case errors.Is(err, domain.ErrMediaLimitReached):
				limited++
			default:
				assert.NoError(t, err, "unexpected error")
			}
		}
		assert.Equal(t, domain.MaxMedias, added, "added medias")
		assert.Equal(t, attempts-domain.MaxMedias, limited, "limited adds")

		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, mediaPositions(stored.Medias), "media positions")
	})

	t.Run("unknown-id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.AddMedia(t.Context(), 999, newMedia("a.webp"))

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "AddMedia(999)")
		assert.Nil(t, got, "AddMedia(999)")
		assert.Empty(t, storedMediaKeys(t), "stored media keys, want none")
	})
}

func TestRepository_DeleteMedia(t *testing.T) {
	t.Run("queues-image", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustAddMedia(t, repo, created.ID, "a.webp")
		removed := mustAddMedia(t, repo, created.ID, "b.webp")
		mustAddMedia(t, repo, created.ID, "c.webp")

		require.NoError(t, repo.DeleteMedia(t.Context(), created.ID, removed.ID), "DeleteMedia()")

		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, []string{"a.webp", "c.webp"}, mediaKeys(stored.Medias), "media keys")
		assert.Equal(t, []int{1, 2}, mediaPositions(stored.Medias), "media positions, the later medias shift down")
		assert.Equal(t, []string{"a.webp", "c.webp"}, storedMediaKeys(t), "stored media keys")
		assert.Equal(t, []string{"b.webp"}, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("already-queued-image", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		removed := mustAddMedia(t, repo, created.ID, "a.webp")
		_, err := testDB.Exec(`INSERT INTO STORAGE_DELETIONS (key) VALUES ('a.webp')`)
		require.NoError(t, err, "queue a.webp")

		require.NoError(t, repo.DeleteMedia(t.Context(), created.ID, removed.ID), "DeleteMedia()")

		assert.Equal(t, []string{"a.webp"}, pgtest.QueuedKeys(t, testDB), "queued keys")
	})

	t.Run("media-of-another", func(t *testing.T) {
		repo := newRepository(t)
		owner := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		other := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
		m := mustAddMedia(t, repo, owner.ID, "a.webp")

		err := repo.DeleteMedia(t.Context(), other.ID, m.ID)

		assert.ErrorIs(t, err, domain.ErrMediaNotFound, "DeleteMedia(%d, %d)", other.ID, m.ID)
		assert.Equal(t, []string{"a.webp"}, storedMediaKeys(t), "stored media keys, the media must be kept")
		assert.Empty(t, pgtest.QueuedKeys(t, testDB), "queued keys, want none")
	})

	t.Run("unknown-media-or-sub-district", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		m := mustAddMedia(t, repo, created.ID, "a.webp")

		assert.ErrorIs(t, repo.DeleteMedia(t.Context(), created.ID, 999), domain.ErrMediaNotFound, "DeleteMedia(%d, 999)", created.ID)
		assert.ErrorIs(t, repo.DeleteMedia(t.Context(), 999, m.ID), domain.ErrMediaNotFound, "DeleteMedia(999, %d)", m.ID)
		assert.Equal(t, []string{"a.webp"}, storedMediaKeys(t), "stored media keys")
	})
}

func TestRepository_SetBoundary(t *testing.T) {
	t.Run("leaves-the-rest-alone", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, &domain.SubDistrict{Name: "Menteng", Slug: "menteng", FloodHistory: new("Banjir")})
		m := mustAddMedia(t, repo, created.ID, "photo.webp")

		got, err := repo.SetBoundary(t.Context(), created.ID, mustBoundary(t, squareJSON))
		require.NoError(t, err, "SetBoundary()")

		assert.Equal(t, created.ID, got.ID, "SetBoundary().ID")
		assert.Equal(t, new("Banjir"), got.FloodHistory, "SetBoundary().FloodHistory")
		assert.Equal(t, []domain.Media{m}, got.Medias, "SetBoundary().Medias")
		assert.Equal(t, mustBoundary(t, squareJSON), got.Boundary, "SetBoundary().Boundary")

		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, got, stored, "Fetch(%d) after SetBoundary", created.ID)
	})

	t.Run("replaces-previous", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustSetBoundary(t, repo, created.ID)
		two := mustBoundary(t, `{"type":"MultiPolygon","coordinates":[`+
			`[[[106.8,-6.2],[106.85,-6.2],[106.85,-6.25],[106.8,-6.2]]],`+
			`[[[106.9,-6.2],[106.95,-6.2],[106.95,-6.25],[106.9,-6.2]]]]}`)

		got, err := repo.SetBoundary(t.Context(), created.ID, two)
		require.NoError(t, err, "SetBoundary()")

		assert.Equal(t, two, got.Boundary, "SetBoundary().Boundary")
	})

	t.Run("keeps-hole", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		holed := mustBoundary(t, `{"type":"Polygon","coordinates":[`+
			`[[106.8,-6.2],[106.9,-6.2],[106.9,-6.3],[106.8,-6.3],[106.8,-6.2]],`+
			`[[106.84,-6.24],[106.86,-6.24],[106.86,-6.26],[106.84,-6.24]]]}`)

		got, err := repo.SetBoundary(t.Context(), created.ID, holed)
		require.NoError(t, err, "SetBoundary()")

		assert.Equal(t, holed, got.Boundary, "SetBoundary().Boundary")
	})

	t.Run("crosses-itself", func(t *testing.T) {
		repo := newRepository(t)
		created := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
		mustSetBoundary(t, repo, created.ID)
		bowTie := mustBoundary(t, `{"type":"Polygon","coordinates":[[[106.8,-6.2],[106.9,-6.3],[106.9,-6.2],[106.8,-6.3],[106.8,-6.2]]]}`)

		got, err := repo.SetBoundary(t.Context(), created.ID, bowTie)

		assert.ErrorIs(t, err, domain.ErrSubDistrictBoundaryInvalid, "SetBoundary()")
		assert.Nil(t, got, "SetBoundary()")
		stored := mustFetch(t, repo, created.ID)
		assert.Equal(t, mustBoundary(t, squareJSON), stored.Boundary, "Boundary after failed SetBoundary, the old one is kept")
	})

	t.Run("unknown-id", func(t *testing.T) {
		repo := newRepository(t)

		got, err := repo.SetBoundary(t.Context(), 999, mustBoundary(t, squareJSON))

		assert.ErrorIs(t, err, domain.ErrSubDistrictNotFound, "SetBoundary(999)")
		assert.Nil(t, got, "SetBoundary(999)")
	})
}

func TestRepository_List_Medias(t *testing.T) {
	repo := newRepository(t)
	first := mustCreate(t, repo, newSubDistrict("Menteng", "menteng"))
	second := mustCreate(t, repo, newSubDistrict("Kemang", "kemang"))
	mustCreate(t, repo, newSubDistrict("Tebet", "tebet"))
	mustAddMedia(t, repo, first.ID, "1/a.webp")
	mustAddMedia(t, repo, second.ID, "2/a.webp")
	mustAddMedia(t, repo, first.ID, "1/b.webp")
	mustSetBoundary(t, repo, second.ID)

	got, err := repo.List(t.Context())
	require.NoError(t, err, "List()")
	require.Len(t, got, 3, "List() sub-districts")

	assert.Equal(t, []string{"1/a.webp", "1/b.webp"}, mediaKeys(got[0].Medias), "List()[0] media keys")
	assert.Nil(t, got[0].Boundary, "List()[0].Boundary")
	assert.Equal(t, []string{"2/a.webp"}, mediaKeys(got[1].Medias), "List()[1] media keys")
	assert.Equal(t, mustBoundary(t, squareJSON), got[1].Boundary, "List()[1].Boundary")
	assert.Equal(t, []domain.Media{}, got[2].Medias, "List()[2].Medias, want an empty, non-nil list")
}
