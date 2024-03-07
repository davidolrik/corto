package handlers

import "errors"

// Sentinel errors returned by stores so handlers can map them to the right
// HTTP status instead of masking everything as one error.
var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
