package domain

import "time"

// RoleAdmin is the role of users who may manage places, sub-districts and
// suggestions.
const RoleAdmin = "admin"

// Lengths in characters, counted after trimming spaces. They match the
// VARCHAR columns of USERS.
const (
	MaxUserNameLength = 100
	MaxEmailLength    = 255
)

// MinPasswordLength is the fewest characters a password may have.
const MinPasswordLength = 8

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

// Author is the user who wrote a Review or submitted a Suggestion, as shown
// next to it.
type Author struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
}
