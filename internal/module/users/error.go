package users

import "errors"

const CodeEmailTaken = "EMAIL_TAKEN"

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")
)
