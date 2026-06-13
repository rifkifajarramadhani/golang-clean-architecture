package user

import "errors"

var (
	ErrNotFound          = errors.New("user not found")
	ErrDuplicateEmail    = errors.New("email already exists")
	ErrDuplicateUsername = errors.New("username already exists")
	ErrInvalidInput      = errors.New("invalid user input")
	ErrForbidden         = errors.New("forbidden")
	ErrLastAdmin         = errors.New("cannot remove the last admin")
	ErrInvalidRole       = errors.New("invalid role")
	ErrInvalidPassword   = errors.New("invalid password")
)
