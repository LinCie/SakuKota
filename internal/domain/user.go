package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

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
	// AvatarKey is the storage key of the profile photo, or nil without one.
	AvatarKey *string `db:"avatar_key"`
}

// Author is the user who wrote a Review, submitted a Suggestion, or filed a
// Report, as shown next to it.
type Author struct {
	ID   int64  `db:"id"`
	Name string `db:"name"`
	// AvatarKey is the storage key of the profile photo, or nil without one.
	AvatarKey *string `db:"avatar_key"`
}

// NormalizeUserName trims name and rejects it if it is empty or too long.
func NormalizeUserName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return "", ErrUserNameRequired
	case utf8.RuneCountInString(name) > MaxUserNameLength:
		return "", ErrUserNameTooLong
	}

	return name, nil
}
