package utils

import "errors"

// Sentinel errors for common failure cases across services and repositories.
// Use errors.Is to check for these, e.g. errors.Is(err, utils.ErrNotFound).
var (
	ErrNotFound     = errors.New("resource not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("resource conflict")
)

// AppError is an application error carrying an HTTP status code alongside
// a user-facing message, for handlers to translate into JSON responses.
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError wraps err with an HTTP status code and user-facing message.
func NewAppError(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}
