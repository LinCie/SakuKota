package storage

import (
	"errors"
	"fmt"
)

// MaxMedias is the most medias a place or sub-district can have. The
// position CHECKs of PLACE_MEDIAS and SUB_DISTRICT_MEDIAS match it.
const MaxMedias = 10

var (
	ErrMediaNotFound     = errors.New("media not found")
	ErrMediaLimitReached = fmt.Errorf("at most %d medias are allowed", MaxMedias)
)

// Media is a stored image or video: a MEDIAS row, and its position when it is
// linked to a place or sub-district.
type Media struct {
	ID          int64  `db:"id"`
	Key         string `db:"key"`
	ContentType string `db:"content_type"`
	Size        int64  `db:"size_bytes"`
	Position    int    `db:"position"`
}
