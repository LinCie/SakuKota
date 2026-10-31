package review

import "errors"

const (
	CodeInvalidPlaceID  = "INVALID_PLACE_ID"
	CodePlaceNotFound   = "PLACE_NOT_FOUND"
	CodeAlreadyReviewed = "ALREADY_REVIEWED"
)

var (
	ErrInvalidRating   = errors.New("rating must be between 1 and 5")
	ErrCommentTooLong  = errors.New("comment must be 1000 characters or fewer")
	ErrPlaceNotFound   = errors.New("place not found")
	ErrAlreadyReviewed = errors.New("place already reviewed by this user")
)
