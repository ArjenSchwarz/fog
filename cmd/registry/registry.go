package registry

import (
	"fmt"

	"github.com/spf13/cobra"
)

// CommandRegistry manages registration of command builders.
type CommandRegistry struct {
	builders map[string]CommandBuilder
	rootCmd  *cobra.Command
}

// NewCommandRegistry creates a new registry associated with the provided root command.
func NewCommandRegistry(rootCmd *cobra.Command) *CommandRegistry {
	return &CommandRegistry{
		builders: make(map[string]CommandBuilder),
		rootCmd:  rootCmd,
	}
}

// Register adds a builder to the registry.
func (r *CommandRegistry) Register(name string, builder CommandBuilder) error {
	if _, exists := r.builders[name]; exists {
		return fmt.Errorf("command %s already registered", name)
	}
	r.builders[name] = builder
	return nil
}

// BuildAll constructs all registered commands and attaches them to the root.
func (r *CommandRegistry) BuildAll() error {
	for name, builder := range r.builders {
		cmd := builder.BuildCommand()
		if cmd == nil {
			return fmt.Errorf("builder for %s returned nil command", name)
		}
		r.rootCmd.AddCommand(cmd)
	}
	return nil
}
