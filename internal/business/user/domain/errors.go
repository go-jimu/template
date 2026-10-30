package domain

import "errors"

var (
	// ErrUserNotFound means the requested User Aggregate does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrConcurrentModification means a stale Aggregate version was saved.
	ErrConcurrentModification = errors.New("user was modified concurrently")
	// ErrInvalidUser means the Aggregate cannot satisfy its required business data.
	ErrInvalidUser = errors.New("invalid user")
	// ErrInvalidPassword means a raw password violates the credential policy.
	ErrInvalidPassword = errors.New("password must contain between 8 and 72 bytes")
	// ErrPasswordMismatch means the supplied current password is incorrect.
	ErrPasswordMismatch = errors.New("current password is incorrect")
)
