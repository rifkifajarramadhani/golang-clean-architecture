package auth

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrExpiredToken       = errors.New("token expired")
	ErrEmailUnverified    = errors.New("email is not verified")
	ErrVerificationRate   = errors.New("verification request rate exceeded")
)
