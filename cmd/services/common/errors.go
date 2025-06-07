package common

import "fmt"

// ErrorCode is a simple identifier for error classification.
type ErrorCode string

const (
	// ErrCodeNotFound indicates a requested resource was not found.
	ErrCodeNotFound ErrorCode = "NotFound"
	// ErrCodeInvalidInput indicates provided data failed validation.
	ErrCodeInvalidInput ErrorCode = "InvalidInput"
	// ErrCodeAWS represents an AWS related error.
	ErrCodeAWS ErrorCode = "AWS"
	// ErrCodeInternal is used for unexpected internal failures.
	ErrCodeInternal ErrorCode = "Internal"
)

// ServiceError provides a minimal structured error type for services.
type ServiceError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// Error implements the error interface.
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *ServiceError) Unwrap() error { return e.Err }

// New creates a new ServiceError with the given code and message.
func New(code ErrorCode, msg string) *ServiceError {
	return &ServiceError{Code: code, Message: msg}
}

// Wrap wraps an underlying error with a ServiceError.
func Wrap(code ErrorCode, msg string, err error) *ServiceError {
	return &ServiceError{Code: code, Message: msg, Err: err}
}
