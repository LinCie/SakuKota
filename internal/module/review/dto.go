package review

import "time"

// CreateReviewRequest is the API payload for reviewing a place. The place
// comes from the path and the author from the access token.
type CreateReviewRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment"`
}

// ReviewResponse is the API representation of a Review.
type ReviewResponse struct {
	ID        int64          `json:"id"`
	PlaceID   int64          `json:"place_id"`
	Rating    int            `json:"rating"`
	Comment   *string        `json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
	User      AuthorResponse `json:"user"`
}

// AuthorResponse is the API representation of a review's Author.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

// toReview builds the review to save from the place, the author, and the request body.
func toReview(placeID, userID int64, req CreateReviewRequest) *Review {
	return &Review{
		PlaceID: placeID,
		UserID:  userID,
		Rating:  req.Rating,
		Comment: req.Comment,
	}
}

// toReviewResponse turns r into the JSON shape sent to clients.
func toReviewResponse(r Review) ReviewResponse {
	return ReviewResponse{
		ID:        r.ID,
		PlaceID:   r.PlaceID,
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: r.CreatedAt,
		User: AuthorResponse{
			ID:   r.User.ID,
			Name: r.User.Name,
		},
	}
}

// toReviewResponses turns each review into its response shape.
func toReviewResponses(reviews []Review) []ReviewResponse {
	responses := make([]ReviewResponse, 0, len(reviews))

	for _, r := range reviews {
		responses = append(responses, toReviewResponse(r))
	}

	return responses
}
