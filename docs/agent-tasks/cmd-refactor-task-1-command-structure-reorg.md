# Task 1: Command Structure Reorganization

## Objective

Establish a clean separation of concerns and consistent command patterns by reorganizing the command structure and creating a centralized command registry system.

## Current State

### Problems
- Commands are scattered across individual files with inconsistent patterns
- Command registration is mixed with business logic
- Global variables used for command configuration
- No standard interface for command handlers
- Command groups are manually managed in `groups.go`

### Current Command Files
- `deploy.go` - 500+ lines mixing command setup, flag handling, and business logic
- `drift.go` - Similar monolithic structure
- `describe.go`, `describe_changeset.go` - Split describe functionality
- Individual files for each command with varying patterns

## Target State

### Goals
- Consistent command handler interface
- Centralized command registration
- Clear separation between command definition and business logic
- Standardized flag validation and processing
- Modular command structure for easy extension

### New Structure
```
cmd/
├── commands/
│   ├── deploy/
│   │   ├── command.go          # Command definition and setup
│   │   ├── handler.go          # Command handler implementation
│   │   └── flags.go            # Command-specific flag handling
│   ├── drift/
│   │   ├── command.go
│   │   ├── handler.go
│   │   └── flags.go
│   ├── describe/
│   │   ├── command.go
│   │   ├── changeset_handler.go
│   │   ├── stack_handler.go
│   │   └── flags.go
│   └── ... (other commands)
├── registry/
│   ├── registry.go             # Command registration system
│   ├── interfaces.go           # Command interfaces
│   └── builder.go              # Command builder utilities
└── middleware/
    ├── validation.go           # Flag validation middleware
    ├── context.go              # Context setup middleware
    └── logging.go              # Command logging middleware
```

## Prerequisites

None - this is a foundational task.

## Step-by-Step Implementation

### Step 1: Create Command Interfaces

**File**: `cmd/registry/interfaces.go`

```go
package registry

import (
    "context"
    "github.com/spf13/cobra"
)

// CommandHandler defines the interface for all command handlers
type CommandHandler interface {
    Execute(ctx context.Context) error
    ValidateFlags() error
}

// CommandBuilder defines how commands are constructed
type CommandBuilder interface {
    BuildCommand() *cobra.Command
    GetHandler() CommandHandler
}

// FlagValidator defines flag validation interface
type FlagValidator interface {
    Validate() error
    RegisterFlags(cmd *cobra.Command)
}
```

### Step 2: Create Command Registry

**File**: `cmd/registry/registry.go`

```go
package registry

import (
    "fmt"
    "github.com/spf13/cobra"
)

// CommandRegistry manages all available commands
type CommandRegistry struct {
    builders map[string]CommandBuilder
    rootCmd  *cobra.Command
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry(rootCmd *cobra.Command) *CommandRegistry {
    return &CommandRegistry{
        builders: make(map[string]CommandBuilder),
        rootCmd:  rootCmd,
    }
}

// Register adds a command builder to the registry
func (r *CommandRegistry) Register(name string, builder CommandBuilder) error {
    if _, exists := r.builders[name]; exists {
        return fmt.Errorf("command %s already registered", name)
    }

    r.builders[name] = builder
    return nil
}

// BuildAll constructs all registered commands and adds them to root
func (r *CommandRegistry) BuildAll() error {
    for name, builder := range r.builders {
        cmd := builder.BuildCommand()
        if cmd == nil {
            return fmt.Errorf("builder for %s returned nil command", name)
        }

        r.rootCmd.AddCommand(cmd)
    }
    return nil
}
```

### Step 3: Create Command Builder Utilities

**File**: `cmd/registry/builder.go`

```go
package registry

import (
    "context"
    "github.com/spf13/cobra"
)

// BaseCommandBuilder provides common functionality for command builders
type BaseCommandBuilder struct {
    name        string
    short       string
    long        string
    handler     CommandHandler
    validator   FlagValidator
    middlewares []Middleware
}

// Middleware defines command middleware interface
type Middleware interface {
    Execute(ctx context.Context, next func(context.Context) error) error
}

// NewBaseCommandBuilder creates a new base command builder
func NewBaseCommandBuilder(name, short, long string) *BaseCommandBuilder {
    return &BaseCommandBuilder{
        name:        name,
        short:       short,
        long:        long,
        middlewares: make([]Middleware, 0),
    }
}

// WithHandler sets the command handler
func (b *BaseCommandBuilder) WithHandler(handler CommandHandler) *BaseCommandBuilder {
    b.handler = handler
    return b
}

// WithValidator sets the flag validator
func (b *BaseCommandBuilder) WithValidator(validator FlagValidator) *BaseCommandBuilder {
    b.validator = validator
    return b
}

// WithMiddleware adds middleware to the command
func (b *BaseCommandBuilder) WithMiddleware(middleware Middleware) *BaseCommandBuilder {
    b.middlewares = append(b.middlewares, middleware)
    return b
}

// BuildCommand creates the cobra command
func (b *BaseCommandBuilder) BuildCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   b.name,
        Short: b.short,
        Long:  b.long,
        RunE:  b.createRunFunc(),
    }

    if b.validator != nil {
        b.validator.RegisterFlags(cmd)
    }

    return cmd
}

// createRunFunc creates the command run function with middleware chain
func (b *BaseCommandBuilder) createRunFunc() func(*cobra.Command, []string) error {
    return func(cmd *cobra.Command, args []string) error {
        ctx := context.WithValue(context.Background(), "command", cmd)
        ctx = context.WithValue(ctx, "args", args)

        // Create middleware chain
        handler := func(ctx context.Context) error {
            if b.validator != nil {
                if err := b.validator.Validate(); err != nil {
                    return err
                }
            }
            return b.handler.Execute(ctx)
        }

        // Apply middleware in reverse order
        for i := len(b.middlewares) - 1; i >= 0; i-- {
            middleware := b.middlewares[i]
            next := handler
            handler = func(ctx context.Context) error {
                return middleware.Execute(ctx, next)
            }
        }

        return handler(ctx)
    }
}

// GetHandler returns the command handler
func (b *BaseCommandBuilder) GetHandler() CommandHandler {
    return b.handler
}
```

### Step 4: Create Middleware Components

**File**: `cmd/middleware/validation.go`

```go
package middleware

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/registry"
)

// ValidationMiddleware handles flag validation
type ValidationMiddleware struct{}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware() *ValidationMiddleware {
    return &ValidationMiddleware{}
}

// Execute runs the validation middleware
func (m *ValidationMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
    // Validation is now handled by the command builder
    // This middleware can be used for additional validation logic
    return next(ctx)
}
```

**File**: `cmd/middleware/context.go`

```go
package middleware

import (
    "context"
    "github.com/ArjenSchwarz/fog/config"
)

// ContextMiddleware sets up the command context
type ContextMiddleware struct {
    configLoader func() (*config.Config, error)
}

// NewContextMiddleware creates a new context middleware
func NewContextMiddleware(configLoader func() (*config.Config, error)) *ContextMiddleware {
    return &ContextMiddleware{
        configLoader: configLoader,
    }
}

// Execute runs the context middleware
func (m *ContextMiddleware) Execute(ctx context.Context, next func(context.Context) error) error {
    // Load configuration
    cfg, err := m.configLoader()
    if err != nil {
        return err
    }

    // Add config to context
    ctx = context.WithValue(ctx, "config", cfg)

    return next(ctx)
}
```

### Step 5: Refactor Deploy Command

**File**: `cmd/commands/deploy/flags.go`

```go
package deploy

import (
    "fmt"
    "github.com/spf13/cobra"
)

// Flags represents deploy command flags
type Flags struct {
    StackName       string
    Template        string
    Parameters      string
    Tags            string
    Bucket          string
    ChangesetName   string
    Dryrun          bool
    NonInteractive  bool
    CreateChangeset bool
    DeployChangeset bool
    DefaultTags     bool
    DeploymentFile  string
}

// Validate validates the deploy flags
func (f *Flags) Validate() error {
    if f.StackName == "" {
        return fmt.Errorf("stack name is required")
    }

    if f.DeploymentFile != "" && (f.Template != "" || f.Parameters != "" || f.Tags != "") {
        return fmt.Errorf("you can't provide a deployment file and other parameters at the same time")
    }

    return nil
}

// RegisterFlags registers all deploy flags to the given command
func (f *Flags) RegisterFlags(cmd *cobra.Command) {
    cmd.Flags().StringVarP(&f.StackName, "stackname", "n", "", "The name for the stack")
    cmd.Flags().StringVarP(&f.Template, "template", "f", "", "The filename for the template")
    cmd.Flags().StringVarP(&f.Parameters, "parameters", "p", "", "The file(s) containing the parameter values, comma-separated for multiple")
    cmd.Flags().StringVarP(&f.Tags, "tags", "t", "", "The file(s) containing the tags, comma-separated for multiple")
    cmd.Flags().StringVarP(&f.Bucket, "bucket", "b", "", "The S3 bucket where the template should be uploaded to (optional)")
    cmd.Flags().StringVarP(&f.ChangesetName, "changeset", "c", "", "The name of the changeset, when not provided it will be autogenerated")
    cmd.Flags().BoolVar(&f.Dryrun, "dry-run", false, "Do a dry run: create the changeset and immediately delete")
    cmd.Flags().BoolVar(&f.NonInteractive, "non-interactive", false, "Run in non-interactive mode: automatically approve the changeset and deploy")
    cmd.Flags().BoolVar(&f.CreateChangeset, "create-changeset", false, "Only create a change set")
    cmd.Flags().BoolVar(&f.DeployChangeset, "deploy-changeset", false, "Deploy a specific change set")
    cmd.Flags().BoolVar(&f.DefaultTags, "default-tags", true, "Add any default tags that are specified in your config file")
    cmd.Flags().StringVarP(&f.DeploymentFile, "deployment-file", "d", "", "The file to use for the deployment")
}
```

**File**: `cmd/commands/deploy/handler.go`

```go
package deploy

import (
    "context"
    "fmt"
)

// Handler implements the deploy command logic
type Handler struct {
    flags *Flags
    // Services will be injected here in Task 2
}

// NewHandler creates a new deploy command handler
func NewHandler(flags *Flags) *Handler {
    return &Handler{
        flags: flags,
    }
}

// Execute runs the deploy command
func (h *Handler) Execute(ctx context.Context) error {
    // This will be implemented in Task 2 - Business Logic Extraction
    // For now, just delegate to the existing function

    // Get command and args from context
    cmd := ctx.Value("command")
    args := ctx.Value("args").([]string)

    // Call existing deploy function (temporary)
    // This will be replaced with proper service calls in Task 2
    return fmt.Errorf("deploy handler not yet implemented - waiting for Task 2")
}

// ValidateFlags validates the command flags
func (h *Handler) ValidateFlags() error {
    return h.flags.Validate()
}
```

**File**: `cmd/commands/deploy/command.go`

```go
package deploy

import (
    "github.com/ArjenSchwarz/fog/cmd/registry"
    "github.com/spf13/cobra"
)

// CommandBuilder builds the deploy command
type CommandBuilder struct {
    *registry.BaseCommandBuilder
    flags *Flags
}

// NewCommandBuilder creates a new deploy command builder
func NewCommandBuilder() *CommandBuilder {
    flags := &Flags{}

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

    return &CommandBuilder{
        BaseCommandBuilder: builder.WithHandler(handler).WithValidator(flags),
        flags:              flags,
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

### Step 6: Update Root Command Setup

**File**: `cmd/root.go` (modify the init function)

```go
// Update the init function to use the registry
func init() {
    cobra.OnInitialize(initConfig)

    // Create command registry
    registry := registry.NewCommandRegistry(rootCmd)

    // Register commands
    registry.Register("deploy", deploy.NewCommandBuilder())
    // Add other commands as they are refactored

    // Build all commands
    if err := registry.BuildAll(); err != nil {
        log.Fatal(err)
    }

    // Initialize command groups (temporary during transition)
    InitGroups()

    // Add aliases for commonly used commands at the root level
    rootCmd.AddCommand(NewCommandAlias("deploy", "stack deploy", "Alias for 'stack deploy'"))
    // ... rest of the aliases

    // ... rest of the persistent flags setup
}
```

### Step 7: Create Migration Guide

**File**: `cmd/commands/README.md`

```markdown
# Command Structure Migration Guide

## Overview
This directory contains the new command structure for the fog CLI application.

## Structure
Each command is organized in its own directory with:
- `command.go` - Command builder and definition
- `handler.go` - Business logic handler
- `flags.go` - Flag definitions and validation

## Migration Status
- [x] Deploy command - Refactored with new structure
- [ ] Drift command - TODO
- [ ] Describe command - TODO
- [ ] Dependencies command - TODO
- [ ] Exports command - TODO
- [ ] History command - TODO
- [ ] Report command - TODO
- [ ] Resources command - TODO

## Adding New Commands
1. Create directory under `cmd/commands/`
2. Implement the three required files
3. Register in `cmd/root.go`
4. Add tests in `cmd/testing/`
```

## Files to Create/Modify

### New Files
- `cmd/registry/interfaces.go`
- `cmd/registry/registry.go`
- `cmd/registry/builder.go`
- `cmd/middleware/validation.go`
- `cmd/middleware/context.go`
- `cmd/commands/deploy/command.go`
- `cmd/commands/deploy/handler.go`
- `cmd/commands/deploy/flags.go`
- `cmd/commands/README.md`

### Modified Files
- `cmd/root.go` - Update init function to use registry
- `cmd/groups.go` - Add compatibility layer during transition

## Testing Strategy

### Unit Tests
- Test command registration system
- Test middleware execution order
- Test flag validation logic
- Test command builder functionality

### Integration Tests
- Test complete command execution flow
- Test middleware chain execution
- Verify backward compatibility with existing commands

### Test Files to Create
- `cmd/registry/registry_test.go`
- `cmd/registry/builder_test.go`
- `cmd/middleware/validation_test.go`
- `cmd/commands/deploy/handler_test.go`

## Success Criteria

### Functional Requirements
- [ ] Deploy command works with new structure
- [ ] All existing flags and functionality preserved
- [ ] Command registration system functional
- [ ] Middleware chain executes correctly

### Quality Requirements
- [ ] Unit tests cover >90% of new code
- [ ] Integration tests verify command execution
- [ ] Documentation explains new structure
- [ ] Code follows established patterns

### Performance Requirements
- [ ] Command startup time not degraded
- [ ] Memory usage remains stable
- [ ] No regression in execution time

## Migration Timeline

### Phase 1 (Current Task)
- Implement command structure framework
- Refactor deploy command as example
- Create documentation and tests

### Phase 2 (Future Tasks)
- Migrate remaining commands
- Remove old command structure
- Complete integration testing

## Dependencies

### Upstream Dependencies
None - this is a foundational task.

### Downstream Dependencies
- Task 2: Business Logic Extraction (will use new command structure)
- Task 5: Error Handling (will integrate with middleware)
- Task 6: Testing Infrastructure (will use new testable structure)

## Risk Mitigation

### Potential Issues
- Breaking changes to existing command behavior
- Performance degradation from middleware overhead
- Complexity in command registration

### Mitigation Strategies
- Maintain backward compatibility layer
- Performance testing for middleware impact
- Clear documentation and examples
- Gradual migration approach
