package deployment

import (
	"context"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// ParameterService implements services.ParameterService with placeholder logic.
type ParameterService struct{}

// NewParameterService creates a new ParameterService.
func NewParameterService() *ParameterService { return &ParameterService{} }

// LoadParameters loads parameters from files. Placeholder implementation.
func (p *ParameterService) LoadParameters(ctx context.Context, parameterFiles []string) ([]cfnTypes.Parameter, ferr.FogError) {
	_ = ctx
	// Real implementation would read parameter files
	return []cfnTypes.Parameter{}, nil
}

// ValidateParameters validates parameter values against a template. Placeholder.
func (p *ParameterService) ValidateParameters(ctx context.Context, parameters []cfnTypes.Parameter, template *services.Template) ferr.FogError {
	_ = ctx
	_ = template
	_ = parameters
	return nil
}
