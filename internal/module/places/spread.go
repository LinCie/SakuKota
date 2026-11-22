package places

import (
	"cmp"
	"math"
	"slices"
)

// spreadPlace is a place inside the bounds of a page, as spreadPlaces orders it.
type spreadPlace struct {
	id         int64
	categoryID int64
	longitude  float64
	latitude   float64
}

// spreadPlaces orders places inside bounds so that any first n of them are spread
// over the box and over the categories, instead of bunching up where most places
// are. PostgresRepository.List orders them the same way in SQL.
//
//  1. The box is split into square cells of bounds.gridCell(limit), counted from
//     longitude and latitude 0 so the grid does not move with the box.
//  2. Each cell takes its categories in turns, starting at a different one in
//     neighboring cells (category index minus cell x minus cell y, modulo the
//     number of categories in the cell, counting categories by ascending id).
//     Within a category, the places nearest the centre of the cell come first.
//  3. Round 1 is the first place of every cell, round 2 the second, and so on.
//     Within a round, the places nearest the centre of the box come first.
//
// Remaining ties are ordered by id.
func spreadPlaces(places []spreadPlace, bounds BoundingBox, limit int) []spreadPlace {
	cell := bounds.gridCell(limit)
	centerLng, centerLat := bounds.Center()

	type key struct{ x, y int64 }
	type ranked struct {
		spreadPlace
		cellDistance   float64
		centerDistance float64
		categoryIndex  int64
		categoryRound  int
		round          int
	}

	cells := make(map[key][]*ranked)
	all := make([]*ranked, 0, len(places))
	for _, p := range places {
		x := int64(math.Floor(p.longitude / cell))
		y := int64(math.Floor(p.latitude / cell))
		r := &ranked{
			spreadPlace:    p,
			cellDistance:   math.Hypot(p.longitude-(float64(x)+0.5)*cell, p.latitude-(float64(y)+0.5)*cell),
			centerDistance: math.Hypot(p.longitude-centerLng, p.latitude-centerLat),
		}
		cells[key{x, y}] = append(cells[key{x, y}], r)
		all = append(all, r)
	}

	for k, members := range cells {
		// Nearest the cell's centre first within each category.
		slices.SortFunc(members, func(a, b *ranked) int {
			return cmp.Or(
				cmp.Compare(a.categoryID, b.categoryID),
				cmp.Compare(a.cellDistance, b.cellDistance),
				cmp.Compare(a.id, b.id),
			)
		})

		var categoryCount int64
		for i, r := range members {
			if i == 0 || r.categoryID != members[i-1].categoryID {
				categoryCount++
				r.categoryRound = 1
			} else {
				r.categoryRound = members[i-1].categoryRound + 1
			}
			r.categoryIndex = categoryCount - 1
		}

		turn := func(r *ranked) int64 {
			return ((r.categoryIndex-k.x-k.y)%categoryCount + categoryCount) % categoryCount
		}
		slices.SortFunc(members, func(a, b *ranked) int {
			return cmp.Or(
				cmp.Compare(a.categoryRound, b.categoryRound),
				cmp.Compare(turn(a), turn(b)),
				cmp.Compare(a.cellDistance, b.cellDistance),
				cmp.Compare(a.id, b.id),
			)
		})
		for i, r := range members {
			r.round = i + 1
		}
	}

	slices.SortFunc(all, func(a, b *ranked) int {
		return cmp.Or(
			cmp.Compare(a.round, b.round),
			cmp.Compare(a.centerDistance, b.centerDistance),
			cmp.Compare(a.id, b.id),
		)
	})

	ordered := make([]spreadPlace, len(all))
	for i, r := range all {
		ordered[i] = r.spreadPlace
	}

	return ordered
}
