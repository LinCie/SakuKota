package auth

import (
	"time"

	"rantau-go/internal/module/users"
)

// RegisterRequest is the API payload for registering a User.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the API payload for logging in a User.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SessionResponse is the API response for a successful login or refresh. The
// refresh token is never in the body; it is set as an HttpOnly cookie.
type SessionResponse struct {
	User        UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
}

// UserResponse is the API representation of a User.
type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// -----------------------------------------------------------------------------
// Conversions
// -----------------------------------------------------------------------------

func toUser(req RegisterRequest) *users.User {
	return &users.User{
		Name:  req.Name,
		Email: req.Email,
	}
}

func toSessionResponse(u *users.User, t *Tokens) SessionResponse {
	return SessionResponse{
		User:        toUserResponse(u),
		AccessToken: t.AccessToken,
	}
}

func toUserResponse(u *users.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
