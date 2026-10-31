package auth

import "errors"

const (
	CodeInvalidCredentials  = "INVALID_CREDENTIALS"
	CodeInvalidRefreshToken = "INVALID_REFRESH_TOKEN"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrWeakPassword        = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrInvalidHash         = errors.New("invalid argon2id hash")
)
