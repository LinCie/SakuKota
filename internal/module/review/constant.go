package review

// Bounds of a star rating, matching the CHECK on PLACE_REVIEWS.rating.
const (
	minRating = 1
	maxRating = 5
)

// maxCommentLength caps the characters in a review's comment.
const maxCommentLength = 1000
