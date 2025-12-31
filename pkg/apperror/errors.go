package apperror

import (
	"errors"
	"net/http"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrBadRequest        = errors.New("bad request")
	ErrInternal          = errors.New("internal server error")
	ErrInvalidInput      = errors.New("invalid input")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// AppError is a custom error type that can hold an HTTP status code
type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// MapErrorToStatus maps common errors to HTTP status codes
func MapErrorToStatus(err error) int {
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrUnauthorized) {
		return http.StatusUnauthorized
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden
	}
	if errors.Is(err, ErrBadRequest) || errors.Is(err, ErrInvalidInput) {
		return http.StatusBadRequest
	}
	if errors.Is(err, ErrRateLimitExceeded) {
		return http.StatusTooManyRequests
	}
	// Default to internal server error
	return http.StatusInternalServerError
}
