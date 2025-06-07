package deployment

import (
	"context"
	"fmt"

	"github.com/ArjenSchwarz/fog/cmd/services"
)

// createChangeSet is a helper used by Service.CreateChangeset.
func (s *Service) createChangeSet(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, error) {
	_ = ctx
	_ = plan
	return nil, fmt.Errorf("changeset logic not implemented")
}
