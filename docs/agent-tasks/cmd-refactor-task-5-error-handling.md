# Task 5: Error Handling Standardization

## Objective

Create a comprehensive error handling system with structured error types, consistent error messaging, and proper error propagation throughout the command layer to improve debugging and user experience.

## Current State

### Problems
- Inconsistent error messages across commands
- Generic error handling without context
- Poor error propagation from services to UI
- No structured error types or codes
- Limited debugging information in errors
- Mixed error handling patterns throughout codebase

### Current Error Implementation
- Basic `fmt.Errorf` usage throughout codebase
- No error categorization or typing
- Limited context in error messages
- Inconsistent error formatting
- No error aggregation for validation
- Missing stack traces for debugging

### Problematic Patterns
```go
// Current: Generic error handling
if err != nil {
    return fmt.Errorf("deployment failed: %w", err)
}

// Inconsistent error messages
return errors.New("invalid template")
return fmt.Errorf("Template validation failed")
return fmt.Errorf("ERROR: Cannot read template file")
```

## Target State

### Goals
- Structured error types with consistent formatting
- Rich error context with operation details
- Error categorization for appropriate handling
- Standardized error codes for programmatic handling
- Comprehensive error aggregation for validation
- Enhanced debugging information

### Error Handling Architecture
```
cmd/
├── errors/
│   ├── types.go               # Error type definitions
│   ├── codes.go               # Error code constants
│   ├── context.go             # Error context handling
│   ├── aggregation.go         # Error collection and aggregation
│   ├── formatting.go          # Error message formatting
│   └── recovery.go            # Error recovery strategies
├── middleware/
│   ├── error_handler.go       # Error handling middleware
│   └── recovery.go            # Panic recovery middleware
└── validation/
    ├── errors.go              # Validation-specific errors
    └── aggregator.go          # Validation error aggregation
```

## Prerequisites

- Task 1: Command Structure Reorganization (provides middleware framework)

## Step-by-Step Implementation

### Step 1: Define Error Types and Interfaces

**File**: `cmd/errors/types.go`

```go
package errors

import (
    "fmt"
    "time"
)

// FogError represents a structured error with context
type FogError interface {
    error

    // Core error information
    Code() ErrorCode
    Message() string
    Details() string

    // Context information
    Operation() string
    Component() string
    Timestamp() time.Time

    // Error classification
    Category() ErrorCategory
    Severity() ErrorSeverity
    Retryable() bool

    // Stack trace and debugging
    StackTrace() []string
    Cause() error

    // User-facing information
    UserMessage() string
    Suggestions() []string

    // Structured data
    Fields() map[string]interface{}
    WithField(key string, value interface{}) FogError
    WithFields(fields map[string]interface{}) FogError
}

// BaseError implements the FogError interface
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

// NewError creates a new BaseError
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

// Error implements the error interface
func (e *BaseError) Error() string {
    if e.details != "" {
        return fmt.Sprintf("%s: %s", e.message, e.details)
    }
    return e.message
}

// Code returns the error code
func (e *BaseError) Code() ErrorCode {
    return e.code
}

// Message returns the error message
func (e *BaseError) Message() string {
    return e.message
}

// Details returns additional error details
func (e *BaseError) Details() string {
    return e.details
}

// Operation returns the operation that caused the error
func (e *BaseError) Operation() string {
    return e.operation
}

// Component returns the component where the error occurred
func (e *BaseError) Component() string {
    return e.component
}

// Timestamp returns when the error occurred
func (e *BaseError) Timestamp() time.Time {
    return e.timestamp
}

// Category returns the error category
func (e *BaseError) Category() ErrorCategory {
    return e.category
}

// Severity returns the error severity
func (e *BaseError) Severity() ErrorSeverity {
    return e.severity
}

// Retryable returns whether the operation can be retried
func (e *BaseError) Retryable() bool {
    return e.retryable
}

// StackTrace returns the stack trace
func (e *BaseError) StackTrace() []string {
    return e.stackTrace
}

// Cause returns the underlying cause
func (e *BaseError) Cause() error {
    return e.cause
}

// UserMessage returns a user-friendly message
func (e *BaseError) UserMessage() string {
    if e.userMessage != "" {
        return e.userMessage
    }
    return e.message
}

// Suggestions returns suggested actions
func (e *BaseError) Suggestions() []string {
    return e.suggestions
}

// Fields returns the error fields
func (e *BaseError) Fields() map[string]interface{} {
    return e.fields
}

// WithField adds a field to the error
func (e *BaseError) WithField(key string, value interface{}) FogError {
    newError := *e
    newError.fields = make(map[string]interface{})
    for k, v := range e.fields {
        newError.fields[k] = v
    }
    newError.fields[key] = value
    return &newError
}

// WithFields adds multiple fields to the error
func (e *BaseError) WithFields(fields map[string]interface{}) FogError {
    newError := *e
    newError.fields = make(map[string]interface{})
    for k, v := range e.fields {
        newError.fields[k] = v
    }
    for k, v := range fields {
        newError.fields[k] = v
    }
    return &newError
}

// WithDetails sets additional details
func (e *BaseError) WithDetails(details string) *BaseError {
    e.details = details
    return e
}

// WithOperation sets the operation context
func (e *BaseError) WithOperation(operation string) *BaseError {
    e.operation = operation
    return e
}

// WithComponent sets the component context
func (e *BaseError) WithComponent(component string) *BaseError {
    e.component = component
    return e
}

// WithCause sets the underlying cause
func (e *BaseError) WithCause(cause error) *BaseError {
    e.cause = cause
    return e
}

// WithUserMessage sets a user-friendly message
func (e *BaseError) WithUserMessage(message string) *BaseError {
    e.userMessage = message
    return e
}

// WithSuggestions sets suggested actions
func (e *BaseError) WithSuggestions(suggestions []string) *BaseError {
    e.suggestions = suggestions
    return e
}

// WithStackTrace captures the current stack trace
func (e *BaseError) WithStackTrace() *BaseError {
    e.stackTrace = captureStackTrace()
    return e
}

// ErrorAggregator collects multiple errors
type ErrorAggregator struct {
    errors   []FogError
    category ErrorCategory
    context  string
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator(context string) *ErrorAggregator {
    return &ErrorAggregator{
        errors:  make([]FogError, 0),
        context: context,
    }
}

// Add adds an error to the aggregator
func (a *ErrorAggregator) Add(err FogError) {
    a.errors = append(a.errors, err)

    // Update category to the highest severity
    if err.Category() > a.category {
        a.category = err.Category()
    }
}

// HasErrors returns true if there are errors
func (a *ErrorAggregator) HasErrors() bool {
    return len(a.errors) > 0
}

// Count returns the number of errors
func (a *ErrorAggregator) Count() int {
    return len(a.errors)
}

// Errors returns all collected errors
func (a *ErrorAggregator) Errors() []FogError {
    return a.errors
}

// FirstError returns the first error
func (a *ErrorAggregator) FirstError() FogError {
    if len(a.errors) == 0 {
        return nil
    }
    return a.errors[0]
}

// ToError converts the aggregator to a single error
func (a *ErrorAggregator) ToError() error {
    if len(a.errors) == 0 {
        return nil
    }

    if len(a.errors) == 1 {
        return a.errors[0]
    }

    return NewMultiError(a.context, a.errors)
}

// MultiError represents multiple errors
type MultiError struct {
    *BaseError
    errors []FogError
}

// NewMultiError creates a new multi-error
func NewMultiError(context string, errors []FogError) *MultiError {
    baseErr := NewError(ErrMultipleErrors, fmt.Sprintf("Multiple errors in %s", context))
    baseErr.WithOperation(context)

    return &MultiError{
        BaseError: baseErr,
        errors:    errors,
    }
}

// Errors returns the individual errors
func (m *MultiError) Errors() []FogError {
    return m.errors
}

// Error returns a formatted error message
func (m *MultiError) Error() string {
    if len(m.errors) == 1 {
        return m.errors[0].Error()
    }

    return fmt.Sprintf("%s (%d errors)", m.BaseError.Error(), len(m.errors))
}

// Private helper functions

func captureStackTrace() []string {
    // Implementation to capture stack trace
    // This would use runtime.Caller() to get the call stack
    return []string{"stack trace not implemented"}
}
```

### Step 2: Define Error Codes and Categories

**File**: `cmd/errors/codes.go`

```go
package errors

// ErrorCode represents a specific error condition
type ErrorCode string

// Error categories for classification
type ErrorCategory int

const (
    CategoryUnknown ErrorCategory = iota
    CategoryValidation
    CategoryConfiguration
    CategoryNetwork
    CategoryAWS
    CategoryFileSystem
    CategoryTemplate
    CategoryPermission
    CategoryResource
    CategoryInternal
)

// Error severity levels
type ErrorSeverity int

const (
    SeverityLow ErrorSeverity = iota
    SeverityMedium
    SeverityHigh
    SeverityCritical
)

// General error codes
const (
    ErrUnknown         ErrorCode = "UNKNOWN"
    ErrInternal        ErrorCode = "INTERNAL"
    ErrNotImplemented  ErrorCode = "NOT_IMPLEMENTED"
    ErrMultipleErrors  ErrorCode = "MULTIPLE_ERRORS"
)

// Validation error codes
const (
    ErrValidationFailed    ErrorCode = "VALIDATION_FAILED"
    ErrRequiredField       ErrorCode = "REQUIRED_FIELD"
    ErrInvalidValue        ErrorCode = "INVALID_VALUE"
    ErrInvalidFormat       ErrorCode = "INVALID_FORMAT"
    ErrConflictingFlags    ErrorCode = "CONFLICTING_FLAGS"
    ErrDependencyMissing   ErrorCode = "DEPENDENCY_MISSING"
)

// Configuration error codes
const (
    ErrConfigNotFound      ErrorCode = "CONFIG_NOT_FOUND"
    ErrConfigInvalid       ErrorCode = "CONFIG_INVALID"
    ErrConfigPermission    ErrorCode = "CONFIG_PERMISSION"
    ErrMissingCredentials  ErrorCode = "MISSING_CREDENTIALS"
    ErrInvalidCredentials  ErrorCode = "INVALID_CREDENTIALS"
)

// File system error codes
const (
    ErrFileNotFound        ErrorCode = "FILE_NOT_FOUND"
    ErrFilePermission      ErrorCode = "FILE_PERMISSION"
    ErrFileInvalid         ErrorCode = "FILE_INVALID"
    ErrDirectoryNotFound   ErrorCode = "DIRECTORY_NOT_FOUND"
    ErrDirectoryPermission ErrorCode = "DIRECTORY_PERMISSION"
)

// Template error codes
const (
    ErrTemplateNotFound    ErrorCode = "TEMPLATE_NOT_FOUND"
    ErrTemplateInvalid     ErrorCode = "TEMPLATE_INVALID"
    ErrTemplateTooLarge    ErrorCode = "TEMPLATE_TOO_LARGE"
    ErrTemplateUploadFailed ErrorCode = "TEMPLATE_UPLOAD_FAILED"
    ErrParameterInvalid    ErrorCode = "PARAMETER_INVALID"
    ErrParameterMissing    ErrorCode = "PARAMETER_MISSING"
)

// AWS error codes
const (
    ErrAWSAuthentication   ErrorCode = "AWS_AUTHENTICATION"
    ErrAWSPermission       ErrorCode = "AWS_PERMISSION"
    ErrAWSRateLimit        ErrorCode = "AWS_RATE_LIMIT"
    ErrAWSServiceError     ErrorCode = "AWS_SERVICE_ERROR"
    ErrAWSRegionInvalid    ErrorCode = "AWS_REGION_INVALID"
    ErrStackNotFound       ErrorCode = "STACK_NOT_FOUND"
    ErrStackInvalidState   ErrorCode = "STACK_INVALID_STATE"
    ErrChangesetFailed     ErrorCode = "CHANGESET_FAILED"
    ErrDeploymentFailed    ErrorCode = "DEPLOYMENT_FAILED"
    ErrDriftDetectionFailed ErrorCode = "DRIFT_DETECTION_FAILED"
)

// Network error codes
const (
    ErrNetworkTimeout      ErrorCode = "NETWORK_TIMEOUT"
    ErrNetworkConnection   ErrorCode = "NETWORK_CONNECTION"
    ErrNetworkUnreachable  ErrorCode = "NETWORK_UNREACHABLE"
)

// Resource error codes
const (
    ErrResourceNotFound    ErrorCode = "RESOURCE_NOT_FOUND"
    ErrResourceConflict    ErrorCode = "RESOURCE_CONFLICT"
    ErrResourceLimit       ErrorCode = "RESOURCE_LIMIT"
    ErrResourceLocked      ErrorCode = "RESOURCE_LOCKED"
)

// GetErrorCategory returns the category for an error code
func GetErrorCategory(code ErrorCode) ErrorCategory {
    switch code {
    case ErrValidationFailed, ErrRequiredField, ErrInvalidValue, ErrInvalidFormat, ErrConflictingFlags, ErrDependencyMissing:
        return CategoryValidation
    case ErrConfigNotFound, ErrConfigInvalid, ErrConfigPermission, ErrMissingCredentials, ErrInvalidCredentials:
        return CategoryConfiguration
    case ErrFileNotFound, ErrFilePermission, ErrFileInvalid, ErrDirectoryNotFound, ErrDirectoryPermission:
        return CategoryFileSystem
    case ErrTemplateNotFound, ErrTemplateInvalid, ErrTemplateTooLarge, ErrTemplateUploadFailed, ErrParameterInvalid, ErrParameterMissing:
        return CategoryTemplate
    case ErrAWSAuthentication, ErrAWSPermission, ErrAWSRateLimit, ErrAWSServiceError, ErrAWSRegionInvalid, ErrStackNotFound, ErrStackInvalidState, ErrChangesetFailed, ErrDeploymentFailed, ErrDriftDetectionFailed:
        return CategoryAWS
    case ErrNetworkTimeout, ErrNetworkConnection, ErrNetworkUnreachable:
        return CategoryNetwork
    case ErrResourceNotFound, ErrResourceConflict, ErrResourceLimit, ErrResourceLocked:
        return CategoryResource
    case ErrInternal, ErrNotImplemented:
        return CategoryInternal
    default:
        return CategoryUnknown
    }
}

// GetErrorSeverity returns the severity for an error code
func GetErrorSeverity(code ErrorCode) ErrorSeverity {
    switch code {
    case ErrInternal, ErrDeploymentFailed, ErrChangesetFailed:
        return SeverityCritical
    case ErrAWSAuthentication, ErrAWSPermission, ErrStackInvalidState, ErrConfigInvalid, ErrMissingCredentials:
        return SeverityHigh
    case ErrValidationFailed, ErrTemplateInvalid, ErrParameterInvalid, ErrFileNotFound, ErrNetworkTimeout:
        return SeverityMedium
    default:
        return SeverityLow
    }
}

// IsRetryable returns whether an error with the given code is retryable
func IsRetryable(code ErrorCode) bool {
    switch code {
    case ErrNetworkTimeout, ErrNetworkConnection, ErrAWSRateLimit, ErrAWSServiceError:
        return true
    case ErrValidationFailed, ErrRequiredField, ErrInvalidValue, ErrConfigInvalid, ErrFileNotFound, ErrTemplateInvalid:
        return false
    default:
        return false
    }
}

// ErrorCodeMetadata provides additional information about error codes
type ErrorCodeMetadata struct {
    Code        ErrorCode
    Category    ErrorCategory
    Severity    ErrorSeverity
    Retryable   bool
    Description string
    Suggestions []string
}

// GetErrorMetadata returns metadata for an error code
func GetErrorMetadata(code ErrorCode) ErrorCodeMetadata {
    metadata := ErrorCodeMetadata{
        Code:      code,
        Category:  GetErrorCategory(code),
        Severity:  GetErrorSeverity(code),
        Retryable: IsRetryable(code),
    }

    switch code {
    case ErrTemplateNotFound:
        metadata.Description = "CloudFormation template file not found"
        metadata.Suggestions = []string{
            "Check that the template file path is correct",
            "Ensure the file exists and is readable",
        }
    case ErrStackNotFound:
        metadata.Description = "CloudFormation stack does not exist"
        metadata.Suggestions = []string{
            "Verify the stack name is correct",
            "Check that you're in the correct AWS region",
            "Use 'fog list' to see available stacks",
        }
    case ErrAWSAuthentication:
        metadata.Description = "AWS authentication failed"
        metadata.Suggestions = []string{
            "Check your AWS credentials",
            "Verify AWS CLI configuration",
            "Ensure correct AWS region is set",
        }
    case ErrValidationFailed:
        metadata.Description = "Input validation failed"
        metadata.Suggestions = []string{
            "Review the validation errors",
            "Check command flags and arguments",
            "Refer to the command help for usage information",
        }
    // Add more cases as needed
    }

    return metadata
}
```

### Step 3: Implement Error Context and Formatting

**File**: `cmd/errors/context.go`

```go
package errors

import (
    "context"
    "fmt"
)

// ErrorContext provides contextual information for errors
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

// NewErrorContext creates a new error context
func NewErrorContext(operation, component string) *ErrorContext {
    return &ErrorContext{
        Operation: operation,
        Component: component,
        Fields:    make(map[string]interface{}),
    }
}

// WithStackName adds stack name to context
func (ec *ErrorContext) WithStackName(stackName string) *ErrorContext {
    ec.StackName = stackName
    return ec
}

// WithTemplate adds template path to context
func (ec *ErrorContext) WithTemplate(templatePath string) *ErrorContext {
    ec.TemplatePath = templatePath
    return ec
}

// WithRegion adds region to context
func (ec *ErrorContext) WithRegion(region string) *ErrorContext {
    ec.Region = region
    return ec
}

// WithAccount adds account to context
func (ec *ErrorContext) WithAccount(account string) *ErrorContext {
    ec.Account = account
    return ec
}

// WithRequestID adds AWS request ID to context
func (ec *ErrorContext) WithRequestID(requestID string) *ErrorContext {
    ec.RequestID = requestID
    return ec
}

// WithCorrelationID adds correlation ID to context
func (ec *ErrorContext) WithCorrelationID(correlationID string) *ErrorContext {
    ec.CorrelationID = correlationID
    return ec
}

// WithField adds a custom field to context
func (ec *ErrorContext) WithField(key string, value interface{}) *ErrorContext {
    ec.Fields[key] = value
    return ec
}

// ToMap converts the context to a map
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

// ContextualError creates an error with context
func ContextualError(ctx *ErrorContext, code ErrorCode, message string) FogError {
    err := NewError(code, message).
        WithOperation(ctx.Operation).
        WithComponent(ctx.Component).
        WithFields(ctx.ToMap())

    return err
}

// WrapError wraps an existing error with context
func WrapError(ctx *ErrorContext, cause error, code ErrorCode, message string) FogError {
    err := NewError(code, message).
        WithCause(cause).
        WithOperation(ctx.Operation).
        WithComponent(ctx.Component).
        WithFields(ctx.ToMap())

    return err
}

// Error context keys for context.Context
type contextKey string

const (
    ErrorContextKey contextKey = "error_context"
)

// WithErrorContext adds error context to a Go context
func WithErrorContext(ctx context.Context, errorCtx *ErrorContext) context.Context {
    return context.WithValue(ctx, ErrorContextKey, errorCtx)
}

// GetErrorContext retrieves error context from a Go context
func GetErrorContext(ctx context.Context) *ErrorContext {
    if errorCtx, ok := ctx.Value(ErrorContextKey).(*ErrorContext); ok {
        return errorCtx
    }
    return NewErrorContext("unknown", "unknown")
}
```

### Step 4: Implement Error Formatting

**File**: `cmd/errors/formatting.go`

```go
package errors

import (
    "fmt"
    "strings"
    "time"
)

// ErrorFormatter handles error formatting for different outputs
type ErrorFormatter interface {
    FormatError(err FogError) string
    FormatMultiError(err *MultiError) string
    FormatValidationErrors(errors []FogError) string
}

// ConsoleErrorFormatter formats errors for console output
type ConsoleErrorFormatter struct {
    colorEnabled bool
    verbose      bool
}

// NewConsoleErrorFormatter creates a new console error formatter
func NewConsoleErrorFormatter(colorEnabled, verbose bool) *ConsoleErrorFormatter {
    return &ConsoleErrorFormatter{
        colorEnabled: colorEnabled,
        verbose:      verbose,
    }
}

// FormatError formats a single error for console output
func (f *ConsoleErrorFormatter) FormatError(err FogError) string {
    var builder strings.Builder

    // Error header with code and severity
    severity := f.formatSeverity(err.Severity())
    builder.WriteString(fmt.Sprintf("%s [%s] %s\n", severity, err.Code(), err.Message()))

    // Details if available
    if details := err.Details(); details != "" {
        builder.WriteString(fmt.Sprintf("Details: %s\n", details))
    }

    // Context information
    if f.verbose {
        builder.WriteString(f.formatContext(err))
    }

    // Suggestions
    if suggestions := err.Suggestions(); len(suggestions) > 0 {
        builder.WriteString("Suggestions:\n")
        for _, suggestion := range suggestions {
            builder.WriteString(fmt.Sprintf("  • %s\n", suggestion))
        }
    }

    // Underlying cause
    if cause := err.Cause(); cause != nil && f.verbose {
        builder.WriteString(fmt.Sprintf("Caused by: %s\n", cause.Error()))
    }

    return builder.String()
}

// FormatMultiError formats multiple errors
func (f *ConsoleErrorFormatter) FormatMultiError(err *MultiError) string {
    var builder strings.Builder

    builder.WriteString(fmt.Sprintf("Multiple errors occurred (%d total):\n\n", len(err.Errors())))

    for i, subErr := range err.Errors() {
        builder.WriteString(fmt.Sprintf("%d. %s", i+1, f.FormatError(subErr)))
        if i < len(err.Errors())-1 {
            builder.WriteString("\n")
        }
    }

    return builder.String()
}

// FormatValidationErrors formats validation errors in a user-friendly way
func (f *ConsoleErrorFormatter) FormatValidationErrors(errors []FogError) string {
    var builder strings.Builder

    builder.WriteString("Validation failed with the following errors:\n\n")

    for i, err := range errors {
        builder.WriteString(fmt.Sprintf("%d. %s", i+1, err.UserMessage()))
        if details := err.Details(); details != "" {
            builder.WriteString(fmt.Sprintf("\n   %s", details))
        }
        builder.WriteString("\n")
    }

    return builder.String()
}

// formatSeverity formats the severity level with colors if enabled
func (f *ConsoleErrorFormatter) formatSeverity(severity ErrorSeverity) string {
    if !f.colorEnabled {
        return f.severityText(severity)
    }

    switch severity {
    case SeverityCritical:
        return fmt.Sprintf("\033[91m%s\033[0m", f.severityText(severity)) // Red
    case SeverityHigh:
        return fmt.Sprintf("\033[93m%s\033[0m", f.severityText(severity)) // Yellow
    case SeverityMedium:
        return fmt.Sprintf("\033[94m%s\033[0m", f.severityText(severity)) // Blue
    default:
        return f.severityText(severity)
    }
}

// severityText returns the text representation of severity
func (f *ConsoleErrorFormatter) severityText(severity ErrorSeverity) string {
    switch severity {
    case SeverityCritical:
        return "CRITICAL"
    case SeverityHigh:
        return "HIGH"
    case SeverityMedium:
        return "MEDIUM"
    case SeverityLow:
        return "LOW"
    default:
        return "UNKNOWN"
    }
}

// formatContext formats the error context
func (f *ConsoleErrorFormatter) formatContext(err FogError) string {
    var builder strings.Builder

    builder.WriteString("Context:\n")

    if operation := err.Operation(); operation != "" {
        builder.WriteString(fmt.Sprintf("  Operation: %s\n", operation))
    }

    if component := err.Component(); component != "" {
        builder.WriteString(fmt.Sprintf("  Component: %s\n", component))
    }

    builder.WriteString(fmt.Sprintf("  Timestamp: %s\n", err.Timestamp().Format(time.RFC3339)))

    if fields := err.Fields(); len(fields) > 0 {
        builder.WriteString("  Fields:\n")
        for key, value := range fields {
            builder.WriteString(fmt.Sprintf("    %s: %v\n", key, value))
        }
    }

    return builder.String()
}

// JSONErrorFormatter formats errors as JSON
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter creates a new JSON error formatter
func NewJSONErrorFormatter() *JSONErrorFormatter {
    return &JSONErrorFormatter{}
}

// FormatError formats an error as JSON
func (f *JSONErrorFormatter) FormatError(err FogError) string {
    // Implementation would marshal the error to JSON
    // This is a simplified version
    return fmt.Sprintf(`{
  "error": {
    "code": "%s",
    "message": "%s",
    "category": "%s",
    "severity": "%s",
    "timestamp": "%s",
    "operation": "%s",
    "component": "%s"
  }
}`, err.Code(), err.Message(), f.categoryName(err.Category()),
    f.severityName(err.Severity()), err.Timestamp().Format(time.RFC3339),
    err.Operation(), err.Component())
}

// FormatMultiError formats multiple errors as JSON
func (f *JSONErrorFormatter) FormatMultiError(err *MultiError) string {
    // Implementation would marshal multiple errors to JSON
    return `{"errors": []}`
}

// FormatValidationErrors formats validation errors as JSON
func (f *JSONErrorFormatter) FormatValidationErrors(errors []FogError) string {
    // Implementation would marshal validation errors to JSON
    return `{"validation_errors": []}`
}

func (f *JSONErrorFormatter) categoryName(category ErrorCategory) string {
    switch category {
    case CategoryValidation:
        return "validation"
    case CategoryConfiguration:
        return "configuration"
    case CategoryNetwork:
        return "network"
    case CategoryAWS:
        return "aws"
    case CategoryFileSystem:
        return "filesystem"
    case CategoryTemplate:
        return "template"
    case CategoryPermission:
        return "permission"
    case CategoryResource:
        return "resource"
    case CategoryInternal:
        return "internal"
    default:
        return "unknown"
    }
}

func (f *JSONErrorFormatter) severityName(severity ErrorSeverity) string {
    switch severity {
    case SeverityCritical:
        return "critical"
    case SeverityHigh:
        return "high"
    case SeverityMedium:
        return "medium"
    case SeverityLow:
        return "low"
    default:
        return "unknown"
    }
}
```

### Step 5: Implement Error Handling Middleware

**File**: `cmd/middleware/error_handler.go`

```go
package middleware

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/errors"
    "github.com/ArjenSchwarz/fog/cmd/registry"
    "github.com/ArjenSchwarz/fog/cmd/ui"
)

// ErrorHandlingMiddleware provides centralized error handling
type ErrorHandlingMiddleware struct {
    formatter errors.ErrorFormatter
    ui        ui.OutputHandler
    verbose   bool
}

// NewErrorHandlingMiddleware creates a new error handling middleware
func NewErrorHandlingMiddleware(formatter errors.ErrorFormatter, ui ui.OutputHandler, verbose bool) *ErrorHandlingMiddleware {
    return &ErrorHandlingMiddleware{
        formatter: formatter,
        ui:        ui,
        verbose:   verbose,
    }
}

// Execute handles errors from command execution
func (m *ErrorHandlingMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
    err := next(ctx)
    if err == nil {
        return nil
    }

    // Convert to FogError if needed
    var fogErr errors.FogError
    if fe, ok := err.(errors.FogError); ok {
        fogErr = fe
    } else {
        // Wrap unknown errors
        errorCtx := errors.GetErrorContext(ctx)
        fogErr = errors.WrapError(errorCtx, err, errors.ErrUnknown, "Unknown error occurred")
    }

    // Format and display the error
    m.displayError(fogErr)

    // Return the original error for proper exit codes
    return err
}

// displayError formats and displays an error
func (m *ErrorHandlingMiddleware) displayError(err errors.FogError) {
    // Handle multi-errors specially
    if multiErr, ok := err.(*errors.MultiError); ok {
        m.ui.Error(m.formatter.FormatMultiError(multiErr))
        return
    }

    // Format single error
    formatted := m.formatter.FormatError(err)

    // Display based on severity
    switch err.Severity() {
    case errors.SeverityCritical:
        m.ui.Error(formatted)
    case errors.SeverityHigh:
        m.ui.Error(formatted)
    case errors.SeverityMedium:
        m.ui.Warning(formatted)
    case errors.SeverityLow:
        m.ui.Info(formatted)
    default:
        m.ui.Error(formatted)
    }

    // Show debug information in verbose mode
    if m.verbose && err.StackTrace() != nil {
        m.ui.Debug("Stack trace:")
        for _, frame := range err.StackTrace() {
            m.ui.Debug("  " + frame)
        }
    }
}

// Implement registry.Middleware interface
var _ registry.Middleware = (*ErrorHandlingMiddleware)(nil)

// RecoveryMiddleware handles panics and converts them to errors
type RecoveryMiddleware struct {
    ui ui.OutputHandler
}

// NewRecoveryMiddleware creates a new recovery middleware
func NewRecoveryMiddleware(ui ui.OutputHandler) *RecoveryMiddleware {
    return &RecoveryMiddleware{
        ui: ui,
    }
}

// Execute handles panic recovery
func (m *RecoveryMiddleware) Execute(ctx context.Context, next func(context.Context) error) (err error) {
    defer func() {
        if r := recover(); r != nil {
            // Convert panic to error
            errorCtx := errors.GetErrorContext(ctx)
            panicErr := errors.ContextualError(
                errorCtx,
                errors.ErrInternal,
                fmt.Sprintf("Internal error (panic): %v", r),
            ).WithStackTrace()

            m.ui.Error(fmt.Sprintf("Internal error occurred: %v", r))
            if m.ui.GetVerbose() {
                m.ui.Debug("This is likely a bug. Please report it with the following information:")
                for _, frame := range panicErr.StackTrace() {
                    m.ui.Debug("  " + frame)
                }
            }

            err = panicErr
        }
    }()

    return next(ctx)
}

// Implement registry.Middleware interface
var _ registry.Middleware = (*RecoveryMiddleware)(nil)
```

### Step 6: Create Validation Error Helpers

**File**: `cmd/validation/errors.go`

```go
package validation

import (
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/errors"
)

// ValidationErrorBuilder helps build validation errors
type ValidationErrorBuilder struct {
    aggregator *errors.ErrorAggregator
    context    *errors.ErrorContext
}

// NewValidationErrorBuilder creates a new validation error builder
func NewValidationErrorBuilder(operation string) *ValidationErrorBuilder {
    return &ValidationErrorBuilder{
        aggregator: errors.NewErrorAggregator(operation),
        context:    errors.NewErrorContext(operation, "validation"),
    }
}

// RequiredField adds a required field error
func (b *ValidationErrorBuilder) RequiredField(fieldName string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrRequiredField,
        fmt.Sprintf("Field '%s' is required", fieldName),
    ).WithField("field_name", fieldName).
      WithUserMessage(fmt.Sprintf("Please provide a value for '%s'", fieldName)).
      WithSuggestions([]string{
          fmt.Sprintf("Add the --%s flag with a valid value", fieldName),
      })

    b.aggregator.Add(err)
    return b
}

// InvalidValue adds an invalid value error
func (b *ValidationErrorBuilder) InvalidValue(fieldName, value, reason string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrInvalidValue,
        fmt.Sprintf("Invalid value for field '%s': %s", fieldName, reason),
    ).WithField("field_name", fieldName).
      WithField("field_value", value).
      WithUserMessage(fmt.Sprintf("The value '%s' for '%s' is invalid: %s", value, fieldName, reason))

    b.aggregator.Add(err)
    return b
}

// ConflictingFlags adds a conflicting flags error
func (b *ValidationErrorBuilder) ConflictingFlags(flags []string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrConflictingFlags,
        fmt.Sprintf("Conflicting flags: %v", flags),
    ).WithField("conflicting_flags", flags).
      WithUserMessage(fmt.Sprintf("The flags %v cannot be used together", flags)).
      WithSuggestions([]string{
          "Use only one of the conflicting flags",
          "Check the command documentation for proper usage",
      })

    b.aggregator.Add(err)
    return b
}

// MissingDependency adds a missing dependency error
func (b *ValidationErrorBuilder) MissingDependency(triggerFlag string, requiredFlags []string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrDependencyMissing,
        fmt.Sprintf("Flag '%s' requires %v to be set", triggerFlag, requiredFlags),
    ).WithField("trigger_flag", triggerFlag).
      WithField("required_flags", requiredFlags).
      WithUserMessage(fmt.Sprintf("When using '%s', you must also provide %v", triggerFlag, requiredFlags))

    b.aggregator.Add(err)
    return b
}

// FileNotFound adds a file not found error
func (b *ValidationErrorBuilder) FileNotFound(fieldName, filePath string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrFileNotFound,
        fmt.Sprintf("File not found for field '%s': %s", fieldName, filePath),
    ).WithField("field_name", fieldName).
      WithField("file_path", filePath).
      WithUserMessage(fmt.Sprintf("The file '%s' specified for '%s' does not exist", filePath, fieldName)).
      WithSuggestions([]string{
          "Check that the file path is correct",
          "Ensure the file exists and is readable",
          "Use an absolute path if the relative path is not working",
      })

    b.aggregator.Add(err)
    return b
}

// InvalidFormat adds an invalid format error
func (b *ValidationErrorBuilder) InvalidFormat(fieldName, value, expectedFormat string) *ValidationErrorBuilder {
    err := errors.ContextualError(
        b.context,
        errors.ErrInvalidFormat,
        fmt.Sprintf("Invalid format for field '%s': expected %s", fieldName, expectedFormat),
    ).WithField("field_name", fieldName).
      WithField("field_value", value).
      WithField("expected_format", expectedFormat).
      WithUserMessage(fmt.Sprintf("The value '%s' for '%s' has an invalid format. Expected: %s", value, fieldName, expectedFormat))

    b.aggregator.Add(err)
    return b
}

// HasErrors returns true if there are validation errors
func (b *ValidationErrorBuilder) HasErrors() bool {
    return b.aggregator.HasErrors()
}

// ErrorCount returns the number of validation errors
func (b *ValidationErrorBuilder) ErrorCount() int {
    return b.aggregator.Count()
}

// Build returns the validation errors as a single error
func (b *ValidationErrorBuilder) Build() error {
    return b.aggregator.ToError()
}

// Errors returns the individual validation errors
func (b *ValidationErrorBuilder) Errors() []errors.FogError {
    return b.aggregator.Errors()
}
```

### Step 7: Update Deploy Command Handler with Error Handling

**File**: `cmd/commands/deploy/handler.go` (update from previous tasks)

```go
package deploy

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/errors"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/ArjenSchwarz/fog/cmd/validation"
    "github.com/ArjenSchwarz/fog/config"
)

// Handler implements the deploy command logic with enhanced error handling
type Handler struct {
    flags             *groups.DeploymentFlags
    deploymentService services.DeploymentService
    config            *config.Config
    ui                ui.OutputHandler
}

// NewHandler creates a new deploy command handler
func NewHandler(flags *groups.DeploymentFlags, deploymentService services.DeploymentService, config *config.Config, ui ui.OutputHandler) *Handler {
    return &Handler{
        flags:             flags,
        deploymentService: deploymentService,
        config:            config,
        ui:                ui,
    }
}

// Execute runs the deploy command with comprehensive error handling
func (h *Handler) Execute(ctx context.Context) error {
    // Create error context for this operation
    errorCtx := errors.NewErrorContext("deploy", "command").
        WithStackName(h.flags.StackName)

    // Add error context to Go context
    ctx = errors.WithErrorContext(ctx, errorCtx)

    // Validate flags with detailed error reporting
    if err := h.validateFlags(); err != nil {
        return err
    }

    // Convert flags to deployment options
    opts, err := h.buildDeploymentOptions()
    if err != nil {
        return errors.WrapError(errorCtx, err, errors.ErrValidationFailed, "Failed to build deployment options")
    }

    // Prepare deployment
    plan, err := h.prepareDeployment(ctx, opts)
    if err != nil {
        return err // Error already wrapped in service layer
    }

    // Validate deployment
    if err := h.validateDeployment(ctx, plan); err != nil {
        return err // Error already wrapped in service layer
    }

    // Create changeset
    changeset, err := h.createChangeset(ctx, plan)
    if err != nil {
        return err // Error already wrapped in service layer
    }

    // Handle different modes
    if opts.DryRun {
        h.ui.Success("Dry run completed successfully")
        return nil
    }

    if opts.CreateOnly {
        h.ui.Success("Changeset created successfully")
        return nil
    }

    // Execute deployment
    result, err := h.executeDeployment(ctx, plan, changeset)
    if err != nil {
        return err // Error already wrapped in service layer
    }

    if result.Success {
        h.ui.Success("Stack deployed successfully!")
    } else {
        return errors.ContextualError(
            errorCtx,
            errors.ErrDeploymentFailed,
            "Deployment completed with errors",
        ).WithDetails(result.ErrorMessage)
    }

    return nil
}

// validateFlags performs comprehensive flag validation
func (h *Handler) validateFlags() error {
    validator := validation.NewValidationErrorBuilder("flag-validation")

    // Validate required fields
    if h.flags.StackName == "" {
        validator.RequiredField("stackname")
    }

    // Validate deployment source
    if h.flags.DeploymentFile == "" && h.flags.Template == "" {
        validator.InvalidValue("deployment-source", "", "either --template or --deployment-file must be provided")
    }

    // Validate conflicting flags
    if h.flags.DeploymentFile != "" {
        conflictingFlags := []string{}
        if h.flags.Template != "" {
            conflictingFlags = append(conflictingFlags, "template")
        }
        if h.flags.Parameters != "" {
            conflictingFlags = append(conflictingFlags, "parameters")
        }
        if h.flags.Tags != "" {
            conflictingFlags = append(conflictingFlags, "tags")
        }

        if len(conflictingFlags) > 0 {
            conflictingFlags = append([]string{"deployment-file"}, conflictingFlags...)
            validator.ConflictingFlags(conflictingFlags)
        }
    }

    // Validate changeset mode conflicts
    if h.flags.CreateChangeset && h.flags.DeployChangeset {
        validator.ConflictingFlags([]string{"create-changeset", "deploy-changeset"})
    }

    // Validate file existence
    if h.flags.Template != "" {
        if _, err := os.Stat(h.flags.Template); os.IsNotExist(err) {
            validator.FileNotFound("template", h.flags.Template)
        }
    }

    if h.flags.DeploymentFile != "" {
        if _, err := os.Stat(h.flags.DeploymentFile); os.IsNotExist(err) {
            validator.FileNotFound("deployment-file", h.flags.DeploymentFile)
        }
    }

    return validator.Build()
}

// buildDeploymentOptions converts flags to deployment options with error handling
func (h *Handler) buildDeploymentOptions() (services.DeploymentOptions, error) {
    opts := services.DeploymentOptions{
        StackName:      h.flags.StackName,
        TemplateSource: h.flags.Template,
        ParameterFiles: parseCommaSeparated(h.flags.Parameters),
        TagFiles:       parseCommaSeparated(h.flags.Tags),
        DefaultTags:    h.flags.DefaultTags,
        Bucket:         h.flags.Bucket,
        ChangesetName:  h.flags.ChangesetName,
        DeploymentFile: h.flags.DeploymentFile,
        DryRun:         h.flags.Dryrun,
        NonInteractive: h.flags.NonInteractive,
        CreateOnly:     h.flags.CreateChangeset,
        DeployOnly:     h.flags.DeployChangeset,
    }

    return opts, nil
}

// prepareDeployment prepares the deployment with error context
func (h *Handler) prepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, error) {
    errorCtx := errors.GetErrorContext(ctx).WithTemplate(opts.TemplateSource)
    ctx = errors.WithErrorContext(ctx, errorCtx)

    plan, err := h.deploymentService.PrepareDeployment(ctx, opts)
    if err != nil {
        // Service layer should already wrap errors appropriately
        return nil, err
    }

    return plan, nil
}

// validateDeployment validates the deployment plan
func (h *Handler) validateDeployment(ctx context.Context, plan *services.DeploymentPlan) error {
    err := h.deploymentService.ValidateDeployment(ctx, plan)
    if err != nil {
        // Service layer should already wrap errors appropriately
        return err
    }

    return nil
}

// createChangeset creates the changeset
func (h *Handler) createChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, error) {
    changeset, err := h.deploymentService.CreateChangeset(ctx, plan)
    if err != nil {
        // Service layer should already wrap errors appropriately
        return nil, err
    }

    return changeset, nil
}

// executeDeployment executes the deployment
func (h *Handler) executeDeployment(ctx context.Context, plan *services.DeploymentPlan, changeset *services.ChangesetResult) (*services.DeploymentResult, error) {
    // Get user confirmation if needed
    if !plan.Options.NonInteractive {
        if !h.ui.Confirm("Do you want to deploy this changeset?") {
            return nil, errors.ContextualError(
                errors.GetErrorContext(ctx),
                errors.ErrUserCancelled,
                "Deployment cancelled by user",
            )
        }
    }

    result, err := h.deploymentService.ExecuteDeployment(ctx, plan, changeset)
    if err != nil {
        // Service layer should already wrap errors appropriately
        return nil, err
    }

    return result, nil
}

// ValidateFlags validates the command flags
func (h *Handler) ValidateFlags() error {
    return h.validateFlags()
}

// Helper functions

func parseCommaSeparated(input string) []string {
    if input == "" {
        return nil
    }
    // Implementation to split by comma and trim whitespace
    return []string{input} // Placeholder
}
```

## Files to Create/Modify

### New Files
- `cmd/errors/types.go`
- `cmd/errors/codes.go`
- `cmd/errors/context.go`
- `cmd/errors/formatting.go`
- `cmd/errors/aggregation.go`
- `cmd/middleware/error_handler.go`
- `cmd/middleware/recovery.go`
- `cmd/validation/errors.go`
- `cmd/validation/aggregator.go`

### Modified Files
- `cmd/commands/deploy/handler.go` - Use structured error handling
- `cmd/services/deployment/service.go` - Return structured errors
- `cmd/commands/deploy/command.go` - Add error handling middleware
- `cmd/root.go` - Setup error handling components

## Testing Strategy

### Unit Tests
- Test error type creation and manipulation
- Test error code categorization and metadata
- Test error formatting for different outputs
- Test error aggregation functionality
- Test validation error building

### Integration Tests
- Test error handling middleware integration
- Test error context propagation
- Test error formatting in real command execution
- Test recovery middleware for panic handling

### Test Files to Create
- `cmd/errors/types_test.go`
- `cmd/errors/codes_test.go`
- `cmd/errors/formatting_test.go`
- `cmd/middleware/error_handler_test.go`
- `cmd/validation/errors_test.go`

## Success Criteria

### Functional Requirements
- [ ] Structured error types with rich context
- [ ] Consistent error codes and categorization
- [ ] Comprehensive error formatting for console and JSON
- [ ] Error aggregation for validation scenarios
- [ ] Proper error middleware integration

### Quality Requirements
- [ ] Unit tests cover >90% of error handling code
- [ ] Clear and actionable error messages
- [ ] Consistent error patterns across all commands
- [ ] Performance impact is minimal

### User Experience Requirements
- [ ] User-friendly error messages with suggestions
- [ ] Proper error severity indication
- [ ] Clear validation error reporting
- [ ] Helpful debugging information in verbose mode

## Migration Timeline

### Phase 1: Foundation
- Create error type system and codes
- Implement basic error formatting
- Create error handling middleware

### Phase 2: Integration
- Update deploy command with structured errors
- Add validation error helpers
- Integrate error middleware

### Phase 3: Expansion
- Migrate remaining commands to structured errors
- Add comprehensive error metadata
- Enhance error formatting options

## Dependencies

### Upstream Dependencies
- Task 1: Command Structure Reorganization (provides middleware framework)

### Downstream Dependencies
- Task 4: Output and UI Standardization (uses structured errors for formatting)
- Task 6: Testing Infrastructure (benefits from structured errors)

## Risk Mitigation

### Potential Issues
- Performance overhead from error context tracking
- Complexity in error type management
- Breaking changes to existing error handling

### Mitigation Strategies
- Minimize error context overhead with lazy evaluation
- Clear documentation of error types and usage
- Gradual migration with backward compatibility
- Comprehensive testing of error scenarios
