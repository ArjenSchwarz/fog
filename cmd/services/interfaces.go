package services

import (
	"context"

	"github.com/ArjenSchwarz/fog/cmd/errors"
	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// DeploymentService handles stack deployment operations
// Methods mirror the high level deployment steps.
type DeploymentService interface {
	PrepareDeployment(ctx context.Context, opts DeploymentOptions) (*DeploymentPlan, errors.FogError)
	ValidateDeployment(ctx context.Context, plan *DeploymentPlan) errors.FogError
	CreateChangeset(ctx context.Context, plan *DeploymentPlan) (*ChangesetResult, errors.FogError)
	ExecuteDeployment(ctx context.Context, plan *DeploymentPlan, changeset *ChangesetResult) (*DeploymentResult, errors.FogError)
}

// DriftService handles drift detection operations
// Placeholder for future implementation.
type DriftService interface {
	DetectDrift(ctx context.Context, stackName string, opts DriftOptions) (*DriftResult, error)
	AnalyzeDrift(ctx context.Context, result *DriftResult) (*DriftAnalysis, error)
}

// StackService handles general stack operations
// Placeholder for future implementation.
type StackService interface {
	DescribeStack(ctx context.Context, stackName string) (*StackDescription, error)
	ListResources(ctx context.Context, stackName string) (*ResourceList, error)
	GetHistory(ctx context.Context, stackName string, opts HistoryOptions) (*StackHistory, error)
}

// TemplateService handles template operations
// It abstracts loading, validating and uploading templates.
type TemplateService interface {
	LoadTemplate(ctx context.Context, templatePath string) (*Template, errors.FogError)
	ValidateTemplate(ctx context.Context, template *Template) errors.FogError
	UploadTemplate(ctx context.Context, template *Template, bucket string) (*TemplateReference, errors.FogError)
}

// ParameterService handles parameter operations.
type ParameterService interface {
	LoadParameters(ctx context.Context, parameterFiles []string) ([]cfnTypes.Parameter, errors.FogError)
	ValidateParameters(ctx context.Context, parameters []cfnTypes.Parameter, template *Template) errors.FogError
}

// TagService handles tag operations.
type TagService interface {
	LoadTags(ctx context.Context, tagFiles []string, defaults map[string]string) ([]cfnTypes.Tag, errors.FogError)
	ValidateTags(ctx context.Context, tags []cfnTypes.Tag) errors.FogError
}

// CloudFormationClient abstracts AWS CloudFormation operations used by services.
type CloudFormationClient interface {
	DescribeStacks(ctx context.Context, input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
	CreateChangeSet(ctx context.Context, input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error)
	ExecuteChangeSet(ctx context.Context, input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error)
	DescribeChangeSet(ctx context.Context, input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error)
}

// S3Client abstracts AWS S3 operations used by services.
type S3Client interface {
	PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}

// ServiceFactory creates service instances with proper dependencies.
type ServiceFactory interface {
	CreateDeploymentService() DeploymentService
	CreateDriftService() DriftService
	CreateStackService() StackService
}

// ConfigProvider represents something that can return application and AWS config.
type ConfigProvider interface {
	AppConfig() *config.Config
	AWSConfig() *config.AWSConfig
}

// These types from lib package are referenced in DeploymentPlan.
var (
	_ = lib.DeployInfo{}
)
