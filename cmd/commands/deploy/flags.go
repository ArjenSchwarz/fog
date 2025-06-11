package deploy

import (
	"context"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
	"github.com/ArjenSchwarz/fog/cmd/flags/groups"
	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/spf13/cobra"
)

// Flags wraps groups.DeploymentFlags and adapts it to the registry.FlagValidator interface.
// Deprecated: use groups.DeploymentFlags directly after the migration completes.
type Flags struct {
	*groups.DeploymentFlags
}

// NewFlags creates a new wrapped deployment flag group.
func NewFlags() *Flags { return &Flags{groups.NewDeploymentFlags()} }

// Validate runs the underlying flag validation logic.
func (f *Flags) Validate(ctx context.Context, vCtx *registry.ValidationContext) error {
	return f.DeploymentFlags.Validate(ctx, &cmdflags.ValidationContext{
		Command:    vCtx.Command,
		Args:       vCtx.Args,
		AWSRegion:  vCtx.AWSRegion,
		ConfigPath: vCtx.ConfigPath,
		Verbose:    vCtx.Verbose,
	})
}

// RegisterFlags registers the deployment flags on the command.
func (f *Flags) RegisterFlags(cmd *cobra.Command) { f.DeploymentFlags.RegisterFlags(cmd) }

// GetValidationRules returns the validation rules for the deployment flags.
func (f *Flags) GetValidationRules() []registry.ValidationRule {
	// Convert flags.ValidationRule to registry.ValidationRule
	rules := f.DeploymentFlags.GetValidationRules()
	registryRules := make([]registry.ValidationRule, len(rules))
	for i, rule := range rules {
		registryRules[i] = &validationRuleAdapter{rule}
	}
	return registryRules
}

// validationRuleAdapter adapts flags.ValidationRule to registry.ValidationRule
type validationRuleAdapter struct {
	rule cmdflags.ValidationRule
}

func (a *validationRuleAdapter) Validate(ctx context.Context, vCtx *registry.ValidationContext) error {
	flagsCtx := &cmdflags.ValidationContext{
		Command:    vCtx.Command,
		Args:       vCtx.Args,
		AWSRegion:  vCtx.AWSRegion,
		ConfigPath: vCtx.ConfigPath,
		Verbose:    vCtx.Verbose,
	}
	// We need a FlagValidator, but we only have a ValidationRule
	// For now, pass nil - this needs a proper implementation
	return a.rule.Validate(ctx, nil, flagsCtx)
}

func (a *validationRuleAdapter) GetDescription() string {
	return a.rule.GetDescription()
}

func (a *validationRuleAdapter) GetSeverity() registry.ValidationSeverity {
	switch a.rule.GetSeverity() {
	case cmdflags.SeverityError:
		return registry.ValidationSeverityError
	case cmdflags.SeverityWarning:
		return registry.ValidationSeverityWarning
	case cmdflags.SeverityInfo:
		return registry.ValidationSeverityInfo
	default:
		return registry.ValidationSeverityError
	}
}
