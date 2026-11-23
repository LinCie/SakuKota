package domain

// MaxMedias is the most medias a place, sub-district, or review can have. The
// position CHECKs of PLACE_MEDIAS, SUB_DISTRICT_MEDIAS, and
// PLACE_REVIEW_MEDIAS match it.
const MaxMedias = 10

// Media is a stored image or video: a MEDIAS row, and its position when it is
// linked to a place, sub-district, or review.
type Media struct {
	ID          int64  `db:"id"`
	Key         string `db:"key"`
	ContentType string `db:"content_type"`
	Size        int64  `db:"size_bytes"`
	Position    int    `db:"position"`
}
