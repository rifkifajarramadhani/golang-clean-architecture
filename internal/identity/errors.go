package identity

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrDuplicateUsername  = errors.New("username already exists")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrNotFound           = errors.New("not found")
	ErrValidation         = errors.New("validation failed")
)
