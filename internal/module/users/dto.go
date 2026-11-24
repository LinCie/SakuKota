package users

import (
	"time"

	"sakukota-go/internal/domain"
	"sakukota-go/internal/storage"
)

// RenameRequest is the API payload for changing the logged-in user's name.
type RenameRequest struct {
	Name string `json:"name"`
}

// Response is the API representation of the logged-in user's profile.
type Response struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
	// AvatarURL is the profile photo's link, or null without one.
	AvatarURL *string   `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toResponse turns u into the JSON shape sent to clients, with the photo key
// turned into a URL by mediaURL.
func toResponse(u *domain.User, mediaURL func(key string) string) Response {
	return Response{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		AvatarURL: storage.OptionalURL(u.AvatarKey, mediaURL),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
