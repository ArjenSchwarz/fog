# Task 3: Flag Management Refactoring

## Objective

Improve flag handling consistency and validation by enhancing the existing flag management system with better validation patterns, dependency checking, and standardized flag processing across all commands.

## Current State

### Problems
- Flag validation scattered across command files
- Inconsistent validation patterns between commands
- No dependency checking between related flags
- Global flag variables mixed with command-specific flags
- Limited validation feedback and error messages
- No standardized flag grouping or conditional logic

### Current Flag Implementation
- `flaggroups.go` - Basic flag structures with minimal validation
- Individual command files have their own flag handling patterns
- Global flags defined in `root.go` with viper binding
- No clear relationship between related flags
- Limited validation beyond basic type checking

### Problematic Patterns
```go
// Current: Basic validation without context
func (f *DeployFlags) Validate() error {
    if f.StackName == "" {
        return fmt.Errorf("stack name is required")
    }
    // Limited validation logic
    return nil
}
```

## Target State

### Goals
- Comprehensive flag validation with clear error messages
- Flag dependency and conflict checking
- Standardized validation patterns across commands
- Better flag organization and grouping
- Context-aware validation (e.g., AWS region availability)
- Reusable validation components

### Enhanced Flag Architecture
```
cmd/
├── flags/
│   ├── interfaces.go           # Flag validation interfaces
│   ├── base.go                # Base flag functionality
│   ├── validators/
│   │   ├── required.go         # Required field validation
│   │   ├── dependencies.go     # Flag dependency validation
│   │   ├── conflicts.go        # Flag conflict validation
│   │   ├── format.go          # Format validation (files, URLs, etc.)
│   │   ├── aws.go             # AWS-specific validation
│   │   └── custom.go          # Custom validation logic
│   ├── groups/
│   │   ├── deployment.go       # Deployment flag group
│   │   ├── drift.go           # Drift detection flag group
│   │   ├── stack.go           # Stack operation flag group
│   │   └── common.go          # Common flag patterns
│   └── middleware/
│       ├── validation.go       # Validation middleware
│       └── preprocessing.go    # Flag preprocessing
```

## Prerequisites

- Task 1: Command Structure Reorganization (provides middleware framework)

## Step-by-Step Implementation

### Step 1: Create Flag Validation Interfaces

**File**: `cmd/flags/interfaces.go`

```go
package flags

import (
    "context"
    "github.com/spf13/cobra"
)

// FlagValidator defines the interface for flag validation
type FlagValidator interface {
    Validate(ctx context.Context) error
    RegisterFlags(cmd *cobra.Command)
    GetValidationRules() []ValidationRule
}

// ValidationRule defines a single validation rule
type ValidationRule interface {
    Validate(ctx context.Context, flags FlagValidator) error
    GetDescription() string
    GetSeverity() ValidationSeverity
}

// ValidationSeverity indicates the importance of a validation rule
type ValidationSeverity int

const (
    SeverityError ValidationSeverity = iota
    SeverityWarning
    SeverityInfo
)

// FlagGroup represents a logical group of related flags
type FlagGroup interface {
    GetName() string
    GetFlags() []FlagDefinition
    GetValidationRules() []ValidationRule
    RegisterFlags(cmd *cobra.Command)
}

// FlagDefinition defines a single flag
type FlagDefinition struct {
    Name         string
    Shorthand    string
    Description  string
    DefaultValue interface{}
    Required     bool
    Hidden       bool
    Deprecated   bool
    FlagType     FlagType
}

// FlagType represents the type of flag
type FlagType int

const (
    StringFlag FlagType = iota
    StringSliceFlag
    BoolFlag
    IntFlag
    DurationFlag
)

// ValidationContext provides context for validation
type ValidationContext struct {
    Command    *cobra.Command
    Args       []string
    AWSRegion  string
    ConfigPath string
    Verbose    bool
}

// FlagPreprocessor handles flag preprocessing
type FlagPreprocessor interface {
    Process(ctx context.Context, flags FlagValidator) error
}
```

### Step 2: Implement Base Flag Functionality

**File**: `cmd/flags/base.go`

```go
package flags

import (
    "context"
    "fmt"
    "github.com/spf13/cobra"
)

// BaseFlagValidator provides common flag validation functionality
type BaseFlagValidator struct {
    rules       []ValidationRule
    groups      []FlagGroup
    preprocessors []FlagPreprocessor
}

// NewBaseFlagValidator creates a new base flag validator
func NewBaseFlagValidator() *BaseFlagValidator {
    return &BaseFlagValidator{
        rules:       make([]ValidationRule, 0),
        groups:      make([]FlagGroup, 0),
        preprocessors: make([]FlagPreprocessor, 0),
    }
}

// AddRule adds a validation rule
func (b *BaseFlagValidator) AddRule(rule ValidationRule) {
    b.rules = append(b.rules, rule)
}

// AddGroup adds a flag group
func (b *BaseFlagValidator) AddGroup(group FlagGroup) {
    b.groups = append(b.groups, group)
    // Add group's validation rules
    for _, rule := range group.GetValidationRules() {
        b.AddRule(rule)
    }
}

// AddPreprocessor adds a flag preprocessor
func (b *BaseFlagValidator) AddPreprocessor(preprocessor FlagPreprocessor) {
    b.preprocessors = append(b.preprocessors, preprocessor)
}

// Validate validates all flags and rules
func (b *BaseFlagValidator) Validate(ctx context.Context) error {
    // Run preprocessors first
    for _, preprocessor := range b.preprocessors {
        if err := preprocessor.Process(ctx, b); err != nil {
            return fmt.Errorf("flag preprocessing failed: %w", err)
        }
    }

    // Collect all validation errors
    var errors []error
    var warnings []string

    // Run all validation rules
    for _, rule := range b.rules {
        if err := rule.Validate(ctx, b); err != nil {
            switch rule.GetSeverity() {
            case SeverityError:
                errors = append(errors, err)
            case SeverityWarning:
                warnings = append(warnings, err.Error())
            case SeverityInfo:
                // Log info messages
                fmt.Printf("Info: %s\n", err.Error())
            }
        }
    }

    // Display warnings
    for _, warning := range warnings {
        fmt.Printf("Warning: %s\n", warning)
    }

    // Return first error if any
    if len(errors) > 0 {
        return fmt.Errorf("flag validation failed: %w", errors[0])
    }

    return nil
}

// RegisterFlags registers all flags from groups
func (b *BaseFlagValidator) RegisterFlags(cmd *cobra.Command) {
    for _, group := range b.groups {
        group.RegisterFlags(cmd)
    }
}

// GetValidationRules returns all validation rules
func (b *BaseFlagValidator) GetValidationRules() []ValidationRule {
    return b.rules
}
```

### Step 3: Implement Validation Rules

**File**: `cmd/flags/validators/required.go`

```go
package validators

import (
    "context"
    "fmt"
    "reflect"
    "github.com/ArjenSchwarz/fog/cmd/flags"
)

// RequiredFieldRule validates that required fields are not empty
type RequiredFieldRule struct {
    FieldName   string
    GetValue    func(flags.FlagValidator) interface{}
    Description string
}

// NewRequiredFieldRule creates a new required field rule
func NewRequiredFieldRule(fieldName string, getValue func(flags.FlagValidator) interface{}) *RequiredFieldRule {
    return &RequiredFieldRule{
        FieldName:   fieldName,
        GetValue:    getValue,
        Description: fmt.Sprintf("%s is required", fieldName),
    }
}

// Validate checks if the field has a value
func (r *RequiredFieldRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    value := r.GetValue(flags)

    if isEmpty(value) {
        return fmt.Errorf("required field '%s' is missing or empty", r.FieldName)
    }

    return nil
}

// GetDescription returns the rule description
func (r *RequiredFieldRule) GetDescription() string {
    return r.Description
}

// GetSeverity returns the rule severity
func (r *RequiredFieldRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}

// isEmpty checks if a value is considered empty
func isEmpty(value interface{}) bool {
    if value == nil {
        return true
    }

    v := reflect.ValueOf(value)
    switch v.Kind() {
    case reflect.String:
        return v.String() == ""
    case reflect.Slice, reflect.Array:
        return v.Len() == 0
    case reflect.Map:
        return v.Len() == 0
    case reflect.Ptr, reflect.Interface:
        return v.IsNil()
    default:
        return false
    }
}
```

**File**: `cmd/flags/validators/dependencies.go`

```go
package validators

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/flags"
)

// DependencyRule validates that when one flag is set, dependent flags are also set
type DependencyRule struct {
    TriggerField    string
    DependentFields []string
    GetTrigger      func(flags.FlagValidator) interface{}
    GetDependent    func(flags.FlagValidator, string) interface{}
    Description     string
}

// NewDependencyRule creates a new dependency rule
func NewDependencyRule(
    triggerField string,
    dependentFields []string,
    getTrigger func(flags.FlagValidator) interface{},
    getDependent func(flags.FlagValidator, string) interface{},
) *DependencyRule {
    return &DependencyRule{
        TriggerField:    triggerField,
        DependentFields: dependentFields,
        GetTrigger:      getTrigger,
        GetDependent:    getDependent,
        Description:     fmt.Sprintf("when %s is set, %v must also be set", triggerField, dependentFields),
    }
}

// Validate checks dependencies
func (d *DependencyRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    triggerValue := d.GetTrigger(flags)

    // If trigger field is not set, no validation needed
    if isEmpty(triggerValue) {
        return nil
    }

    // Check all dependent fields
    for _, depField := range d.DependentFields {
        depValue := d.GetDependent(flags, depField)
        if isEmpty(depValue) {
            return fmt.Errorf("when '%s' is specified, '%s' must also be provided",
                d.TriggerField, depField)
        }
    }

    return nil
}

// GetDescription returns the rule description
func (d *DependencyRule) GetDescription() string {
    return d.Description
}

// GetSeverity returns the rule severity
func (d *DependencyRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}
```

**File**: `cmd/flags/validators/conflicts.go`

```go
package validators

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/flags"
)

// ConflictRule validates that conflicting flags are not used together
type ConflictRule struct {
    ConflictingFields []string
    GetValue          func(flags.FlagValidator, string) interface{}
    Description       string
}

// NewConflictRule creates a new conflict rule
func NewConflictRule(
    conflictingFields []string,
    getValue func(flags.FlagValidator, string) interface{},
) *ConflictRule {
    return &ConflictRule{
        ConflictingFields: conflictingFields,
        GetValue:          getValue,
        Description:       fmt.Sprintf("flags %v cannot be used together", conflictingFields),
    }
}

// Validate checks for conflicts
func (c *ConflictRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    setFields := make([]string, 0)

    // Check which fields are set
    for _, field := range c.ConflictingFields {
        value := c.GetValue(flags, field)
        if !isEmpty(value) {
            setFields = append(setFields, field)
        }
    }

    // If more than one field is set, it's a conflict
    if len(setFields) > 1 {
        return fmt.Errorf("conflicting flags specified: %v (only one allowed)", setFields)
    }

    return nil
}

// GetDescription returns the rule description
func (c *ConflictRule) GetDescription() string {
    return c.Description
}

// GetSeverity returns the rule severity
func (c *ConflictRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}
```

**File**: `cmd/flags/validators/format.go`

```go
package validators

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "github.com/ArjenSchwarz/fog/cmd/flags"
)

// FileExistsRule validates that a file exists
type FileExistsRule struct {
    FieldName   string
    GetValue    func(flags.FlagValidator) string
    Required    bool
    Description string
}

// NewFileExistsRule creates a new file exists rule
func NewFileExistsRule(fieldName string, getValue func(flags.FlagValidator) string, required bool) *FileExistsRule {
    return &FileExistsRule{
        FieldName:   fieldName,
        GetValue:    getValue,
        Required:    required,
        Description: fmt.Sprintf("%s must be a valid file path", fieldName),
    }
}

// Validate checks if the file exists
func (f *FileExistsRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    value := f.GetValue(flags)

    if value == "" {
        if f.Required {
            return fmt.Errorf("file path for '%s' is required", f.FieldName)
        }
        return nil
    }

    if _, err := os.Stat(value); os.IsNotExist(err) {
        return fmt.Errorf("file '%s' specified for '%s' does not exist", value, f.FieldName)
    }

    return nil
}

// GetDescription returns the rule description
func (f *FileExistsRule) GetDescription() string {
    return f.Description
}

// GetSeverity returns the rule severity
func (f *FileExistsRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}

// FileExtensionRule validates file extensions
type FileExtensionRule struct {
    FieldName           string
    GetValue            func(flags.FlagValidator) string
    AllowedExtensions   []string
    Description         string
}

// NewFileExtensionRule creates a new file extension rule
func NewFileExtensionRule(fieldName string, getValue func(flags.FlagValidator) string, allowedExtensions []string) *FileExtensionRule {
    return &FileExtensionRule{
        FieldName:         fieldName,
        GetValue:          getValue,
        AllowedExtensions: allowedExtensions,
        Description:       fmt.Sprintf("%s must have one of these extensions: %v", fieldName, allowedExtensions),
    }
}

// Validate checks the file extension
func (f *FileExtensionRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    value := f.GetValue(flags)

    if value == "" {
        return nil
    }

    ext := strings.ToLower(filepath.Ext(value))
    for _, allowedExt := range f.AllowedExtensions {
        if ext == allowedExt {
            return nil
        }
    }

    return fmt.Errorf("file '%s' has invalid extension '%s', allowed: %v",
        value, ext, f.AllowedExtensions)
}

// GetDescription returns the rule description
func (f *FileExtensionRule) GetDescription() string {
    return f.Description
}

// GetSeverity returns the rule severity
func (f *FileExtensionRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityWarning
}

// RegexRule validates values against a regular expression
type RegexRule struct {
    FieldName   string
    GetValue    func(flags.FlagValidator) string
    Pattern     *regexp.Regexp
    ErrorMsg    string
    Description string
}

// NewRegexRule creates a new regex validation rule
func NewRegexRule(fieldName string, getValue func(flags.FlagValidator) string, pattern string, errorMsg string) *RegexRule {
    regex, err := regexp.Compile(pattern)
    if err != nil {
        panic(fmt.Sprintf("Invalid regex pattern: %s", pattern))
    }

    return &RegexRule{
        FieldName:   fieldName,
        GetValue:    getValue,
        Pattern:     regex,
        ErrorMsg:    errorMsg,
        Description: fmt.Sprintf("%s must match pattern: %s", fieldName, pattern),
    }
}

// Validate checks the regex pattern
func (r *RegexRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    value := r.GetValue(flags)

    if value == "" {
        return nil
    }

    if !r.Pattern.MatchString(value) {
        return fmt.Errorf("'%s' for field '%s': %s", value, r.FieldName, r.ErrorMsg)
    }

    return nil
}

// GetDescription returns the rule description
func (r *RegexRule) GetDescription() string {
    return r.Description
}

// GetSeverity returns the rule severity
func (r *RegexRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}
```

### Step 4: Implement Enhanced Deployment Flags

**File**: `cmd/flags/groups/deployment.go`

```go
package groups

import (
    "context"
    "github.com/ArjenSchwarz/fog/cmd/flags"
    "github.com/ArjenSchwarz/fog/cmd/flags/validators"
    "github.com/spf13/cobra"
)

// DeploymentFlags represents deployment command flags with enhanced validation
type DeploymentFlags struct {
    *flags.BaseFlagValidator

    // Core deployment flags
    StackName       string
    Template        string
    Parameters      string
    Tags            string
    Bucket          string
    ChangesetName   string
    DeploymentFile  string

    // Deployment mode flags
    Dryrun          bool
    NonInteractive  bool
    CreateChangeset bool
    DeployChangeset bool
    DefaultTags     bool
}

// NewDeploymentFlags creates a new deployment flags group
func NewDeploymentFlags() *DeploymentFlags {
    df := &DeploymentFlags{
        BaseFlagValidator: flags.NewBaseFlagValidator(),
        DefaultTags:       true,
    }

    df.setupValidationRules()
    return df
}

// setupValidationRules configures all validation rules for deployment flags
func (df *DeploymentFlags) setupValidationRules() {
    // Stack name is always required
    df.AddRule(validators.NewRequiredFieldRule("stackname",
        func(flags flags.FlagValidator) interface{} {
            return df.StackName
        }))

    // Stack name format validation
    df.AddRule(validators.NewRegexRule("stackname",
        func(flags flags.FlagValidator) string {
            return df.StackName
        },
        `^[a-zA-Z][a-zA-Z0-9-]*$`,
        "stack name must start with a letter and contain only letters, numbers, and hyphens"))

    // Deployment file conflicts with individual parameters
    df.AddRule(validators.NewConflictRule(
        []string{"deployment-file", "template"},
        func(flags flags.FlagValidator, field string) interface{} {
            switch field {
            case "deployment-file":
                return df.DeploymentFile
            case "template":
                return df.Template
            default:
                return ""
            }
        }))

    df.AddRule(validators.NewConflictRule(
        []string{"deployment-file", "parameters"},
        func(flags flags.FlagValidator, field string) interface{} {
            switch field {
            case "deployment-file":
                return df.DeploymentFile
            case "parameters":
                return df.Parameters
            default:
                return ""
            }
        }))

    df.AddRule(validators.NewConflictRule(
        []string{"deployment-file", "tags"},
        func(flags flags.FlagValidator, field string) interface{} {
            switch field {
            case "deployment-file":
                return df.DeploymentFile
            case "tags":
                return df.Tags
            default:
                return ""
            }
        }))

    // Template is required when not using deployment file
    df.AddRule(&TemplateRequiredRule{df})

    // File existence validation
    df.AddRule(validators.NewFileExistsRule("template",
        func(flags flags.FlagValidator) string {
            return df.Template
        }, false))

    df.AddRule(validators.NewFileExistsRule("deployment-file",
        func(flags flags.FlagValidator) string {
            return df.DeploymentFile
        }, false))

    // Template file extension validation
    df.AddRule(validators.NewFileExtensionRule("template",
        func(flags flags.FlagValidator) string {
            return df.Template
        },
        []string{".yaml", ".yml", ".json", ".template", ".tmpl"}))

    // Deployment file extension validation
    df.AddRule(validators.NewFileExtensionRule("deployment-file",
        func(flags flags.FlagValidator) string {
            return df.DeploymentFile
        },
        []string{".yaml", ".yml", ".json"}))

    // Create and deploy changeset are mutually exclusive
    df.AddRule(&ChangesetModeRule{df})
}

// RegisterFlags registers all deployment flags
func (df *DeploymentFlags) RegisterFlags(cmd *cobra.Command) {
    cmd.Flags().StringVarP(&df.StackName, "stackname", "n", "", "The name for the stack")
    cmd.Flags().StringVarP(&df.Template, "template", "f", "", "The filename for the template")
    cmd.Flags().StringVarP(&df.Parameters, "parameters", "p", "", "The file(s) containing the parameter values, comma-separated for multiple")
    cmd.Flags().StringVarP(&df.Tags, "tags", "t", "", "The file(s) containing the tags, comma-separated for multiple")
    cmd.Flags().StringVarP(&df.Bucket, "bucket", "b", "", "The S3 bucket where the template should be uploaded to (optional)")
    cmd.Flags().StringVarP(&df.ChangesetName, "changeset", "c", "", "The name of the changeset, when not provided it will be autogenerated")
    cmd.Flags().StringVarP(&df.DeploymentFile, "deployment-file", "d", "", "The file to use for the deployment")
    cmd.Flags().BoolVar(&df.Dryrun, "dry-run", false, "Do a dry run: create the changeset and immediately delete")
    cmd.Flags().BoolVar(&df.NonInteractive, "non-interactive", false, "Run in non-interactive mode: automatically approve the changeset and deploy")
    cmd.Flags().BoolVar(&df.CreateChangeset, "create-changeset", false, "Only create a change set")
    cmd.Flags().BoolVar(&df.DeployChangeset, "deploy-changeset", false, "Deploy a specific change set")
    cmd.Flags().BoolVar(&df.DefaultTags, "default-tags", true, "Add any default tags that are specified in your config file")
}

// GetName returns the group name
func (df *DeploymentFlags) GetName() string {
    return "deployment"
}

// GetFlags returns the flag definitions
func (df *DeploymentFlags) GetFlags() []flags.FlagDefinition {
    return []flags.FlagDefinition{
        {Name: "stackname", Shorthand: "n", Description: "The name for the stack", Required: true, FlagType: flags.StringFlag},
        {Name: "template", Shorthand: "f", Description: "The filename for the template", FlagType: flags.StringFlag},
        {Name: "parameters", Shorthand: "p", Description: "The file(s) containing the parameter values", FlagType: flags.StringFlag},
        {Name: "tags", Shorthand: "t", Description: "The file(s) containing the tags", FlagType: flags.StringFlag},
        {Name: "bucket", Shorthand: "b", Description: "The S3 bucket for template upload", FlagType: flags.StringFlag},
        {Name: "changeset", Shorthand: "c", Description: "The name of the changeset", FlagType: flags.StringFlag},
        {Name: "deployment-file", Shorthand: "d", Description: "The file to use for deployment", FlagType: flags.StringFlag},
        {Name: "dry-run", Description: "Do a dry run", FlagType: flags.BoolFlag},
        {Name: "non-interactive", Description: "Run in non-interactive mode", FlagType: flags.BoolFlag},
        {Name: "create-changeset", Description: "Only create a change set", FlagType: flags.BoolFlag},
        {Name: "deploy-changeset", Description: "Deploy a specific change set", FlagType: flags.BoolFlag},
        {Name: "default-tags", Description: "Add default tags", DefaultValue: true, FlagType: flags.BoolFlag},
    }
}

// Custom validation rules

// TemplateRequiredRule validates that template is provided when not using deployment file
type TemplateRequiredRule struct {
    flags *DeploymentFlags
}

func (t *TemplateRequiredRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    if t.flags.DeploymentFile == "" && t.flags.Template == "" {
        return fmt.Errorf("either --template or --deployment-file must be specified")
    }
    return nil
}

func (t *TemplateRequiredRule) GetDescription() string {
    return "template or deployment file is required"
}

func (t *TemplateRequiredRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}

// ChangesetModeRule validates changeset mode flags
type ChangesetModeRule struct {
    flags *DeploymentFlags
}

func (c *ChangesetModeRule) Validate(ctx context.Context, flags flags.FlagValidator) error {
    if c.flags.CreateChangeset && c.flags.DeployChangeset {
        return fmt.Errorf("--create-changeset and --deploy-changeset cannot be used together")
    }
    return nil
}

func (c *ChangesetModeRule) GetDescription() string {
    return "changeset creation and deployment modes are mutually exclusive"
}

func (c *ChangesetModeRule) GetSeverity() flags.ValidationSeverity {
    return flags.SeverityError
}
```

### Step 5: Create Flag Validation Middleware

**File**: `cmd/flags/middleware/validation.go`

```go
package middleware

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/flags"
    "github.com/ArjenSchwarz/fog/cmd/registry"
)

// FlagValidationMiddleware provides enhanced flag validation
type FlagValidationMiddleware struct {
    validator flags.FlagValidator
}

// NewFlagValidationMiddleware creates new flag validation middleware
func NewFlagValidationMiddleware(validator flags.FlagValidator) *FlagValidationMiddleware {
    return &FlagValidationMiddleware{
        validator: validator,
    }
}

// Execute runs the flag validation middleware
func (m *FlagValidationMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
    // Create validation context
    validationCtx := context.WithValue(ctx, "validator", m.validator)

    // Run validation
    if err := m.validator.Validate(validationCtx); err != nil {
        return fmt.Errorf("flag validation failed: %w", err)
    }

    // Continue to next middleware/handler
    return next(ctx)
}

// Implement registry.Middleware interface
var _ registry.Middleware = (*FlagValidationMiddleware)(nil)
```

### Step 6: Update Deploy Command to Use Enhanced Flags

**File**: `cmd/commands/deploy/command.go` (update from previous tasks)

```go
package deploy

import (
    "github.com/ArjenSchwarz/fog/cmd/flags/groups"
    "github.com/ArjenSchwarz/fog/cmd/flags/middleware"
    "github.com/ArjenSchwarz/fog/cmd/registry"
    "github.com/spf13/cobra"
)

// CommandBuilder builds the deploy command with enhanced flag validation
type CommandBuilder struct {
    *registry.BaseCommandBuilder
    flags *groups.DeploymentFlags
}

// NewCommandBuilder creates a new deploy command builder with enhanced flags
func NewCommandBuilder() *CommandBuilder {
    flags := groups.NewDeploymentFlags()

    builder := registry.NewBaseCommandBuilder(
        "deploy",
        "Deploy a CloudFormation stack",
        `deploy allows you to deploy a CloudFormation stack

It does so by creating a ChangeSet and then asking you for approval before continuing. You can automatically approve or only create or deploy a changeset by using flags.

A name for the changeset will automatically be generated based on your preferred name, but can be overwritten as well.

When providing tag and/or parameter files, you can add multiple files for each. These are parsed in the order provided and later values will override earlier ones.

Examples:

  fog deploy --stackname testvpc --template basicvpc --parameters vpc-private-only --tags "../globaltags/project,dev"
  fog deploy --stackname fails3 --template fails3 --non-interactive
  fog deploy --stackname myvpc --template basicvpc --parameters vpc-public --tags "../globaltags/project,dev" --config testconf/fog.yaml`,
    )

    handler := NewHandler(flags)

    // Add flag validation middleware
    flagValidationMiddleware := middleware.NewFlagValidationMiddleware(flags)

    return &CommandBuilder{
        BaseCommandBuilder: builder.
            WithHandler(handler).
            WithValidator(flags).
            WithMiddleware(flagValidationMiddleware),
        flags: flags,
    }
}

// BuildCommand creates the cobra command
func (b *CommandBuilder) BuildCommand() *cobra.Command {
    return b.BaseCommandBuilder.BuildCommand()
}

// GetHandler returns the command handler
func (b *CommandBuilder) GetHandler() registry.CommandHandler {
    return b.BaseCommandBuilder.GetHandler()
}
```

## Files to Create/Modify

### New Files
- `cmd/flags/interfaces.go`
- `cmd/flags/base.go`
- `cmd/flags/validators/required.go`
- `cmd/flags/validators/dependencies.go`
- `cmd/flags/validators/conflicts.go`
- `cmd/flags/validators/format.go`
- `cmd/flags/validators/aws.go`
- `cmd/flags/groups/deployment.go`
- `cmd/flags/groups/drift.go`
- `cmd/flags/groups/stack.go`
- `cmd/flags/groups/common.go`
- `cmd/flags/middleware/validation.go`
- `cmd/flags/middleware/preprocessing.go`

### Modified Files
- `cmd/commands/deploy/command.go` - Use enhanced flags
- `cmd/commands/deploy/handler.go` - Update to use new flag structure
- `cmd/flaggroups.go` - Deprecated, mark for removal
- `cmd/commands/deploy/flags.go` - Replace with flag group reference

## Testing Strategy

### Unit Tests
- Test individual validation rules
- Test flag group registration
- Test validation middleware
- Test conflict and dependency detection
- Test file validation rules

### Integration Tests
- Test complete flag validation flow
- Test error message quality and clarity
- Test validation context handling
- Test middleware integration

### Test Files to Create
- `cmd/flags/validators/required_test.go`
- `cmd/flags/validators/dependencies_test.go`
- `cmd/flags/validators/conflicts_test.go`
- `cmd/flags/validators/format_test.go`
- `cmd/flags/groups/deployment_test.go`
- `cmd/flags/middleware/validation_test.go`

## Success Criteria

### Functional Requirements
- [ ] Enhanced flag validation with clear error messages
- [ ] Dependency and conflict checking works correctly
- [ ] File validation (existence, extensions) functions
- [ ] Validation middleware integrates with command structure
- [ ] All existing flag functionality preserved

### Quality Requirements
- [ ] Unit tests cover >90% of validation logic
- [ ] Error messages are user-friendly and actionable
- [ ] Validation rules are easily configurable
- [ ] Performance impact is minimal

### User Experience Requirements
- [ ] Clear validation error messages
- [ ] Helpful suggestions for fixing validation errors
- [ ] Progressive validation (warnings vs errors)
- [ ] Consistent validation behavior across commands

## Migration Timeline

### Phase 1: Foundation
- Create flag validation framework
- Implement basic validation rules
- Create validation middleware

### Phase 2: Enhanced Deployment Flags
- Implement enhanced deployment flag group
- Add comprehensive validation rules
- Integrate with deploy command

### Phase 3: Other Commands
- Migrate remaining commands to enhanced flags
- Add command-specific validation rules
- Remove deprecated flag structures

## Dependencies

### Upstream Dependencies
- Task 1: Command Structure Reorganization (provides middleware framework)

### Downstream Dependencies
None - this task enhances existing functionality without breaking changes.

## Risk Mitigation

### Potential Issues
- Breaking changes to existing flag behavior
- Performance overhead from extensive validation
- Complex validation rule interactions

### Mitigation Strategies
- Maintain backward compatibility during transition
- Performance testing for validation overhead
- Clear documentation of validation rules
- Progressive rollout with fallback options
