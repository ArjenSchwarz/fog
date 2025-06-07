package validation

import (
	"fmt"

	"github.com/ArjenSchwarz/fog/cmd/errors"
)

// ValidationErrorBuilder helps build validation errors.
type ValidationErrorBuilder struct {
	aggregator *errors.ErrorAggregator
	context    *errors.ErrorContext
}

// NewValidationErrorBuilder creates a new validation error builder.
func NewValidationErrorBuilder(operation string) *ValidationErrorBuilder {
	return &ValidationErrorBuilder{
		aggregator: errors.NewErrorAggregator(operation),
		context:    errors.NewErrorContext(operation, "validation"),
	}
}

// RequiredField adds a required field error.
func (b *ValidationErrorBuilder) RequiredField(fieldName string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrRequiredField,
		fmt.Sprintf("Field '%s' is required", fieldName),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("Please provide a value for '%s'", fieldName)).
		WithSuggestions([]string{
			fmt.Sprintf("Add the --%s flag with a valid value", fieldName),
		})

	// Add context fields and specific field
	contextFields := b.context.ToMap()
	contextFields["field_name"] = fieldName
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// InvalidValue adds an invalid value error.
func (b *ValidationErrorBuilder) InvalidValue(fieldName, value, reason string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrInvalidValue,
		fmt.Sprintf("Invalid value for field '%s': %s", fieldName, reason),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("The value '%s' for '%s' is invalid: %s", value, fieldName, reason))

	// Add context fields and specific fields
	contextFields := b.context.ToMap()
	contextFields["field_name"] = fieldName
	contextFields["field_value"] = value
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// ConflictingFlags adds a conflicting flags error.
func (b *ValidationErrorBuilder) ConflictingFlags(flags []string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrConflictingFlags,
		fmt.Sprintf("Conflicting flags: %v", flags),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("The flags %v cannot be used together", flags)).
		WithSuggestions([]string{
			"Use only one of the conflicting flags",
			"Check the command documentation for proper usage",
		})

	// Add context fields and specific field
	contextFields := b.context.ToMap()
	contextFields["conflicting_flags"] = flags
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// MissingDependency adds a missing dependency error.
func (b *ValidationErrorBuilder) MissingDependency(triggerFlag string, requiredFlags []string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrDependencyMissing,
		fmt.Sprintf("Flag '%s' requires %v to be set", triggerFlag, requiredFlags),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("When using '%s', you must also provide %v", triggerFlag, requiredFlags))

	// Add context fields and specific fields
	contextFields := b.context.ToMap()
	contextFields["trigger_flag"] = triggerFlag
	contextFields["required_flags"] = requiredFlags
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// FileNotFound adds a file not found error.
func (b *ValidationErrorBuilder) FileNotFound(fieldName, filePath string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrFileNotFound,
		fmt.Sprintf("File not found for field '%s': %s", fieldName, filePath),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("The file '%s' specified for '%s' does not exist", filePath, fieldName)).
		WithSuggestions([]string{
			"Check that the file path is correct",
			"Ensure the file exists and is readable",
			"Use an absolute path if the relative path is not working",
		})

	// Add context fields and specific fields
	contextFields := b.context.ToMap()
	contextFields["field_name"] = fieldName
	contextFields["file_path"] = filePath
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// InvalidFormat adds an invalid format error.
func (b *ValidationErrorBuilder) InvalidFormat(fieldName, value, expectedFormat string) *ValidationErrorBuilder {
	baseErr := errors.NewError(
		errors.ErrInvalidFormat,
		fmt.Sprintf("Invalid format for field '%s': expected %s", fieldName, expectedFormat),
	).WithOperation(b.context.Operation).
		WithComponent(b.context.Component).
		WithUserMessage(fmt.Sprintf("The value '%s' for '%s' has an invalid format. Expected: %s", value, fieldName, expectedFormat))

	// Add context fields and specific fields
	contextFields := b.context.ToMap()
	contextFields["field_name"] = fieldName
	contextFields["field_value"] = value
	contextFields["expected_format"] = expectedFormat
	err := baseErr.WithFields(contextFields)

	b.aggregator.Add(err)
	return b
}

// HasErrors returns true if there are validation errors.
func (b *ValidationErrorBuilder) HasErrors() bool {
	return b.aggregator.HasErrors()
}

// ErrorCount returns the number of validation errors.
func (b *ValidationErrorBuilder) ErrorCount() int {
	return b.aggregator.Count()
}

// Build returns the validation errors as a single error.
func (b *ValidationErrorBuilder) Build() error {
	return b.aggregator.ToError()
}

// Errors returns the individual validation errors.
func (b *ValidationErrorBuilder) Errors() []errors.FogError {
	return b.aggregator.Errors()
}
