package domain

import "time"

// Bounds of a star rating, matching the CHECK on PLACE_REVIEWS.rating.
const (
	MinRating = 1
	MaxRating = 5
)

// MaxReviewCommentLength caps the characters in a review's comment.
const MaxReviewCommentLength = 1000

// Review represents a row in the PLACE_REVIEWS table, joined with its author.
type Review struct {
	ID      int64 `db:"id"`
	PlaceID int64 `db:"place_id"`
	UserID  int64 `db:"user_id"`
	// Rating is the number of stars given, from MinRating to MaxRating.
	Rating int `db:"rating"`
	// Comment is nil when the review is only a rating.
	Comment   *string   `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	User      Author    `db:"user"`
}
