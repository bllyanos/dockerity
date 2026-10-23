package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors shared by services and stores. The API maps them to HTTP status codes.
var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// ValidationError reports invalid user input for one field.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Invalid returns a ValidationError for field.
func Invalid(field, format string, args ...any) error {
	return &ValidationError{Field: field, Message: fmt.Sprintf(format, args...)}
}

// sentinelError carries a caller-facing message and matches a sentinel with errors.Is.
type sentinelError struct {
	msg      string
	sentinel error
}

func (e *sentinelError) Error() string { return e.msg }
func (e *sentinelError) Unwrap() error { return e.sentinel }

// NotFoundf returns an error with a clear message that matches ErrNotFound.
func NotFoundf(format string, args ...any) error {
	return &sentinelError{msg: fmt.Sprintf(format, args...), sentinel: ErrNotFound}
}

// Conflictf returns an error with a clear message that matches ErrConflict.
func Conflictf(format string, args ...any) error {
	return &sentinelError{msg: fmt.Sprintf(format, args...), sentinel: ErrConflict}
}
