package flags

import (
	"context"
	"strings"

	"github.com/spf13/cobra"
)

// FlagValidator defines the interface for flag validation
// Validate should process all registered validation rules and return an error
// if any rules of severity Error fail.
type FlagValidator interface {
	Validate(ctx context.Context, vCtx *ValidationContext) error
	RegisterFlags(cmd *cobra.Command)
	GetValidationRules() []ValidationRule
}

// ValidationRule defines a single validation rule used by a FlagValidator
// implementation.
type ValidationRule interface {
	Validate(ctx context.Context, flags FlagValidator, vCtx *ValidationContext) error
	GetDescription() string
	GetSeverity() ValidationSeverity
}

// ValidationSeverity indicates the importance of a validation rule.
type ValidationSeverity int

const (
	SeverityError ValidationSeverity = iota
	SeverityWarning
	SeverityInfo
)

// FlagGroup represents a logical group of related flags.
type FlagGroup interface {
	GetName() string
	GetFlags() []FlagDefinition
	GetValidationRules() []ValidationRule
	RegisterFlags(cmd *cobra.Command)
}

// FlagDefinition defines a single flag within a group.
type FlagDefinition struct {
	Name         string
	Shorthand    string
	Description  string
	DefaultValue interface{}
	Required     bool
	Hidden       bool
	Deprecated   bool
	FlagType     FlagType
}

// FlagType represents the type of flag.
type FlagType int

const (
	StringFlag FlagType = iota
	StringSliceFlag
	BoolFlag
	IntFlag
	DurationFlag
)

// ValidationContext provides contextual data for validation.
type ValidationContext struct {
	Command    *cobra.Command
	Args       []string
	AWSRegion  string
	ConfigPath string
	Verbose    bool
}

// FlagPreprocessor handles flag preprocessing before validation occurs.
type FlagPreprocessor interface {
	Process(ctx context.Context, flags FlagValidator, vCtx *ValidationContext) error
}

// ValidationError aggregates all validation errors, warnings, and informational
// messages encountered during flag validation.
type ValidationError struct {
	Err      error
	Warnings []string
	Infos    []string
}

// Error implements the error interface.
func (v *ValidationError) Error() string {
	messages := make([]string, 0)
	if v.Err != nil {
		messages = append(messages, v.Err.Error())
	}
	for _, w := range v.Warnings {
		messages = append(messages, "warning: "+w)
	}
	for _, i := range v.Infos {
		messages = append(messages, "info: "+i)
	}
	if len(messages) == 0 {
		return ""
	}
	return strings.Join(messages, "; ")
}

// Unwrap returns the underlying aggregated error, enabling errors.Is/As
// behaviour.
func (v *ValidationError) Unwrap() error {
	return v.Err
}
