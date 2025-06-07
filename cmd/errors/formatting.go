package errors

import (
	"fmt"
	"strings"
	"time"
)

// ErrorCode represents a specific error condition.
type ErrorCode string

// ErrorCategory is the classification of an error.
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

// ErrorSeverity indicates how severe an error is.
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// FogError represents a structured error with context information.
type FogError interface {
	error

	Code() ErrorCode
	Message() string
	Details() string

	Operation() string
	Component() string
	Timestamp() time.Time

	Category() ErrorCategory
	Severity() ErrorSeverity
	Retryable() bool

	StackTrace() []string
	Cause() error

	UserMessage() string
	Suggestions() []string

	Fields() map[string]interface{}
	WithField(key string, value interface{}) FogError
	WithFields(fields map[string]interface{}) FogError
}

// MultiError is a collection of FogErrors.
type MultiError struct {
	errors []FogError
}

// Errors returns the contained errors.
func (m *MultiError) Errors() []FogError { return m.errors }

// ErrorFormatter handles error formatting for different outputs.
type ErrorFormatter interface {
	FormatError(err FogError) string
	FormatMultiError(err *MultiError) string
	FormatValidationErrors(errors []FogError) string
}

// ConsoleErrorFormatter formats errors for console output.
type ConsoleErrorFormatter struct {
	colorEnabled bool
	verbose      bool
}

// NewConsoleErrorFormatter creates a new console error formatter.
func NewConsoleErrorFormatter(colorEnabled, verbose bool) *ConsoleErrorFormatter {
	return &ConsoleErrorFormatter{colorEnabled: colorEnabled, verbose: verbose}
}

// FormatError formats a single error for console output.
func (f *ConsoleErrorFormatter) FormatError(err FogError) string {
	var builder strings.Builder

	severity := f.formatSeverity(err.Severity())
	builder.WriteString(fmt.Sprintf("%s [%s] %s\n", severity, err.Code(), err.Message()))

	if details := err.Details(); details != "" {
		builder.WriteString(fmt.Sprintf("Details: %s\n", details))
	}

	if f.verbose {
		builder.WriteString(f.formatContext(err))
	}

	if suggestions := err.Suggestions(); len(suggestions) > 0 {
		builder.WriteString("Suggestions:\n")
		for _, s := range suggestions {
			builder.WriteString(fmt.Sprintf("  • %s\n", s))
		}
	}

	if cause := err.Cause(); cause != nil && f.verbose {
		builder.WriteString(fmt.Sprintf("Caused by: %s\n", cause.Error()))
	}

	if f.verbose {
		if stack := err.StackTrace(); len(stack) > 0 {
			builder.WriteString("Stack trace:\n")
			for _, frame := range stack {
				builder.WriteString(fmt.Sprintf("  %s\n", frame))
			}
		}
	}

	return builder.String()
}

// FormatMultiError formats multiple errors.
func (f *ConsoleErrorFormatter) FormatMultiError(err *MultiError) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("Multiple errors occurred (%d total):\n\n", len(err.Errors())))

	for i, sub := range err.Errors() {
		builder.WriteString(fmt.Sprintf("%d. %s", i+1, f.FormatError(sub)))
		if i < len(err.Errors())-1 {
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

// FormatValidationErrors formats validation errors in a user-friendly way.
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

func (f *ConsoleErrorFormatter) formatSeverity(severity ErrorSeverity) string {
	if !f.colorEnabled {
		return f.severityText(severity)
	}

	switch severity {
	case SeverityCritical:
		return fmt.Sprintf("\033[91m%s\033[0m", f.severityText(severity))
	case SeverityHigh:
		return fmt.Sprintf("\033[93m%s\033[0m", f.severityText(severity))
	case SeverityMedium:
		return fmt.Sprintf("\033[94m%s\033[0m", f.severityText(severity))
	default:
		return f.severityText(severity)
	}
}

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

func (f *ConsoleErrorFormatter) formatContext(err FogError) string {
	var builder strings.Builder

	builder.WriteString("Context:\n")

	if op := err.Operation(); op != "" {
		builder.WriteString(fmt.Sprintf("  Operation: %s\n", op))
	}

	if comp := err.Component(); comp != "" {
		builder.WriteString(fmt.Sprintf("  Component: %s\n", comp))
	}

	builder.WriteString(fmt.Sprintf("  Timestamp: %s\n", err.Timestamp().Format(time.RFC3339)))

	if fields := err.Fields(); len(fields) > 0 {
		builder.WriteString("  Fields:\n")
		for k, v := range fields {
			builder.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
		}
	}

	return builder.String()
}

// JSONErrorFormatter formats errors as JSON.
type JSONErrorFormatter struct{}

// NewJSONErrorFormatter creates a new JSON error formatter.
func NewJSONErrorFormatter() *JSONErrorFormatter { return &JSONErrorFormatter{} }

// FormatError formats an error as JSON.
func (f *JSONErrorFormatter) FormatError(err FogError) string {
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
}`,
		err.Code(), err.Message(), f.categoryName(err.Category()),
		f.severityName(err.Severity()), err.Timestamp().Format(time.RFC3339),
		err.Operation(), err.Component(),
	)
}

// FormatMultiError formats multiple errors as JSON.
func (f *JSONErrorFormatter) FormatMultiError(err *MultiError) string {
	return `{"errors": []}`
}

// FormatValidationErrors formats validation errors as JSON.
func (f *JSONErrorFormatter) FormatValidationErrors(errors []FogError) string {
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
