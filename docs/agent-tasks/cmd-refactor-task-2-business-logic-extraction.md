# Task 2: Business Logic Extraction

## Objective

Separate business logic from CLI presentation concerns by creating dedicated service layers for each major operation (deploy, drift, describe, etc.) and moving AWS interaction logic to testable service classes.

## Current State

### Problems
- Business logic mixed with CLI presentation code in monolithic command files
- Direct AWS SDK calls scattered throughout command handlers
- Template processing, parameter handling embedded in command logic
- Difficult to unit test business operations independently
- Code duplication across similar operations

### Current Monolithic Files
- `deploy.go` - 500+ lines with deployment logic, AWS calls, UI interactions
- `drift.go` - Drift detection mixed with result formatting
- `describe_changeset.go` - Changeset logic mixed with output formatting
- `deploy_helpers.go` - Some separation but still tightly coupled

### Problematic Patterns
```go
// Current: Business logic mixed with UI
func deployTemplate(cmd *cobra.Command, args []string) {
    // Flag validation
    // AWS client creation
    // Template processing
    // Parameter handling
    // Changeset creation
    // User confirmation
    // Deployment execution
    // Result formatting
    // Error handling
}
```

## Target State

### Goals
- Clean separation between business logic and presentation
- Testable service classes with dependency injection
- Reusable business operations across different interfaces
- Consistent error handling and operation patterns
- Clear interfaces for external dependencies

### Service Layer Architecture
```
cmd/
├── services/
│   ├── interfaces.go           # Service interfaces
│   ├── deployment/
│   │   ├── service.go          # Deployment orchestration
│   │   ├── changeset.go        # Changeset operations
│   │   ├── template.go         # Template processing
│   │   ├── parameters.go       # Parameter handling
│   │   └── validation.go       # Deployment validation
│   ├── drift/
│   │   ├── service.go          # Drift detection service
│   │   ├── analyzer.go         # Drift analysis logic
│   │   └── comparison.go       # Resource comparison
│   ├── stack/
│   │   ├── service.go          # Stack operations
│   │   ├── description.go      # Stack description logic
│   │   └── history.go          # Stack history operations
│   ├── aws/
│   │   ├── clients.go          # AWS client management
│   │   ├── cloudformation.go   # CloudFormation operations
│   │   ├── s3.go              # S3 operations
│   │   └── mocks.go           # Mock implementations
│   └── common/
│       ├── errors.go           # Common error types
│       ├── context.go          # Operation context
│       └── logging.go          # Operation logging
```

## Prerequisites

- Task 1: Command Structure Reorganization (provides the foundation)

## Step-by-Step Implementation

### Step 1: Define Service Interfaces

**File**: `cmd/services/interfaces.go`

```go
package services

import (
    "context"
    "github.com/ArjenSchwarz/fog/config"
    "github.com/ArjenSchwarz/fog/lib"
    "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// DeploymentService handles stack deployment operations
type DeploymentService interface {
    PrepareDeployment(ctx context.Context, opts DeploymentOptions) (*DeploymentPlan, error)
    ValidateDeployment(ctx context.Context, plan *DeploymentPlan) error
    CreateChangeset(ctx context.Context, plan *DeploymentPlan) (*ChangesetResult, error)
    ExecuteDeployment(ctx context.Context, plan *DeploymentPlan, changeset *ChangesetResult) (*DeploymentResult, error)
}

// DriftService handles drift detection operations
type DriftService interface {
    DetectDrift(ctx context.Context, stackName string, opts DriftOptions) (*DriftResult, error)
    AnalyzeDrift(ctx context.Context, result *DriftResult) (*DriftAnalysis, error)
}

// StackService handles general stack operations
type StackService interface {
    DescribeStack(ctx context.Context, stackName string) (*StackDescription, error)
    ListResources(ctx context.Context, stackName string) (*ResourceList, error)
    GetHistory(ctx context.Context, stackName string, opts HistoryOptions) (*StackHistory, error)
}

// TemplateService handles template operations
type TemplateService interface {
    LoadTemplate(ctx context.Context, templatePath string) (*Template, error)
    ValidateTemplate(ctx context.Context, template *Template) error
    UploadTemplate(ctx context.Context, template *Template, bucket string) (*TemplateReference, error)
}

// ParameterService handles parameter operations
type ParameterService interface {
    LoadParameters(ctx context.Context, parameterFiles []string) ([]types.Parameter, error)
    ValidateParameters(ctx context.Context, parameters []types.Parameter, template *Template) error
}

// TagService handles tag operations
type TagService interface {
    LoadTags(ctx context.Context, tagFiles []string, defaults map[string]string) ([]types.Tag, error)
    ValidateTags(ctx context.Context, tags []types.Tag) error
}

// CloudFormationClient abstracts AWS CloudFormation operations
type CloudFormationClient interface {
    DescribeStacks(ctx context.Context, input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error)
    CreateChangeSet(ctx context.Context, input *cloudformation.CreateChangeSetInput) (*cloudformation.CreateChangeSetOutput, error)
    ExecuteChangeSet(ctx context.Context, input *cloudformation.ExecuteChangeSetInput) (*cloudformation.ExecuteChangeSetOutput, error)
    DescribeChangeSet(ctx context.Context, input *cloudformation.DescribeChangeSetInput) (*cloudformation.DescribeChangeSetOutput, error)
    // Add other required CloudFormation operations
}

// S3Client abstracts AWS S3 operations
type S3Client interface {
    PutObject(ctx context.Context, input *s3.PutObjectInput) (*s3.PutObjectOutput, error)
    GetObject(ctx context.Context, input *s3.GetObjectInput) (*s3.GetObjectOutput, error)
}
```

### Step 2: Define Data Transfer Objects

**File**: `cmd/services/types.go`

```go
package services

import (
    "time"
    "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// DeploymentOptions contains all options for deployment
type DeploymentOptions struct {
    StackName       string
    TemplateSource  string
    ParameterFiles  []string
    TagFiles        []string
    DefaultTags     bool
    Bucket          string
    ChangesetName   string
    DeploymentFile  string
    DryRun          bool
    NonInteractive  bool
    CreateOnly      bool
    DeployOnly      bool
}

// DeploymentPlan contains the prepared deployment configuration
type DeploymentPlan struct {
    StackName       string
    IsNewStack      bool
    Template        *Template
    Parameters      []types.Parameter
    Tags            []types.Tag
    ChangesetName   string
    DeploymentInfo  *lib.DeployInfo
    Options         DeploymentOptions
}

// Template represents a CloudFormation template
type Template struct {
    Content     string
    LocalPath   string
    S3URL       string
    Size        int64
    Hash        string
}

// TemplateReference points to a template location
type TemplateReference struct {
    URL         string
    Bucket      string
    Key         string
    Version     string
}

// ChangesetResult contains changeset creation results
type ChangesetResult struct {
    Name           string
    ID             string
    Status         types.ChangeSetStatus
    StatusReason   string
    Changes        []types.Change
    CreationTime   time.Time
    StackID        string
    ConsoleURL     string
}

// DeploymentResult contains deployment execution results
type DeploymentResult struct {
    StackID        string
    Status         types.StackStatus
    Outputs        []types.Output
    Events         []types.StackEvent
    ExecutionTime  time.Duration
    Success        bool
    ErrorMessage   string
}

// DriftOptions contains drift detection options
type DriftOptions struct {
    ResultsOnly         bool
    SeparateProperties  bool
    IgnoreTags          []string
}

// DriftResult contains raw drift detection results
type DriftResult struct {
    StackID            string
    DriftStatus        types.StackDriftStatus
    DriftedResources   []types.StackResourceDrift
    DetectionTime      time.Time
}

// DriftAnalysis contains analyzed drift information
type DriftAnalysis struct {
    Summary            DriftSummary
    CriticalChanges    []ResourceDrift
    MinorChanges       []ResourceDrift
    ManagedResources   []string
    UnmanagedResources []string
}

// DriftSummary provides overview of drift
type DriftSummary struct {
    TotalResources   int
    DriftedResources int
    CriticalDrifts   int
    MinorDrifts      int
}

// ResourceDrift represents drift in a single resource
type ResourceDrift struct {
    LogicalID    string
    PhysicalID   string
    ResourceType string
    DriftStatus  types.ResourceStatus
    Properties   []PropertyDrift
}

// PropertyDrift represents drift in a resource property
type PropertyDrift struct {
    Path         string
    Expected     interface{}
    Actual       interface{}
    ChangeType   string
}

// StackDescription contains stack information
type StackDescription struct {
    StackID      string
    StackName    string
    Status       types.StackStatus
    CreationTime time.Time
    UpdateTime   *time.Time
    Description  string
    Parameters   []types.Parameter
    Outputs      []types.Output
    Tags         []types.Tag
}

// ResourceList contains stack resources
type ResourceList struct {
    Resources []StackResource
}

// StackResource represents a CloudFormation resource
type StackResource struct {
    LogicalID    string
    PhysicalID   string
    Type         string
    Status       types.ResourceStatus
    LastUpdated  time.Time
}

// HistoryOptions contains history query options
type HistoryOptions struct {
    Limit        int
    StartTime    *time.Time
    EndTime      *time.Time
    EventTypes   []string
}

// StackHistory contains stack event history
type StackHistory struct {
    Events []StackEvent
}

// StackEvent represents a CloudFormation stack event
type StackEvent struct {
    EventID      string
    StackID      string
    LogicalID    string
    PhysicalID   string
    ResourceType string
    Status       types.ResourceStatus
    Reason       string
    Timestamp    time.Time
}
```

### Step 3: Implement Deployment Service

**File**: `cmd/services/deployment/service.go`

```go
package deployment

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/config"
)

// Service implements the DeploymentService interface
type Service struct {
    templateService  services.TemplateService
    parameterService services.ParameterService
    tagService       services.TagService
    cfnClient        services.CloudFormationClient
    s3Client         services.S3Client
    config           *config.Config
}

// NewService creates a new deployment service
func NewService(
    templateService services.TemplateService,
    parameterService services.ParameterService,
    tagService services.TagService,
    cfnClient services.CloudFormationClient,
    s3Client services.S3Client,
    config *config.Config,
) *Service {
    return &Service{
        templateService:  templateService,
        parameterService: parameterService,
        tagService:       tagService,
        cfnClient:        cfnClient,
        s3Client:         s3Client,
        config:           config,
    }
}

// PrepareDeployment creates a deployment plan from options
func (s *Service) PrepareDeployment(ctx context.Context, opts services.DeploymentOptions) (*services.DeploymentPlan, error) {
    plan := &services.DeploymentPlan{
        StackName: opts.StackName,
        Options:   opts,
    }

    // Check if stack exists
    isNew, err := s.isNewStack(ctx, opts.StackName)
    if err != nil {
        return nil, fmt.Errorf("failed to check stack status: %w", err)
    }
    plan.IsNewStack = isNew

    // Load template
    template, err := s.templateService.LoadTemplate(ctx, opts.TemplateSource)
    if err != nil {
        return nil, fmt.Errorf("failed to load template: %w", err)
    }
    plan.Template = template

    // Upload template if bucket specified
    if opts.Bucket != "" {
        templateRef, err := s.templateService.UploadTemplate(ctx, template, opts.Bucket)
        if err != nil {
            return nil, fmt.Errorf("failed to upload template: %w", err)
        }
        template.S3URL = templateRef.URL
    }

    // Load parameters
    if len(opts.ParameterFiles) > 0 {
        parameters, err := s.parameterService.LoadParameters(ctx, opts.ParameterFiles)
        if err != nil {
            return nil, fmt.Errorf("failed to load parameters: %w", err)
        }
        plan.Parameters = parameters
    }

    // Load tags
    defaultTags := make(map[string]string)
    if opts.DefaultTags {
        defaultTags = s.config.GetStringMapString("tags.default")
    }

    tags, err := s.tagService.LoadTags(ctx, opts.TagFiles, defaultTags)
    if err != nil {
        return nil, fmt.Errorf("failed to load tags: %w", err)
    }
    plan.Tags = tags

    // Generate changeset name
    if opts.ChangesetName == "" {
        plan.ChangesetName = s.generateChangesetName()
    } else {
        plan.ChangesetName = opts.ChangesetName
    }

    return plan, nil
}

// ValidateDeployment validates the deployment plan
func (s *Service) ValidateDeployment(ctx context.Context, plan *services.DeploymentPlan) error {
    // Validate template
    if err := s.templateService.ValidateTemplate(ctx, plan.Template); err != nil {
        return fmt.Errorf("template validation failed: %w", err)
    }

    // Validate parameters
    if err := s.parameterService.ValidateParameters(ctx, plan.Parameters, plan.Template); err != nil {
        return fmt.Errorf("parameter validation failed: %w", err)
    }

    // Validate tags
    if err := s.tagService.ValidateTags(ctx, plan.Tags); err != nil {
        return fmt.Errorf("tag validation failed: %w", err)
    }

    // Check if stack is in valid state for update
    if !plan.IsNewStack {
        ready, status, err := s.isStackReadyForUpdate(ctx, plan.StackName)
        if err != nil {
            return fmt.Errorf("failed to check stack status: %w", err)
        }
        if !ready {
            return fmt.Errorf("stack '%s' is in status %s and cannot be updated", plan.StackName, status)
        }
    }

    return nil
}

// CreateChangeset creates a changeset for the deployment
func (s *Service) CreateChangeset(ctx context.Context, plan *services.DeploymentPlan) (*services.ChangesetResult, error) {
    // Implementation details for changeset creation
    // This would use the CloudFormation client to create the changeset
    return nil, fmt.Errorf("changeset creation not yet implemented")
}

// ExecuteDeployment executes the deployment plan
func (s *Service) ExecuteDeployment(ctx context.Context, plan *services.DeploymentPlan, changeset *services.ChangesetResult) (*services.DeploymentResult, error) {
    // Implementation details for deployment execution
    // This would execute the changeset and monitor progress
    return nil, fmt.Errorf("deployment execution not yet implemented")
}

// Private helper methods
func (s *Service) isNewStack(ctx context.Context, stackName string) (bool, error) {
    // Implementation to check if stack exists
    return false, nil
}

func (s *Service) isStackReadyForUpdate(ctx context.Context, stackName string) (bool, string, error) {
    // Implementation to check stack status
    return false, "", nil
}

func (s *Service) generateChangesetName() string {
    // Implementation to generate changeset name
    return "fog-changeset"
}
```

### Step 4: Implement Template Service

**File**: `cmd/services/deployment/template.go`

```go
package deployment

import (
    "context"
    "crypto/sha256"
    "fmt"
    "os"
    "path/filepath"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/lib"
)

// TemplateService implements template operations
type TemplateService struct {
    s3Client services.S3Client
}

// NewTemplateService creates a new template service
func NewTemplateService(s3Client services.S3Client) *TemplateService {
    return &TemplateService{
        s3Client: s3Client,
    }
}

// LoadTemplate loads a template from file system
func (ts *TemplateService) LoadTemplate(ctx context.Context, templatePath string) (*services.Template, error) {
    content, path, err := lib.ReadTemplate(&templatePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read template: %w", err)
    }

    // Get file info
    fileInfo, err := os.Stat(path)
    if err != nil {
        return nil, fmt.Errorf("failed to get template file info: %w", err)
    }

    // Calculate hash
    hash := sha256.Sum256([]byte(content))

    // Get absolute path
    absPath, err := filepath.Abs(path)
    if err != nil {
        return nil, fmt.Errorf("failed to get absolute path: %w", err)
    }

    template := &services.Template{
        Content:   content,
        LocalPath: absPath,
        Size:      fileInfo.Size(),
        Hash:      fmt.Sprintf("%x", hash),
    }

    return template, nil
}

// ValidateTemplate validates a CloudFormation template
func (ts *TemplateService) ValidateTemplate(ctx context.Context, template *services.Template) error {
    // Template validation logic
    // This could use AWS CloudFormation validate-template API
    // or local validation using go-cfn-lint or similar

    if template.Content == "" {
        return fmt.Errorf("template content is empty")
    }

    if template.Size > 460800 { // 450KB limit for direct upload
        if template.S3URL == "" {
            return fmt.Errorf("template is too large (%d bytes) and no S3 bucket specified", template.Size)
        }
    }

    return nil
}

// UploadTemplate uploads a template to S3
func (ts *TemplateService) UploadTemplate(ctx context.Context, template *services.Template, bucket string) (*services.TemplateReference, error) {
    // Generate S3 key
    key := fmt.Sprintf("templates/%s/%s.yaml", template.Hash[:8], filepath.Base(template.LocalPath))

    // Upload to S3 using lib.UploadTemplate or similar
    // This is a placeholder - the actual implementation would use the S3 client
    url := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)

    template.S3URL = url

    return &services.TemplateReference{
        URL:     url,
        Bucket:  bucket,
        Key:     key,
        Version: template.Hash,
    }, nil
}
```

### Step 5: Update Deploy Command Handler

**File**: `cmd/commands/deploy/handler.go` (update from Task 1)

```go
package deploy

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/cmd/services/deployment"
    "github.com/ArjenSchwarz/fog/config"
)

// Handler implements the deploy command logic
type Handler struct {
    flags             *Flags
    deploymentService services.DeploymentService
    config            *config.Config
}

// NewHandler creates a new deploy command handler
func NewHandler(flags *Flags, deploymentService services.DeploymentService, config *config.Config) *Handler {
    return &Handler{
        flags:             flags,
        deploymentService: deploymentService,
        config:            config,
    }
}

// Execute runs the deploy command
func (h *Handler) Execute(ctx context.Context) error {
    // Convert flags to deployment options
    opts := services.DeploymentOptions{
        StackName:      h.flags.StackName,
        TemplateSource: h.flags.Template,
        ParameterFiles: parseCommaSeparated(h.flags.Parameters),
        TagFiles:       parseCommaSeparated(h.flags.Tags),
        DefaultTags:    h.flags.DefaultTags,
        Bucket:         h.flags.Bucket,
        ChangesetName:  h.flags.ChangesetName,
        DeploymentFile: h.flags.DeploymentFile,
        DryRun:         h.flags.Dryrun,
        NonInteractive: h.flags.NonInteractive,
        CreateOnly:     h.flags.CreateChangeset,
        DeployOnly:     h.flags.DeployChangeset,
    }

    // Prepare deployment
    plan, err := h.deploymentService.PrepareDeployment(ctx, opts)
    if err != nil {
        return fmt.Errorf("failed to prepare deployment: %w", err)
    }

    // Validate deployment
    if err := h.deploymentService.ValidateDeployment(ctx, plan); err != nil {
        return fmt.Errorf("deployment validation failed: %w", err)
    }

    // Create changeset
    changeset, err := h.deploymentService.CreateChangeset(ctx, plan)
    if err != nil {
        return fmt.Errorf("failed to create changeset: %w", err)
    }

    // Handle dry run
    if opts.DryRun {
        // Show changeset and exit
        return nil
    }

    // Handle create-only mode
    if opts.CreateOnly {
        // Show changeset creation success and exit
        return nil
    }

    // Execute deployment
    result, err := h.deploymentService.ExecuteDeployment(ctx, plan, changeset)
    if err != nil {
        return fmt.Errorf("deployment failed: %w", err)
    }

    // Show results
    if result.Success {
        return nil
    } else {
        return fmt.Errorf("deployment completed with errors: %s", result.ErrorMessage)
    }
}

// ValidateFlags validates the command flags
func (h *Handler) ValidateFlags() error {
    return h.flags.Validate()
}

// Helper function to parse comma-separated strings
func parseCommaSeparated(input string) []string {
    if input == "" {
        return nil
    }
    // Implementation to split by comma and trim whitespace
    return []string{input} // Placeholder
}
```

### Step 6: Create Service Factory

**File**: `cmd/services/factory.go`

```go
package services

import (
    "github.com/ArjenSchwarz/fog/cmd/services/deployment"
    "github.com/ArjenSchwarz/fog/config"
)

// ServiceFactory creates service instances with proper dependencies
type ServiceFactory struct {
    config    *config.Config
    awsConfig *config.AWSConfig
}

// NewServiceFactory creates a new service factory
func NewServiceFactory(config *config.Config, awsConfig *config.AWSConfig) *ServiceFactory {
    return &ServiceFactory{
        config:    config,
        awsConfig: awsConfig,
    }
}

// CreateDeploymentService creates a deployment service with dependencies
func (f *ServiceFactory) CreateDeploymentService() DeploymentService {
    // Create AWS clients
    cfnClient := f.awsConfig.CloudformationClient()
    s3Client := f.awsConfig.S3Client()

    // Create sub-services
    templateService := deployment.NewTemplateService(s3Client)
    parameterService := deployment.NewParameterService()
    tagService := deployment.NewTagService()

    // Create main service
    return deployment.NewService(
        templateService,
        parameterService,
        tagService,
        cfnClient,
        s3Client,
        f.config,
    )
}

// CreateDriftService creates a drift detection service
func (f *ServiceFactory) CreateDriftService() DriftService {
    // Implementation for drift service creation
    return nil
}

// CreateStackService creates a stack operations service
func (f *ServiceFactory) CreateStackService() StackService {
    // Implementation for stack service creation
    return nil
}
```

## Files to Create/Modify

### New Files
- `cmd/services/interfaces.go`
- `cmd/services/types.go`
- `cmd/services/factory.go`
- `cmd/services/deployment/service.go`
- `cmd/services/deployment/template.go`
- `cmd/services/deployment/parameters.go`
- `cmd/services/deployment/tags.go`
- `cmd/services/deployment/changeset.go`
- `cmd/services/deployment/validation.go`
- `cmd/services/aws/clients.go`
- `cmd/services/aws/cloudformation.go`
- `cmd/services/aws/s3.go`
- `cmd/services/aws/mocks.go`
- `cmd/services/common/errors.go`
- `cmd/services/common/context.go`

### Modified Files
- `cmd/commands/deploy/handler.go` - Use deployment service
- `cmd/commands/deploy/command.go` - Inject services
- `cmd/root.go` - Create service factory

## Testing Strategy

### Unit Tests
- Test each service independently with mocked dependencies
- Test service factory dependency injection
- Test data transformation between DTOs and existing types
- Test error handling and validation logic

### Integration Tests
- Test service orchestration with real AWS clients
- Test backward compatibility with existing lib types
- Test end-to-end deployment flows

### Test Files to Create
- `cmd/services/deployment/service_test.go`
- `cmd/services/deployment/template_test.go`
- `cmd/services/aws/mocks_test.go`
- `cmd/services/factory_test.go`

## Success Criteria

### Functional Requirements
- [ ] Deploy command uses new service layer
- [ ] All existing deployment functionality preserved
- [ ] Services are independently testable
- [ ] Clear separation between business logic and presentation

### Quality Requirements
- [ ] Unit tests cover >85% of service code
- [ ] All AWS calls abstracted behind interfaces
- [ ] Error handling follows consistent patterns
- [ ] Services use dependency injection

### Performance Requirements
- [ ] No performance degradation in deployment operations
- [ ] Memory usage remains stable
- [ ] Service creation overhead minimal

## Migration Timeline

### Phase 1: Foundation
- Create service interfaces and DTOs
- Implement deployment service framework
- Create service factory

### Phase 2: Core Services
- Implement template, parameter, tag services
- Migrate deployment operations
- Add comprehensive testing

### Phase 3: Additional Services
- Implement drift detection service
- Implement stack operations service
- Migrate remaining commands

## Dependencies

### Upstream Dependencies
- Task 1: Command Structure Reorganization (provides handler framework)

### Downstream Dependencies
- Task 6: Testing Infrastructure (will use service interfaces for mocking)

## Risk Mitigation

### Potential Issues
- Breaking changes to existing lib package integration
- Performance overhead from service layer
- Complexity in service dependency management

### Mitigation Strategies
- Maintain compatibility with existing lib types
- Performance testing for service layer overhead
- Clear service boundaries and minimal dependencies
- Gradual migration with fallback options
