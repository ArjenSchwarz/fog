package deployment

import (
	"context"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
)

// validateStackState checks if a stack can be updated. Placeholder implementation.
func (s *Service) validateStackState(ctx context.Context, plan *services.DeploymentPlan) ferr.FogError {
	_ = plan
	_ = ctx
	return nil
}
