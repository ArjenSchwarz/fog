package flags

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// BaseFlagValidator provides common flag validation functionality.
type BaseFlagValidator struct {
	rules         []ValidationRule
	groups        []FlagGroup
	preprocessors []FlagPreprocessor
}

// NewBaseFlagValidator creates a new base flag validator.
func NewBaseFlagValidator() *BaseFlagValidator {
	return &BaseFlagValidator{
		rules:         make([]ValidationRule, 0),
		groups:        make([]FlagGroup, 0),
		preprocessors: make([]FlagPreprocessor, 0),
	}
}

// AddRule adds a validation rule.
func (b *BaseFlagValidator) AddRule(rule ValidationRule) {
	b.rules = append(b.rules, rule)
}

// AddGroup adds a flag group and its rules.
func (b *BaseFlagValidator) AddGroup(group FlagGroup) {
	b.groups = append(b.groups, group)
	for _, rule := range group.GetValidationRules() {
		b.AddRule(rule)
	}
}

// AddPreprocessor adds a flag preprocessor.
func (b *BaseFlagValidator) AddPreprocessor(preprocessor FlagPreprocessor) {
	b.preprocessors = append(b.preprocessors, preprocessor)
}

// Validate validates all flags and rules.
func (b *BaseFlagValidator) Validate(ctx context.Context) error {
	for _, preprocessor := range b.preprocessors {
		if err := preprocessor.Process(ctx, b); err != nil {
			return fmt.Errorf("flag preprocessing failed: %w", err)
		}
	}

	var errors []error
	var warnings []string

	for _, rule := range b.rules {
		if err := rule.Validate(ctx, b); err != nil {
			switch rule.GetSeverity() {
			case SeverityError:
				errors = append(errors, err)
			case SeverityWarning:
				warnings = append(warnings, err.Error())
			case SeverityInfo:
				fmt.Printf("Info: %s\n", err.Error())
			}
		}
	}

	for _, warning := range warnings {
		fmt.Printf("Warning: %s\n", warning)
	}

	if len(errors) > 0 {
		return fmt.Errorf("flag validation failed: %w", errors[0])
	}

	return nil
}

// RegisterFlags registers all flags from groups.
func (b *BaseFlagValidator) RegisterFlags(cmd *cobra.Command) {
	for _, group := range b.groups {
		group.RegisterFlags(cmd)
	}
}

// GetValidationRules returns all validation rules.
func (b *BaseFlagValidator) GetValidationRules() []ValidationRule {
	return b.rules
}
