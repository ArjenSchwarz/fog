package registry

import (
	"context"

	"github.com/spf13/cobra"
)

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
	Validate() error
	RegisterFlags(cmd *cobra.Command)
}
