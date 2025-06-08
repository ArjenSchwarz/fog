package deploy

import (
	"context"

	cmdflags "github.com/ArjenSchwarz/fog/cmd/flags"
	"github.com/ArjenSchwarz/fog/cmd/flags/groups"
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
func (f *Flags) Validate() error {
	return f.DeploymentFlags.Validate(context.Background(), &cmdflags.ValidationContext{})
}

// RegisterFlags registers the deployment flags on the command.
func (f *Flags) RegisterFlags(cmd *cobra.Command) { f.DeploymentFlags.RegisterFlags(cmd) }
