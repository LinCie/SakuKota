package auth

// Tokens is an access token and refresh token pair issued to a User.
type Tokens struct {
	AccessToken  string
	RefreshToken string
}

const minPasswordLength = 8
