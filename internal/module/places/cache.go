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

// cachedRepository wraps a Repository and serves GetMany pages from a Redis index of ids and
// coordinates; entries only expire after listCacheTTL.
type cachedRepository struct {
	Repository
	client *redis.Client
}

// NewCachedRepository wraps repo so that GetMany answers from Redis when it can.
// Redis errors never fail a request: the index is then read from repo.
func NewCachedRepository(repo Repository, client *redis.Client) Repository {
	return &cachedRepository{
		Repository: repo,
		client:     client,
	}
}

// GetMany returns one page of places matching the filter, plus the total number of matches.
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

// index returns the encoded index for the filter, reading it from the repository and
// caching it on a miss or a Redis failure.
func (r *cachedRepository) index(ctx context.Context, filter PlaceFilter) ([]byte, error) {
	key := indexCacheKey(filter)

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

// indexCacheKey returns the Redis key for the filter's index, which depends only on the
// sub-district, the categories (in any order), and the query.
func indexCacheKey(filter PlaceFilter) string {
	data, _ := json.Marshal(PlaceFilter{
		SubDistrictID: filter.SubDistrictID,
		CategoryIDs:   slices.Sorted(slices.Values(filter.CategoryIDs)),
		Query:         filter.Query,
	})
	sum := sha256.Sum256(data)

	return listCacheKeyPrefix + hex.EncodeToString(sum[:])
}

// indexEntrySize is the bytes one place takes in an index: id, longitude, and latitude,
// 8 little-endian bytes each. A place without a location has NaN coordinates.
const indexEntrySize = 24

// encodeIndex packs the locations into the compact form kept in Redis.
func encodeIndex(locations []PlaceLocation) []byte {
	data := make([]byte, 0, len(locations)*indexEntrySize)

	for _, l := range locations {
		longitude, latitude := math.NaN(), math.NaN()
		if l.Longitude != nil && l.Latitude != nil {
			longitude, latitude = *l.Longitude, *l.Latitude
		}

		data = binary.LittleEndian.AppendUint64(data, uint64(l.ID))
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(longitude))
		data = binary.LittleEndian.AppendUint64(data, math.Float64bits(latitude))
	}

	return data
}

// pageFromIndex returns the ids on the filter's page of the index, plus the number of
// places inside the filter's bounds (edges included). A limit of zero means no limit.
func pageFromIndex(index []byte, filter PlaceFilter) ([]int64, int) {
	var ids []int64
	total := 0

	for entry := index; len(entry) >= indexEntrySize; entry = entry[indexEntrySize:] {
		if b := filter.Bounds; b != nil {
			longitude := math.Float64frombits(binary.LittleEndian.Uint64(entry[8:]))
			latitude := math.Float64frombits(binary.LittleEndian.Uint64(entry[16:]))

			if !(longitude >= b.MinLng && longitude <= b.MaxLng && latitude >= b.MinLat && latitude <= b.MaxLat) {
				continue
			}
		}

		if total >= filter.Offset && (filter.Limit <= 0 || len(ids) < filter.Limit) {
			ids = append(ids, int64(binary.LittleEndian.Uint64(entry)))
		}
		total++
	}

	return ids, total
}
