package registry

import (
	"context"

	"github.com/spf13/cobra"
)

// ValidationSeverity represents the severity level of a validation rule
type ValidationSeverity int

const (
	ValidationSeverityError ValidationSeverity = iota
	ValidationSeverityWarning
	ValidationSeverityInfo
)

// ValidationContext provides context information for validation operations
type ValidationContext struct {
	Command    *cobra.Command
	Args       []string
	AWSRegion  string
	ConfigPath string
	Verbose    bool
}

// ValidationRule defines a single validation rule
type ValidationRule interface {
	Validate(ctx context.Context, vCtx *ValidationContext) error
	GetDescription() string
	GetSeverity() ValidationSeverity
}

// FlagGroup represents a logical grouping of related flags
type FlagGroup interface {
	GetName() string
	GetDescription() string
	RegisterFlags(cmd *cobra.Command)
	GetValidationRules() []ValidationRule
}

// FlagPreprocessor handles preprocessing of flag values before validation
type FlagPreprocessor interface {
	Preprocess(ctx context.Context, vCtx *ValidationContext) error
	GetDescription() string
}

// Middleware defines the interface for command middleware
type Middleware interface {
	Execute(ctx context.Context, next func(context.Context) error) error
	GetName() string
}

// CommandHandler defines the interface for command business logic.
type CommandHandler interface {
	Execute(ctx context.Context) error
	ValidateFlags() error
}

// CommandBuilder describes how to build a cobra command.
type CommandBuilder interface {
	BuildCommand() *cobra.Command
	GetHandler() CommandHandler
}

// FlagValidator defines validation and flag registration behaviour.
type FlagValidator interface {
	Validate(ctx context.Context, vCtx *ValidationContext) error
	RegisterFlags(cmd *cobra.Command)
	GetValidationRules() []ValidationRule
}
