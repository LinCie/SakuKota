package review

import "errors"

var (
	ErrInvalidRating   = errors.New("rating must be between 1 and 5")
	ErrCommentTooLong  = errors.New("comment must be 1000 characters or fewer")
	ErrPlaceNotFound   = errors.New("place not found")
	ErrAlreadyReviewed = errors.New("place already reviewed by this user")
)
