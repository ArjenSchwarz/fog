package factory

import (
	"github.com/ArjenSchwarz/fog/cmd/services"
	"github.com/ArjenSchwarz/fog/cmd/services/aws"
	"github.com/ArjenSchwarz/fog/cmd/services/deployment"
	"github.com/ArjenSchwarz/fog/config"
)

// ServiceFactory creates service instances with proper dependencies
// and provides access to configuration objects.
// ServiceFactory creates service instances with proper dependencies
// and provides access to configuration objects.
type ServiceFactory struct {
	config    *config.Config
	awsConfig *config.AWSConfig
}

// NewServiceFactory creates a new service factory.
func NewServiceFactory(cfg *config.Config, awsCfg *config.AWSConfig) *ServiceFactory {
	return &ServiceFactory{config: cfg, awsConfig: awsCfg}
}

// AppConfig returns the application config.
func (f *ServiceFactory) AppConfig() *config.Config { return f.config }

// AWSConfig returns the AWS configuration.
func (f *ServiceFactory) AWSConfig() *config.AWSConfig { return f.awsConfig }

// CreateDeploymentService creates a deployment service with dependencies.
func (f *ServiceFactory) CreateDeploymentService() services.DeploymentService {
	cfnClient := aws.NewCloudFormationClient(*f.awsConfig)
	s3Client := aws.NewS3Client(*f.awsConfig)

	templateService := deployment.NewTemplateService(s3Client)
	parameterService := deployment.NewParameterService()
	tagService := deployment.NewTagService()

	return deployment.NewService(
		templateService,
		parameterService,
		tagService,
		cfnClient,
		s3Client,
		f.config,
	)
}

// CreateDriftService creates a drift detection service.
func (f *ServiceFactory) CreateDriftService() services.DriftService {
	return nil
}

// CreateStackService creates a stack operations service.
func (f *ServiceFactory) CreateStackService() services.StackService {
	return nil
}
