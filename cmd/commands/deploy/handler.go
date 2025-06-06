package deploy

import (
	"context"
	"fmt"
)

// Handler implements the deploy command logic.
type Handler struct {
	flags *Flags
}

// NewHandler creates a new deploy command handler.
func NewHandler(flags *Flags) *Handler {
	return &Handler{flags: flags}
}

// Execute runs the deploy command.
// The actual business logic will be implemented in a later refactor task.
func (h *Handler) Execute(ctx context.Context) error {
	// Retrieve command and args from context (for future use)
	_ = ctx.Value("command")
	_ = ctx.Value("args")
	return fmt.Errorf("deploy handler not yet implemented - waiting for Task 2")
}

// ValidateFlags validates the command flags using the Flags struct.
func (h *Handler) ValidateFlags() error {
	if h.flags == nil {
		return fmt.Errorf("no flags provided")
	}
	return h.flags.Validate()
}
