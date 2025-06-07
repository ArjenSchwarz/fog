package validators

import (
	"context"
	"fmt"

	"github.com/ArjenSchwarz/fog/cmd/flags"
)

// ConflictRule validates that conflicting flags are not used together
type ConflictRule struct {
	ConflictingFields []string
	GetValue          func(flags.FlagValidator, string) interface{}
	Description       string
}

// NewConflictRule creates a new conflict rule
func NewConflictRule(conflictingFields []string, getValue func(flags.FlagValidator, string) interface{}) *ConflictRule {
	return &ConflictRule{
		ConflictingFields: conflictingFields,
		GetValue:          getValue,
		Description:       fmt.Sprintf("flags %v cannot be used together", conflictingFields),
	}
}

// Validate checks for conflicts
func (c *ConflictRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	setFields := make([]string, 0)
	for _, field := range c.ConflictingFields {
		value := c.GetValue(flags, field)
		if !isEmpty(value) {
			setFields = append(setFields, field)
		}
	}
	if len(setFields) > 1 {
		return fmt.Errorf("conflicting flags specified: %v (only one allowed)", setFields)
	}
	return nil
}

// GetDescription returns the rule description
func (c *ConflictRule) GetDescription() string { return c.Description }

// GetSeverity returns the rule severity
func (c *ConflictRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }
