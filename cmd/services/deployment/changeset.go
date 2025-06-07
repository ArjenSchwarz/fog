package deployment

import (
	"context"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
)

// createChangeSet is a helper used by Service.CreateChangeset.
func (s *Service) createChangeSet(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, ferr.FogError) {
	_ = plan
	errorCtx := ferr.GetErrorContext(ctx)
	return nil, ferr.ContextualError(errorCtx, ferr.ErrNotImplemented, "changeset logic not implemented")
}
