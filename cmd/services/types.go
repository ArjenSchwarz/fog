package services

import (
	"time"

	"github.com/ArjenSchwarz/fog/lib"
	cfnTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
)

// DeploymentOptions contains all options for deployment.
type DeploymentOptions struct {
	StackName      string
	TemplateSource string
	ParameterFiles []string
	TagFiles       []string
	DefaultTags    bool
	Bucket         string
	ChangesetName  string
	DeploymentFile string
	DryRun         bool
	NonInteractive bool
	CreateOnly     bool
	DeployOnly     bool
}

// DeploymentPlan contains the prepared deployment configuration.
type DeploymentPlan struct {
	StackName      string
	IsNewStack     bool
	Template       *Template
	Parameters     []cfnTypes.Parameter
	Tags           []cfnTypes.Tag
	ChangesetName  string
	DeploymentInfo *lib.DeployInfo
	Options        DeploymentOptions
}

// Template represents a CloudFormation template.
type Template struct {
	Content   string
	LocalPath string
	S3URL     string
	Size      int64
	Hash      string
}

// TemplateReference points to a template location.
type TemplateReference struct {
	URL     string
	Bucket  string
	Key     string
	Version string
}

// ChangesetResult contains changeset creation results.
type ChangesetResult struct {
	Name         string
	ID           string
	Status       cfnTypes.ChangeSetStatus
	StatusReason string
	Changes      []cfnTypes.Change
	CreationTime time.Time
	StackID      string
	ConsoleURL   string
}

// DeploymentResult contains deployment execution results.
type DeploymentResult struct {
	StackID       string
	Status        cfnTypes.StackStatus
	Outputs       []cfnTypes.Output
	Events        []cfnTypes.StackEvent
	ExecutionTime time.Duration
	Success       bool
	ErrorMessage  string
}

// DriftOptions contains drift detection options.
type DriftOptions struct {
	ResultsOnly        bool
	SeparateProperties bool
	IgnoreTags         []string
}

// DriftResult contains raw drift detection results.
type DriftResult struct {
	StackID          string
	DriftStatus      cfnTypes.StackDriftStatus
	DriftedResources []cfnTypes.StackResourceDrift
	DetectionTime    time.Time
}

// DriftAnalysis contains analyzed drift information.
type DriftAnalysis struct {
	Summary            DriftSummary
	CriticalChanges    []ResourceDrift
	MinorChanges       []ResourceDrift
	ManagedResources   []string
	UnmanagedResources []string
}

// DriftSummary provides overview of drift.
type DriftSummary struct {
	TotalResources   int
	DriftedResources int
	CriticalDrifts   int
	MinorDrifts      int
}

// ResourceDrift represents drift in a single resource.
type ResourceDrift struct {
	LogicalID    string
	PhysicalID   string
	ResourceType string
	DriftStatus  cfnTypes.ResourceStatus
	Properties   []PropertyDrift
}

// PropertyDrift represents drift in a resource property.
type PropertyDrift struct {
	Path       string
	Expected   interface{}
	Actual     interface{}
	ChangeType string
}

// StackDescription contains stack information.
type StackDescription struct {
	StackID      string
	StackName    string
	Status       cfnTypes.StackStatus
	CreationTime time.Time
	UpdateTime   *time.Time
	Description  string
	Parameters   []cfnTypes.Parameter
	Outputs      []cfnTypes.Output
	Tags         []cfnTypes.Tag
}

// ResourceList contains stack resources.
type ResourceList struct {
	Resources []StackResource
}

// StackResource represents a CloudFormation resource.
type StackResource struct {
	LogicalID   string
	PhysicalID  string
	Type        string
	Status      cfnTypes.ResourceStatus
	LastUpdated time.Time
}

// HistoryOptions contains history query options.
type HistoryOptions struct {
	Limit      int
	StartTime  *time.Time
	EndTime    *time.Time
	EventTypes []string
}

// StackHistory contains stack event history.
type StackHistory struct {
	Events []StackEvent
}

// StackEvent represents a CloudFormation stack event.
type StackEvent struct {
	EventID      string
	StackID      string
	LogicalID    string
	PhysicalID   string
	ResourceType string
	Status       cfnTypes.ResourceStatus
	Reason       string
	Timestamp    time.Time
}
