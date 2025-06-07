package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode identifies a particular error condition.
type ErrorCode string

// Default error codes used in this package.
const (
	ErrUnknown        ErrorCode = "UNKNOWN"
	ErrMultipleErrors ErrorCode = "MULTIPLE_ERRORS"
)

// ErrorCategory classifies the source of an error.
type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
)

// ErrorSeverity indicates how severe an error is.
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
)

// FogError represents a structured error with context.
type FogError interface {
	error

	// Core information
	Code() ErrorCode
	Message() string
	Details() string

	// Context information
	Operation() string
	Component() string
	Timestamp() time.Time

	// Classification
	Category() ErrorCategory
	Severity() ErrorSeverity
	Retryable() bool

	// Debugging
	StackTrace() []string
	Cause() error

	// User facing
	UserMessage() string
	Suggestions() []string

	// Structured data
	Fields() map[string]interface{}
	WithField(key string, value interface{}) FogError
	WithFields(fields map[string]interface{}) FogError
}

// BaseError implements FogError.
type BaseError struct {
	code        ErrorCode
	message     string
	details     string
	operation   string
	component   string
	timestamp   time.Time
	category    ErrorCategory
	severity    ErrorSeverity
	retryable   bool
	stackTrace  []string
	cause       error
	userMessage string
	suggestions []string
	fields      map[string]interface{}
}

// NewError creates a new BaseError with default metadata.
func NewError(code ErrorCode, message string) *BaseError {
	return &BaseError{
		code:      code,
		message:   message,
		timestamp: time.Now(),
		fields:    make(map[string]interface{}),
		category:  GetErrorCategory(code),
		severity:  GetErrorSeverity(code),
		retryable: IsRetryable(code),
	}
}

// Error implements the error interface.
func (e *BaseError) Error() string {
	if e.details != "" {
		return fmt.Sprintf("%s: %s", e.message, e.details)
	}
	return e.message
}

// Code returns the error code.
func (e *BaseError) Code() ErrorCode { return e.code }

// Message returns the message.
func (e *BaseError) Message() string { return e.message }

// Details returns additional details.
func (e *BaseError) Details() string { return e.details }

// Operation returns the operation context.
func (e *BaseError) Operation() string { return e.operation }

// Component returns the component context.
func (e *BaseError) Component() string { return e.component }

// Timestamp returns when the error occurred.
func (e *BaseError) Timestamp() time.Time { return e.timestamp }

// Category returns the error category.
func (e *BaseError) Category() ErrorCategory { return e.category }

// Severity returns the error severity.
func (e *BaseError) Severity() ErrorSeverity { return e.severity }

// Retryable indicates if the error is retryable.
func (e *BaseError) Retryable() bool { return e.retryable }

// StackTrace returns the captured stack trace.
func (e *BaseError) StackTrace() []string { return e.stackTrace }

// Cause returns the underlying error.
func (e *BaseError) Cause() error { return e.cause }

// UserMessage returns a user friendly message.
func (e *BaseError) UserMessage() string {
	if e.userMessage != "" {
		return e.userMessage
	}
	return e.message
}

// Suggestions returns suggested actions.
func (e *BaseError) Suggestions() []string { return e.suggestions }

// Fields returns structured fields.
func (e *BaseError) Fields() map[string]interface{} { return e.fields }

// WithField returns a copy of the error with an additional field.
func (e *BaseError) WithField(key string, value interface{}) FogError {
	newErr := *e
	newErr.fields = copyMap(e.fields)
	newErr.fields[key] = value
	return &newErr
}

// WithFields returns a copy of the error with additional fields.
func (e *BaseError) WithFields(fields map[string]interface{}) FogError {
	newErr := *e
	newErr.fields = copyMap(e.fields)
	for k, v := range fields {
		newErr.fields[k] = v
	}
	return &newErr
}

// WithDetails sets details on the error.
func (e *BaseError) WithDetails(details string) *BaseError {
	e.details = details
	return e
}

// WithOperation sets the operation context.
func (e *BaseError) WithOperation(op string) *BaseError {
	e.operation = op
	return e
}

// WithComponent sets the component context.
func (e *BaseError) WithComponent(comp string) *BaseError {
	e.component = comp
	return e
}

// WithCause sets the underlying cause.
func (e *BaseError) WithCause(err error) *BaseError {
	e.cause = err
	return e
}

// WithUserMessage sets a user friendly message.
func (e *BaseError) WithUserMessage(msg string) *BaseError {
	e.userMessage = msg
	return e
}

// WithSuggestions sets suggested actions.
func (e *BaseError) WithSuggestions(s []string) *BaseError {
	e.suggestions = s
	return e
}

// WithStackTrace captures and stores the current stack trace.
func (e *BaseError) WithStackTrace() *BaseError {
	e.stackTrace = captureStackTrace()
	return e
}

// ErrorAggregator collects multiple errors.
type ErrorAggregator struct {
	errors   []FogError
	category ErrorCategory
	context  string
}

// NewErrorAggregator creates a new aggregator with context.
func NewErrorAggregator(ctx string) *ErrorAggregator {
	return &ErrorAggregator{
		errors:  make([]FogError, 0),
		context: ctx,
	}
}

// Add adds an error to the aggregator.
func (a *ErrorAggregator) Add(err FogError) {
	a.errors = append(a.errors, err)
	if err.Category() > a.category {
		a.category = err.Category()
	}
}

// HasErrors reports whether errors were added.
func (a *ErrorAggregator) HasErrors() bool { return len(a.errors) > 0 }

// Count returns the number of collected errors.
func (a *ErrorAggregator) Count() int { return len(a.errors) }

// Errors returns all collected errors.
func (a *ErrorAggregator) Errors() []FogError { return a.errors }

// FirstError returns the first error or nil.
func (a *ErrorAggregator) FirstError() FogError {
	if len(a.errors) == 0 {
		return nil
	}
	return a.errors[0]
}

// ToError converts the aggregation to a single error.
func (a *ErrorAggregator) ToError() error {
	switch len(a.errors) {
	case 0:
		return nil
	case 1:
		return a.errors[0]
	default:
		return NewMultiError(a.context, a.errors)
	}
}

// MultiError represents multiple aggregated errors.
type MultiError struct {
	*BaseError
	errors []FogError
}

// NewMultiError creates a MultiError for the given context and errors.
func NewMultiError(context string, errs []FogError) *MultiError {
	base := NewError(ErrMultipleErrors, fmt.Sprintf("Multiple errors in %s", context))
	base = base.WithOperation(context)
	return &MultiError{BaseError: base, errors: errs}
}

// Errors returns the individual errors.
func (m *MultiError) Errors() []FogError { return m.errors }

// Error implements the error interface.
func (m *MultiError) Error() string {
	if len(m.errors) == 1 {
		return m.errors[0].Error()
	}
	return fmt.Sprintf("%s (%d errors)", m.BaseError.Error(), len(m.errors))
}

// Helper to copy a map.
func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// captureStackTrace captures the current call stack for debugging.
func captureStackTrace() []string {
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)
	stack := make([]string, 0, n)
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d", frame.Function, frame.Line))
		if !more {
			break
		}
	}
	return stack
}

// GetErrorCategory returns the category for an error code.
func GetErrorCategory(code ErrorCode) ErrorCategory { return CategoryUnknown }

// GetErrorSeverity returns the severity for an error code.
func GetErrorSeverity(code ErrorCode) ErrorSeverity { return SeverityLow }

// IsRetryable returns whether the code is retryable.
func IsRetryable(code ErrorCode) bool { return false }
