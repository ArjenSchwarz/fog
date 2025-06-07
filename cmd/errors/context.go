package errors

import (
	"context"
	"fmt"
)

// ErrorCode identifies a type of error.
type ErrorCode string

// FogError represents an error with additional context.
type FogError interface {
	error
	Code() ErrorCode
	Message() string
	Operation() string
	Component() string
	Fields() map[string]interface{}
	Cause() error
}

// BaseError is a basic implementation of FogError.
type BaseError struct {
	code      ErrorCode
	message   string
	operation string
	component string
	cause     error
	fields    map[string]interface{}
}

// NewError creates a new BaseError with the provided code and message.
func NewError(code ErrorCode, message string) *BaseError {
	return &BaseError{
		code:    code,
		message: message,
		fields:  make(map[string]interface{}),
	}
}

// Error implements the error interface.
func (e *BaseError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.message, e.cause)
	}
	return e.message
}

// Code returns the error code.
func (e *BaseError) Code() ErrorCode { return e.code }

// Message returns the error message.
func (e *BaseError) Message() string { return e.message }

// Operation returns the associated operation.
func (e *BaseError) Operation() string { return e.operation }

// Component returns the originating component.
func (e *BaseError) Component() string { return e.component }

// Fields returns any additional fields.
func (e *BaseError) Fields() map[string]interface{} { return e.fields }

// Cause returns the underlying cause.
func (e *BaseError) Cause() error { return e.cause }

// WithOperation sets the operation on the error.
func (e *BaseError) WithOperation(op string) *BaseError {
	e.operation = op
	return e
}

// WithComponent sets the component on the error.
func (e *BaseError) WithComponent(comp string) *BaseError {
	e.component = comp
	return e
}

// WithFields adds multiple fields to the error.
func (e *BaseError) WithFields(fields map[string]interface{}) *BaseError {
	for k, v := range fields {
		e.fields[k] = v
	}
	return e
}

// WithCause sets the underlying cause.
func (e *BaseError) WithCause(cause error) *BaseError {
	e.cause = cause
	return e
}

// ErrorContext provides contextual information for errors.
type ErrorContext struct {
	Operation     string
	Component     string
	StackName     string
	TemplatePath  string
	Region        string
	Account       string
	RequestID     string
	CorrelationID string
	UserID        string
	Fields        map[string]interface{}
}

// NewErrorContext creates a new error context.
func NewErrorContext(operation, component string) *ErrorContext {
	return &ErrorContext{
		Operation: operation,
		Component: component,
		Fields:    make(map[string]interface{}),
	}
}

// WithStackName adds a stack name to the context.
func (ec *ErrorContext) WithStackName(stackName string) *ErrorContext {
	ec.StackName = stackName
	return ec
}

// WithTemplate adds a template path to the context.
func (ec *ErrorContext) WithTemplate(templatePath string) *ErrorContext {
	ec.TemplatePath = templatePath
	return ec
}

// WithRegion adds a region to the context.
func (ec *ErrorContext) WithRegion(region string) *ErrorContext {
	ec.Region = region
	return ec
}

// WithAccount adds an account to the context.
func (ec *ErrorContext) WithAccount(account string) *ErrorContext {
	ec.Account = account
	return ec
}

// WithRequestID adds an AWS request ID to the context.
func (ec *ErrorContext) WithRequestID(requestID string) *ErrorContext {
	ec.RequestID = requestID
	return ec
}

// WithCorrelationID adds a correlation ID to the context.
func (ec *ErrorContext) WithCorrelationID(correlationID string) *ErrorContext {
	ec.CorrelationID = correlationID
	return ec
}

// WithField adds a custom field to the context.
func (ec *ErrorContext) WithField(key string, value interface{}) *ErrorContext {
	ec.Fields[key] = value
	return ec
}

// ToMap converts the context to a map for structured logging.
func (ec *ErrorContext) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	if ec.Operation != "" {
		result["operation"] = ec.Operation
	}
	if ec.Component != "" {
		result["component"] = ec.Component
	}
	if ec.StackName != "" {
		result["stack_name"] = ec.StackName
	}
	if ec.TemplatePath != "" {
		result["template_path"] = ec.TemplatePath
	}
	if ec.Region != "" {
		result["region"] = ec.Region
	}
	if ec.Account != "" {
		result["account"] = ec.Account
	}
	if ec.RequestID != "" {
		result["request_id"] = ec.RequestID
	}
	if ec.CorrelationID != "" {
		result["correlation_id"] = ec.CorrelationID
	}
	for k, v := range ec.Fields {
		result[k] = v
	}
	return result
}

// ContextualError creates a new error using the provided context.
func ContextualError(ctx *ErrorContext, code ErrorCode, message string) FogError {
	err := NewError(code, message).
		WithOperation(ctx.Operation).
		WithComponent(ctx.Component).
		WithFields(ctx.ToMap())
	return err
}

// WrapError wraps an existing error with context information.
func WrapError(ctx *ErrorContext, cause error, code ErrorCode, message string) FogError {
	err := NewError(code, message).
		WithCause(cause).
		WithOperation(ctx.Operation).
		WithComponent(ctx.Component).
		WithFields(ctx.ToMap())
	return err
}

type contextKey string

const ErrorContextKey contextKey = "error_context"

// WithErrorContext attaches an ErrorContext to a Go context.Context.
func WithErrorContext(ctx context.Context, errorCtx *ErrorContext) context.Context {
	return context.WithValue(ctx, ErrorContextKey, errorCtx)
}

// GetErrorContext retrieves the ErrorContext from a Go context.Context.
// If no context is present, a default unknown context is returned.
func GetErrorContext(ctx context.Context) *ErrorContext {
	if errorCtx, ok := ctx.Value(ErrorContextKey).(*ErrorContext); ok {
		return errorCtx
	}
	return NewErrorContext("unknown", "unknown")
}
