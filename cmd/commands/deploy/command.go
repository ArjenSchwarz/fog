package deploy

import (
	"github.com/ArjenSchwarz/fog/cmd/registry"
	"github.com/spf13/cobra"
)

// CommandBuilder constructs the deploy command using the BaseCommandBuilder.
type CommandBuilder struct {
	*registry.BaseCommandBuilder
	flags *Flags
}

// NewCommandBuilder creates a new deploy command builder.
func NewCommandBuilder() *CommandBuilder {
	flags := &Flags{}
	builder := registry.NewBaseCommandBuilder(
		"deploy",
		"Deploy a CloudFormation stack",
		`deploy allows you to deploy a CloudFormation stack

It does so by creating a ChangeSet and then asking you for approval before continuing. You can automatically approve or only create or deploy a changeset by using flags.

A name for the changeset will automatically be generated based on your preferred name, but can be overwritten as well.

When providing tag and/or parameter files, you can add multiple files for each. These are parsed in the order provided and later values will override earlier ones.
`,
	)

	handler := NewHandler(flags)

	return &CommandBuilder{
		BaseCommandBuilder: builder.WithHandler(handler).WithValidator(flags),
		flags:              flags,
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
