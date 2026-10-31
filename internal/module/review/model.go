package review

import "time"

// Review represents a row in the PLACE_REVIEWS table, joined with its author.
type Review struct {
	ID      int64 `db:"id"`
	PlaceID int64 `db:"place_id"`
	UserID  int64 `db:"user_id"`
	// Rating is the number of stars given, from 1 to 5.
	Rating int `db:"rating"`
	// Comment is nil when the review is only a rating.
	Comment   *string   `db:"comment"`
	CreatedAt time.Time `db:"created_at"`
	User      Author    `db:"user"`
}

// Author is the user who wrote a Review, as shown next to it.
type Author struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}
