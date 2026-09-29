package auth

import "time"

// User represents a row in the USERS table.
type User struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	Role         string    `db:"role"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Tokens is an access token and refresh token pair issued to a User.
type Tokens struct {
	AccessToken  string
	RefreshToken string
}

const minPasswordLength = 8
