package reviews

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRating   = fmt.Errorf("rating must be between %d and %d", minRating, maxRating)
	ErrCommentTooLong  = fmt.Errorf("comment must be %d characters or fewer", maxCommentLength)
	ErrPlaceNotFound   = errors.New("place not found")
	ErrAlreadyReviewed = errors.New("place already reviewed by this user")
)
