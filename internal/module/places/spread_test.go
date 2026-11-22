package places

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBoundingBox_gridCell(t *testing.T) {
	unit := BoundingBox{MinLng: 0, MinLat: 0, MaxLng: 1, MaxLat: 1}

	tests := []struct {
		name   string
		bounds BoundingBox
		limit  int
		want   float64
	}{
		{name: "a power of two that splits the box into limit cells", bounds: unit, limit: 4, want: 0.5},
		{name: "rounds up to fewer, bigger cells", bounds: unit, limit: 5, want: 0.5},
		{name: "more places make smaller cells", bounds: unit, limit: 16, want: 0.25},
		{name: "an empty box gets the smallest cell", bounds: BoundingBox{MinLng: 1, MinLat: 1, MaxLng: 1, MaxLat: 1}, limit: 4, want: 1.0 / (1 << 16)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.bounds.gridCell(tt.limit))
		})
	}
}

func TestSpreadPlaces(t *testing.T) {
	// With a limit of 4, the unit box is split into four cells of 0.5.
	unit := BoundingBox{MinLng: 0, MinLat: 0, MaxLng: 1, MaxLat: 1}

	ids := func(places []spreadPlace) []int64 {
		out := make([]int64, 0, len(places))
		for _, p := range places {
			out = append(out, p.id)
		}
		return out
	}

	t.Run("takes a place from every cell before a second one from any", func(t *testing.T) {
		got := spreadPlaces([]spreadPlace{
			{id: 1, categoryID: 1, longitude: 0.45, latitude: 0.45},
			{id: 2, categoryID: 1, longitude: 0.44, latitude: 0.45},
			{id: 3, categoryID: 1, longitude: 0.45, latitude: 0.44},
			{id: 4, categoryID: 1, longitude: 0.43, latitude: 0.43},
			{id: 5, categoryID: 1, longitude: 0.9, latitude: 0.9},
			{id: 6, categoryID: 1, longitude: 0.1, latitude: 0.8},
		}, unit, 4)

		// Round 1 is the place nearest each cell's centre, nearest the box's centre
		// first; the bottom-left cell's others follow, nearest its centre first and
		// ties (2 and 3) by id.
		assert.Equal(t, []int64{4, 6, 5, 2, 3, 1}, ids(got))
	})

	t.Run("takes a cell's categories in turns, starting at a different one in neighboring cells", func(t *testing.T) {
		got := spreadPlaces([]spreadPlace{
			{id: 1, categoryID: 10, longitude: 0.25, latitude: 0.25},
			{id: 2, categoryID: 20, longitude: 0.3, latitude: 0.3},
			{id: 3, categoryID: 10, longitude: 0.75, latitude: 0.25},
			{id: 4, categoryID: 20, longitude: 0.8, latitude: 0.3},
			{id: 5, categoryID: 10, longitude: 0.26, latitude: 0.26},
		}, unit, 4)

		// Cell (0,0) starts at category 10 and cell (1,0) at category 20, though 3
		// is at its cell's centre. Place 5 waits for category 20's turn in its cell.
		assert.Equal(t, []int64{1, 4, 2, 3, 5}, ids(got))
	})

	t.Run("the cells do not move with the box", func(t *testing.T) {
		panned := BoundingBox{MinLng: 0.1, MinLat: 0, MaxLng: 1.1, MaxLat: 1}
		got := spreadPlaces([]spreadPlace{
			{id: 1, categoryID: 10, longitude: 0.25, latitude: 0.25},
			{id: 2, categoryID: 20, longitude: 0.3, latitude: 0.3},
			{id: 3, categoryID: 10, longitude: 0.75, latitude: 0.25},
			{id: 4, categoryID: 20, longitude: 0.8, latitude: 0.3},
		}, panned, 4)

		// The same places start their cells; only the order by the new centre changes.
		assert.Equal(t, []int64{4, 1, 3, 2}, ids(got))
	})
}

func TestPageFromIndex(t *testing.T) {
	at := func(longitude, latitude float64) (*float64, *float64) { return &longitude, &latitude }
	location := func(id, categoryID int64, longitude, latitude *float64) PlaceLocation {
		return PlaceLocation{ID: id, CategoryID: categoryID, Longitude: longitude, Latitude: latitude}
	}
	lng1, lat1 := at(0.45, 0.45)
	lng2, lat2 := at(0.9, 0.9)
	lng3, lat3 := at(2, 2)
	index := encodeIndex([]PlaceLocation{
		location(1, 1, lng1, lat1),
		location(2, 1, lng2, lat2),
		location(3, 1, lng3, lat3),
		location(4, 1, nil, nil),
	})
	unit := &BoundingBox{MinLng: 0, MinLat: 0, MaxLng: 1, MaxLat: 1}

	tests := []struct {
		name      string
		filter    PlaceFilter
		want      []int64
		wantTotal int
	}{
		{name: "without bounds keeps the index order", filter: PlaceFilter{Limit: 10}, want: []int64{1, 2, 3, 4}, wantTotal: 4},
		{name: "bounds drop places outside and without a location", filter: PlaceFilter{Bounds: unit, Limit: 4}, want: []int64{1, 2}, wantTotal: 2},
		{name: "limit cuts the spread order", filter: PlaceFilter{Bounds: unit, Limit: 1}, want: []int64{1}, wantTotal: 2},
		{name: "offset skips into the spread order", filter: PlaceFilter{Bounds: unit, Limit: 4, Offset: 1}, want: []int64{2}, wantTotal: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, total := pageFromIndex(index, tt.filter)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}
