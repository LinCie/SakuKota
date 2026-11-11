package user

import "time"

// RoleAdmin is the role of users who may manage places, sub-districts and
// suggestions.
const RoleAdmin = "admin"

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
