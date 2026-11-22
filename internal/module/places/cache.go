package places

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"slices"

	"github.com/redis/go-redis/v9"
)

// cachedRepository serves GetMany from a Redis index of place ids, categories, and coordinates.
// Entries expire only after listCacheTTL, so a list can be that stale.
type cachedRepository struct {
	Repository
	client *redis.Client
}

// -----------------------------------------------------------------------------
// Exported
// -----------------------------------------------------------------------------

// NewCachedRepository wraps repo so GetMany reads from Redis when it can.
// A Redis failure never fails a request; the index is read from repo instead.
func NewCachedRepository(repo Repository, client *redis.Client) Repository {
	return &cachedRepository{
		Repository: repo,
		client:     client,
	}
}

// GetMany pages through the cached index of the filter's sub-district.
// A filter without a sub-district is not cached.
func (r *cachedRepository) GetMany(ctx context.Context, filter PlaceFilter) ([]Place, int, error) {
	if filter.SubDistrictID == 0 {
		return r.Repository.GetMany(ctx, filter)
	}

	index, err := r.index(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	ids, total := pageFromIndex(index, filter)
	if len(ids) == 0 {
		return []Place{}, total, nil
	}

	places, err := r.Repository.GetByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}

	return places, total, nil
}

// -----------------------------------------------------------------------------
// Unexported
// -----------------------------------------------------------------------------

// index returns the encoded index for the filter. On a miss, a bad entry, or a
// Redis failure it reads the index from the repository and tries to cache it.
func (r *cachedRepository) index(ctx context.Context, filter PlaceFilter) ([]byte, error) {
	key := indexCacheKey(filter)

	// A length that is not a whole number of entries means a corrupt entry.
	if data, err := r.client.Get(ctx, key).Bytes(); err == nil && len(data)%indexEntrySize == 0 {
		return data, nil
	}

	locations, err := r.Repository.GetLocations(ctx, filter)
	if err != nil {
		return nil, err
	}

	data := encodeIndex(locations)
	_ = r.client.Set(ctx, key, data, listCacheTTL).Err()

	return data, nil
}

// indexCacheKey returns the Redis key of the filter's index.
// Bounds, limit, and offset are left out because they are applied to the index afterwards.
// The categories are sorted so their order does not matter.
func indexCacheKey(filter PlaceFilter) string {
	data, _ := json.Marshal(PlaceFilter{
		SubDistrictID: filter.SubDistrictID,
		CategoryIDs:   slices.Sorted(slices.Values(filter.CategoryIDs)),
		Query:         filter.Query,
	})
	sum := sha256.Sum256(data)

	return listCacheKeyPrefix + hex.EncodeToString(sum[:])
}

// encodeIndex packs the locations into the compact form kept in Redis.
func encodeIndex(locations []PlaceLocation) []byte {
	data := make([]byte, 0, len(locations)*indexEntrySize)

	for _, l := range locations {
		longitude, latitude := math.NaN(), math.NaN()
		if l.Longitude != nil && l.Latitude != nil {
			longitude, latitude = *l.Longitude, *l.Latitude
		}

		data = binary.LittleEndian.AppendUint64(data, uint64(l.ID))
		data = binary.LittleEndian.AppendUint64(data, uint64(l.CategoryID))
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(longitude))
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(latitude))
	}

	return data
}

// pageFromIndex returns the ids on the filter's page of the index, and the number
// of places inside the filter's bounds (edges included). With bounds, the places
// are spread over the box (see spreadPlaces); otherwise the index order is kept.
func pageFromIndex(index []byte, filter PlaceFilter) ([]int64, int) {
	var matches []spreadPlace

	for entry := index; len(entry) >= indexEntrySize; entry = entry[indexEntrySize:] {
		m := spreadPlace{
			id:         int64(binary.LittleEndian.Uint64(entry)),
			categoryID: int64(binary.LittleEndian.Uint64(entry[8:])),
			longitude:  math.Float64frombits(binary.LittleEndian.Uint64(entry[16:])),
			latitude:   math.Float64frombits(binary.LittleEndian.Uint64(entry[24:])),
		}

		// Written as "not inside" so NaN (no location) is excluded: every NaN comparison is false.
		if b := filter.Bounds; b != nil &&
			!(m.longitude >= b.MinLng && m.longitude <= b.MaxLng && m.latitude >= b.MinLat && m.latitude <= b.MaxLat) {
			continue
		}

		matches = append(matches, m)
	}

	if filter.Bounds != nil {
		matches = spreadPlaces(matches, *filter.Bounds, filter.Limit)
	}

	total := len(matches)
	start := min(filter.Offset, total)
	end := min(start+filter.Limit, total)

	var ids []int64
	for _, m := range matches[start:end] {
		ids = append(ids, m.id)
	}

	return ids, total
}
