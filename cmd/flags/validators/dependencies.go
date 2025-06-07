package validators

import (
	"context"
	"fmt"

	"github.com/ArjenSchwarz/fog/cmd/flags"
)

// DependencyRule validates that when one flag is set, dependent flags are also set
type DependencyRule struct {
	TriggerField    string
	DependentFields []string
	GetTrigger      func(flags.FlagValidator) interface{}
	GetDependent    func(flags.FlagValidator, string) interface{}
	Description     string
}

// NewDependencyRule creates a new dependency rule
func NewDependencyRule(triggerField string, dependentFields []string, getTrigger func(flags.FlagValidator) interface{}, getDependent func(flags.FlagValidator, string) interface{}) *DependencyRule {
	return &DependencyRule{
		TriggerField:    triggerField,
		DependentFields: dependentFields,
		GetTrigger:      getTrigger,
		GetDependent:    getDependent,
		Description:     fmt.Sprintf("when %s is set, %v must also be set", triggerField, dependentFields),
	}
}

// Validate checks dependencies
func (d *DependencyRule) Validate(ctx context.Context, flags flags.FlagValidator, vCtx *flags.ValidationContext) error {
	triggerValue := d.GetTrigger(flags)
	if isEmpty(triggerValue) {
		return nil
	}

	for _, depField := range d.DependentFields {
		depValue := d.GetDependent(flags, depField)
		if isEmpty(depValue) {
			return fmt.Errorf("when '%s' is specified, '%s' must also be provided", d.TriggerField, depField)
		}
	}
	return nil
}

// GetDescription returns the rule description
func (d *DependencyRule) GetDescription() string { return d.Description }

// GetSeverity returns the rule severity
func (d *DependencyRule) GetSeverity() flags.ValidationSeverity { return flags.SeverityError }
