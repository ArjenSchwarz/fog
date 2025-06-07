package ui

import (
	"io"
	"time"
)

// OutputHandler manages all output operations
// This is a simplified version of the interface described in the documentation.
type OutputHandler interface {
	Success(message string)
	Info(message string)
	Warning(message string)
	Error(message string)
	Debug(message string)

	Table(data interface{}, options TableOptions) error
	JSON(data interface{}) error

	StartProgress(message string) ProgressIndicator
	SetStatus(message string)

	Confirm(message string) bool
	ConfirmWithDefault(message string, defaultValue bool) bool

	SetVerbose(verbose bool)
	SetQuiet(quiet bool)
	SetOutputFormat(format OutputFormat)

	GetWriter() io.Writer
	GetErrorWriter() io.Writer

	// GetVerbose returns whether verbose mode is enabled
	GetVerbose() bool
}

// ProgressIndicator represents a progress indicator
// used for long running operations.
type ProgressIndicator interface {
	Update(message string)
	Success(message string)
	Error(message string)
	Stop()
}

// TableOptions configures table output
// for various command results.
type TableOptions struct {
	Title     string
	Headers   []string
	MaxWidth  int
	Style     TableStyle
	SortBy    string
	ShowIndex bool
}

// TableStyle defines table visual styles.
type TableStyle int

const (
	TableStyleDefault TableStyle = iota
	TableStyleMinimal
	TableStyleBordered
	TableStyleCompact
)

// OutputFormat defines the output format
// for structured command results.
type OutputFormat int

const (
	FormatTable OutputFormat = iota
	FormatJSON
	FormatCSV
	FormatYAML
	FormatText
)

// Theme defines UI styling. Only included here
// for completeness; implementations may ignore it.
type Theme interface {
	Success() string
	Info() string
	Warning() string
	Error() string
	Debug() string
	Emphasis() string
	Muted() string

	ProgressSpinner() string
	ProgressSuccess() string
	ProgressError() string

	TableHeader() string
	TableBorder() string
	TableData() string
}

// Formatter formats specific output structures.
type Formatter interface {
	FormatDeploymentInfo(info DeploymentInfo) string
	FormatChangeset(changeset ChangesetInfo) string
	FormatDriftResult(result DriftResult) string
	FormatStackInfo(stack StackInfo) string
}

// ValidationDisplayer handles validation message display.
type ValidationDisplayer interface {
	ShowValidationErrors(errors []ValidationError)
	ShowValidationWarnings(warnings []ValidationWarning)
	ShowValidationSummary(summary ValidationSummary)
}

// Various data structures referenced by Formatter and ValidationDisplayer.
type DeploymentInfo struct {
	StackName    string
	Region       string
	Account      string
	IsNew        bool
	DryRun       bool
	TemplateInfo TemplateInfo
}

type TemplateInfo struct {
	Path  string
	Size  int64
	S3URL string
	Hash  string
}

type ChangesetInfo struct {
	Name        string
	Status      string
	Changes     []ChangeInfo
	Summary     ChangeSummary
	DangerLevel DangerLevel
}

type ChangeInfo struct {
	Action       string
	ResourceType string
	LogicalID    string
	PhysicalID   string
	Replacement  string
	Details      []ChangeDetail
}

type ChangeDetail struct {
	Property   string
	OldValue   interface{}
	NewValue   interface{}
	ChangeType string
}

type ChangeSummary struct {
	TotalChanges  int
	Additions     int
	Modifications int
	Deletions     int
	Replacements  int
}

type DangerLevel int

const (
	DangerLow DangerLevel = iota
	DangerMedium
	DangerHigh
	DangerCritical
)

type DriftResult struct {
	StackName      string
	DriftStatus    string
	TotalResources int
	DriftedCount   int
	Resources      []DriftedResource
}

type DriftedResource struct {
	LogicalID    string
	PhysicalID   string
	ResourceType string
	DriftStatus  string
	Properties   []PropertyDrift
}

type PropertyDrift struct {
	PropertyPath string
	Expected     interface{}
	Actual       interface{}
	DriftType    string
}

type StackInfo struct {
	Name       string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  *time.Time
	Parameters []Parameter
	Outputs    []Output
	Tags       []Tag
}

// Import time after using above struct referencing time
type Parameter struct {
	Key   string
	Value string
}

type Output struct {
	Key         string
	Value       string
	Description string
	ExportName  string
}

type Tag struct {
	Key   string
	Value string
}

type ValidationError struct {
	Field   string
	Message string
	Code    string
}

type ValidationWarning struct {
	Field   string
	Message string
	Code    string
}

type ValidationSummary struct {
	ErrorCount   int
	WarningCount int
	InfoCount    int
}
