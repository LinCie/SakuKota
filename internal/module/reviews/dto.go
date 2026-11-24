package reviews

import (
	"time"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/storage"
)

// CreateRequest is the API payload for reviewing a place. The place comes from
// the path and the author from the access token.
type CreateRequest struct {
	Rating  int     `json:"rating"`
	Comment *string `json:"comment"`
}

// Response is the API representation of a domain.Review.
type Response struct {
	ID        int64          `json:"id"`
	PlaceID   int64          `json:"place_id"`
	PlaceName string         `json:"place_name"`
	Rating    int            `json:"rating"`
	Comment   *string        `json:"comment"`
	CreatedAt time.Time      `json:"created_at"`
	User      AuthorResponse `json:"user"`
	// Medias are the review's photos in position order, or [] if it has none.
	Medias []storage.MediaResponse `json:"medias"`
}

// AuthorResponse is the API representation of a review's domain.Author.
type AuthorResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// AvatarURL is the author's profile photo link, or null without one.
	AvatarURL *string `json:"avatar_url"`
}

// toReview builds the review to save from the place, the author, and the request body.
func toReview(placeID, userID int64, req CreateRequest) *domain.Review {
	return &domain.Review{
		PlaceID: placeID,
		UserID:  userID,
		Rating:  req.Rating,
		Comment: req.Comment,
	}
}

// toResponse turns r into the JSON shape sent to clients, with media keys
// turned into URLs by mediaURL.
func toResponse(r domain.Review, mediaURL func(key string) string) Response {
	return Response{
		ID:        r.ID,
		PlaceID:   r.PlaceID,
		PlaceName: r.PlaceName,
		Rating:    r.Rating,
		Comment:   r.Comment,
		CreatedAt: r.CreatedAt,
		User: AuthorResponse{
			ID:        r.User.ID,
			Name:      r.User.Name,
			AvatarURL: storage.OptionalURL(r.User.AvatarKey, mediaURL),
		},
		Medias: storage.ToMediaResponses(r.Medias, mediaURL),
	}
}
