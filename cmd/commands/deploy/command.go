package deploy

import (
	"github.com/ArjenSchwarz/fog/cmd/flags/groups"
	middleware "github.com/ArjenSchwarz/fog/cmd/flags/middleware"
	"github.com/ArjenSchwarz/fog/cmd/registry"
	services "github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/cobra"
)

// CommandBuilder constructs the deploy command using the BaseCommandBuilder.
type CommandBuilder struct {
	*registry.BaseCommandBuilder
	flags *groups.DeploymentFlags
}

// NewCommandBuilder creates a new deploy command builder.
// NewCommandBuilder creates a new deploy command builder with injected services.
func NewCommandBuilder(factory services.ServiceFactory, middlewares ...registry.Middleware) *CommandBuilder {
	flagGroup := NewFlags()
	builder := registry.NewBaseCommandBuilder(
		"deploy",
		"Deploy a CloudFormation stack",
		`deploy allows you to deploy a CloudFormation stack

It does so by creating a ChangeSet and then asking you for approval before continuing. You can automatically approve or only create or deploy a changeset by using flags.

A name for the changeset will automatically be generated based on your preferred name, but can be overwritten as well.

When providing tag and/or parameter files, you can add multiple files for each. These are parsed in the order provided and later values will override earlier ones.
`,
	)
	var cfg *config.Config
	if cp, ok := factory.(services.ConfigProvider); ok {
		cfg = cp.AppConfig()
	}
	handler := NewHandler(flagGroup.DeploymentFlags, factory.CreateDeploymentService(), cfg)
	flagValidationMiddleware := middleware.NewFlagValidationMiddleware(flagGroup.DeploymentFlags)

	base := builder.WithHandler(handler).
		WithValidator(flagGroup).
		WithMiddleware(flagValidationMiddleware)
	for _, mw := range middlewares {
		base = base.WithMiddleware(mw)
	}

	return &CommandBuilder{
		BaseCommandBuilder: base,
		flags:              flagGroup.DeploymentFlags,
	}
}

// BuildCommand creates the cobra command.
func (b *CommandBuilder) BuildCommand() *cobra.Command {
	return b.BaseCommandBuilder.BuildCommand()
}

// GetHandler returns the command handler associated with the builder.
func (b *CommandBuilder) GetHandler() registry.CommandHandler {
	return b.BaseCommandBuilder.GetHandler()
}
