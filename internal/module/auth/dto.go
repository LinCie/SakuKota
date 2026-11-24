package auth

import (
	"time"

	"sakukota-go/internal/domain"
)

// RegisterRequest is the API payload for registering a domain.User.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the API payload for logging in a domain.User.
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

// UserResponse is the API representation of a domain.User.
type UserResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toUser turns the sign-up body into a User. The password is set later.
func toUser(req RegisterRequest) *domain.User {
	return &domain.User{
		Name:  req.Name,
		Email: req.Email,
	}
}

// toSessionResponse builds the login or refresh reply from the user and access token.
func toSessionResponse(u *domain.User, t *Tokens) SessionResponse {
	return SessionResponse{
		User:        toUserResponse(u),
		AccessToken: t.AccessToken,
	}
}

// toUserResponse turns u into the JSON shape sent to clients.
func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
