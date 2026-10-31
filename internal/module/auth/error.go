package auth

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidName         = errors.New("name is required")
	ErrNameTooLong         = fmt.Errorf("name must be %d characters or fewer", maxNameLength)
	ErrInvalidEmail        = errors.New("invalid email")
	ErrEmailTooLong        = fmt.Errorf("email must be %d characters or fewer", maxEmailLength)
	ErrWeakPassword        = fmt.Errorf("password must be at least %d characters", minPasswordLength)
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrInvalidHash         = errors.New("invalid argon2id hash")
)
