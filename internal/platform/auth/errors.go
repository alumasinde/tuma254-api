package auth

import "errors"

var (
	ErrMissingToken       = errors.New("missing bearer token")
	ErrInvalidToken       = errors.New("invalid access token")
	ErrForbidden          = errors.New("forbidden")
	ErrPrincipalNotFound  = errors.New("authenticated principal not found")
)
