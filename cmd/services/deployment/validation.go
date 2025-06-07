package deployment

import (
	"context"

	"github.com/ArjenSchwarz/fog/cmd/services"
)

// validateStackState checks if a stack can be updated. Placeholder implementation.
func (s *Service) validateStackState(ctx context.Context, plan *services.DeploymentPlan) error {
	_ = ctx
	_ = plan
	return nil
}
