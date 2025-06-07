package deployment

import (
	"context"

	ferr "github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/config"
)

// Service implements the services.DeploymentService interface.
type Service struct {
	templateService  services.TemplateService
	parameterService services.ParameterService
	tagService       services.TagService
	cfnClient        services.CloudFormationClient
	s3Client         services.S3Client
	config           *config.Config
}

// NewService creates a new deployment service with its dependencies.
func NewService(tmpl services.TemplateService, params services.ParameterService, tags services.TagService, cfn services.CloudFormationClient, s3 services.S3Client, cfg *config.Config) *Service {
	return &Service{
		templateService:  tmpl,
		parameterService: params,
		tagService:       tags,
		cfnClient:        cfn,
		s3Client:         s3,
		config:           cfg,
	}
}

// PrepareDeployment builds a DeploymentPlan from the given options.
// This placeholder only fills a few fields and performs no AWS calls.
func (s *Service) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, ferr.FogError) {
	plan := &services.DeploymentPlan{
		StackName:     opts.StackName,
		Options:       opts,
		ChangesetName: opts.ChangesetName,
	}

	tmpl, err := s.templateService.LoadTemplate(ctx, opts.TemplateSource)
	if err != nil {
		return nil, err
	}
	plan.Template = tmpl

	params, err := s.parameterService.LoadParameters(ctx, opts.ParameterFiles)
	if err != nil {
		return nil, err
	}
	plan.Parameters = params

	tags, err := s.tagService.LoadTags(ctx, opts.TagFiles, map[string]string{})
	if err != nil {
		return nil, err
	}
	plan.Tags = tags

	if plan.ChangesetName == "" {
		plan.ChangesetName = "fog-changeset"
	}

	return plan, nil
}

// ValidateDeployment performs basic validation of the deployment plan.
func (s *Service) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) ferr.FogError {
	if err := s.templateService.ValidateTemplate(ctx, plan.Template); err != nil {
		return err
	}
	if err := s.parameterService.ValidateParameters(ctx, plan.Parameters, plan.Template); err != nil {
		return err
	}
	if err := s.tagService.ValidateTags(ctx, plan.Tags); err != nil {
		return err
	}
	if err := s.validateStackState(ctx, plan); err != nil {
		return err
	}
	return nil
}

// CreateChangeset creates a CloudFormation changeset.
// Placeholder implementation returning a not implemented error.
func (s *Service) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, ferr.FogError) {
	return s.createChangeSet(ctx, plan)
}

// ExecuteDeployment executes the previously created changeset.
// Placeholder implementation returning a not implemented error.
func (s *Service) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, changeset *services.ChangesetResult) (*services.DeploymentResult, ferr.FogError) {
	errorCtx := ferr.GetErrorContext(ctx)
	return nil, ferr.ContextualError(errorCtx, ferr.ErrNotImplemented, "deployment execution not implemented")
}
