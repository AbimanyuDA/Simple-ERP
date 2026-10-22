package domain

import "errors"

var (
	ErrInvalidPassword = errors.New("password must have at least 12 characters including uppercase, lowercase, number, and symbol")
	ErrInvalidRole     = errors.New("role must be STAFF, SUPERVISOR, or SUPER_ADMIN")
	ErrInvalidState    = errors.New("invalid state transition")
)
