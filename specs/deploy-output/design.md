# Deploy Output Feature - Design Document

## Overview

This document details the design for adding multi-format output support to the `fog deploy` command. The feature enables users to specify output formats (JSON, CSV, YAML, Markdown, table) for deployment results while maintaining real-time progress visibility through proper stream separation.

The implementation follows Unix conventions by separating progress output (stderr) from data output (stdout), enabling users to redirect final results while still seeing deployment progress in the terminal.

## Architecture

### Output Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                      fog deploy command                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │ Prepare Deployment│
                    └──────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────┐
        │         Create Changeset                │
        │  (capture data for final output)        │
        └─────────────────────────────────────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
              ▼                               ▼
     ┌────────────────┐            ┌──────────────────┐
     │   Dry-Run?     │            │  Create-Changeset│
     └────────────────┘            │      Mode?       │
              │                    └──────────────────┘
              │ Yes                         │ Yes
              ▼                             ▼
     ┌────────────────────────────────────────────────┐
     │  Output changeset to stdout                    │
     │  (reuse describe changeset builder)            │
     └────────────────────────────────────────────────┘
              │                             │
              └──────────────┬──────────────┘
                             │
                             ▼
                          [EXIT]

              │ No
              ▼
     ┌────────────────────────────────────────┐
     │  DEPLOYMENT PHASE                      │
     │                                        │
     │  ┌──────────────────────────────────┐ │
     │  │ Progress Output → stderr         │ │
     │  │ (unless --quiet)                 │ │
     │  │                                  │ │
     │  │ • Stack info (table format)      │ │
     │  │ • Changeset overview             │ │
     │  │ • Streaming events               │ │
     │  │ • Interactive prompts            │ │
     │  └──────────────────────────────────┘ │
     │                                        │
     │  ┌──────────────────────────────────┐ │
     │  │ Execute Changeset                │ │
     │  └──────────────────────────────────┘ │
     └────────────────────────────────────────┘
                    │
        ┌───────────┴───────────┐
        │                       │
        ▼                       ▼
   [SUCCESS]              [FAILURE/NO-CHANGES]
        │                       │
        ▼                       ▼
┌──────────────┐        ┌──────────────┐
│ Capture:     │        │ Capture:     │
│ • Final stack│        │ • Error info │
│ • Outputs    │        │ • Stack state│
│ • Timing     │        │ • Failed res │
└──────────────┘        └──────────────┘
        │                       │
        └───────────┬───────────┘
                    │
                    ▼
           ┌─────────────────┐
           │  os.Stderr.Sync()│
           │  (flush stderr)  │
           └─────────────────┘
                    │
                    ▼
        ┌────────────────────────────┐
        │ Format final output        │
        │ (user's --output format)   │
        └────────────────────────────┘
                    │
                    ▼
        ┌────────────────────────────┐
        │ Write to stdout            │
        │ (or --file if specified)   │
        └────────────────────────────┘
                    │
                    ▼
                 [EXIT]


Stream Separation Examples:
═══════════════════════════════════════════════════════════════

1. Default behavior (table format):
   $ fog deploy --template stack.yaml

   stderr: [Progress table with colored output]
   stdout: [Final deployment summary in table format]

2. JSON output with redirection:
   $ fog deploy --template stack.yaml --output json > result.json

   stderr: [Progress table visible in terminal]
   stdout: [JSON data → redirected to result.json]

3. Quiet mode for CI/CD:
   $ fog deploy --template stack.yaml --output json --quiet > result.json

   stderr: [Empty - no progress output]
   stdout: [JSON data → redirected to result.json]

4. Separate progress and data:
   $ fog deploy --template stack.yaml --output yaml > data.yaml 2> progress.log

   stderr: [Progress table → redirected to progress.log]
   stdout: [YAML data → redirected to data.yaml]

5. File output flag:
   $ fog deploy --template stack.yaml --output json --file result.json

   stderr: [Progress table visible in terminal]
   stdout: [Empty - data written to result.json via --file]
```

### High-Level Design

The implementation uses a dual-output architecture:

1. **Streaming Progress Output (stderr)**
   - Real-time deployment progress using table format
   - Stack information and changeset overview
   - Interactive prompts
   - Always active unless `--quiet` mode is enabled

2. **Final Formatted Output (stdout)**
   - Generated after deployment completion
   - Uses user-specified format (JSON, CSV, YAML, Markdown, table)
   - Contains deployment summary with all relevant data
   - Can be redirected to files for programmatic processing

### Stream Separation Strategy

Based on Decision D1, all progress and diagnostic information routes to stderr, while final formatted output routes to stdout. This enables standard Unix redirection patterns:

```bash
# See progress in terminal, save results to file
fog deploy --template stack.yaml --output json > result.json

# Suppress progress, capture only data
fog deploy --template stack.yaml --output json 2>/dev/null

# Quiet mode for CI/CD (progress suppressed)
fog deploy --template stack.yaml --output json --quiet > result.json
```

## Components and Interfaces

### 1. Output Writers

The design leverages go-output v2.6.0's dual-writer capability:

```go
// Stderr Writer for Progress Output
stderrWriter := output.NewStderrWriter()
stderrOptions := []output.OutputOption{
    output.WithFormat(output.Table()),  // Always table format
    output.WithWriter(stderrWriter),
}

// Stdout Writer for Final Output
stdoutWriter := output.NewStdoutWriter()
stdoutOptions := []output.OutputOption{
    output.WithFormat(getUserFormat()),  // User's requested format
    output.WithWriter(stdoutWriter),
}
```

### 2. DeployFlags Enhancement

Add new flag to existing structure in `cmd/flaggroups.go`:

```go
type DeployFlags struct {
    // ... existing fields ...
    Quiet bool  // New: suppress all stderr output
}
```

Flag registration in `cmd/deploy.go`:

```go
func init() {
    // ... existing flags ...
    deployCmd.Flags().BoolVar(&flags.Quiet, "quiet", false, "Suppress progress output (stderr), show only final result")
}
```

### 3. Deployment State Tracking

Enhance `DeployInfo` struct in `lib/stacks.go` to capture data needed for final output:

```go
type DeployInfo struct {
    // ... existing fields ...

    // New fields for final output
    CapturedChangeset *ChangesetInfo  // Captured immediately after creation
    FinalStackState   *types.Stack     // Final state after deployment
    DeploymentError   error            // Error details if deployment failed
    DeploymentStart   time.Time        // Deployment start timestamp
    DeploymentEnd     time.Time        // Deployment completion timestamp
}
```

### 4. Progress Output Functions

Modified functions that currently write to stdout will use stderr with TTY detection:

```go
// Helper function to create stderr output with TTY detection
func createStderrOutput() *output.Output {
    opts := []output.OutputOption{
        output.WithFormat(output.Table()),
        output.WithWriter(output.NewStderrWriter()),
    }

    // Only add colors and emojis if stderr is a TTY
    // When stderr is redirected to a file, avoid ANSI codes
    if isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd()) {
        opts = append(opts,
            output.WithTransformers(
                &output.EnhancedEmojiTransformer{},
                &output.EnhancedColorTransformer{},
            ),
        )
    }

    return output.NewOutput(opts...)
}

// Usage in showEvents()
func showEvents(deployment *DeployInfo, quiet bool) error {
    if quiet {
        return nil  // No output in quiet mode
    }

    out := createStderrOutput()
    // ... render events to stderr ...
}
```

**Note:** Requires `github.com/mattn/go-isatty` dependency for TTY detection. This prevents polluting log files with ANSI escape codes when stderr is redirected.

### 5. Final Output Builders

Four new builder functions for different deployment outcomes:

#### 5.1 Dry-Run and Create-Changeset Output

Reuses existing changeset rendering logic from `cmd/describe_changeset.go`:

```go
func outputDryRunResult(deployment *DeployInfo, awsConfig config.AWSConfig) error {
    // Flush stderr before stdout output to ensure clean separation
    // Note: This is best-effort ordering, not atomic. In practice works 99.9% of the time.
    os.Stderr.Sync()

    // Reuse existing buildAndRenderChangeset function
    // Signature: buildAndRenderChangeset(changeset ChangesetInfo, deployment DeployInfo, awsConfig AWSConfig)
    // Note: This function internally calls settings.GetOutputOptions() which uses stdout by default
    buildAndRenderChangeset(*deployment.CapturedChangeset, *deployment, awsConfig)

    return nil
}
```

**Note on buildAndRenderChangeset:**
- Actual signature: `func buildAndRenderChangeset(changeset lib.ChangesetInfo, deployment lib.DeployInfo, awsConfig config.AWSConfig)`
- Calls `os.Exit(1)` on error (acceptable for this context - deployment has already succeeded/failed)
- Calls `settings.GetOutputOptions()` internally, so it respects user's output format
- Returns no error (uses os.Exit directly)

#### 5.2 Successful Deployment Output

```go
type DeploymentSummary struct {
    Status          string
    StackARN        string
    StackName       string
    ChangesetID     string
    PlannedChanges  []ChangesetChanges
    StackOutputs    []map[string]string
    StartTime       time.Time
    EndTime         time.Time
    Duration        string
}

func outputSuccessResult(deployment *DeployInfo) error {
    os.Stderr.Sync()
    fmt.Println("\n=== Deployment Summary ===")

    summary := DeploymentSummary{
        Status:         deployment.FinalStackState.StackStatus,
        StackARN:       deployment.StackArn,
        StackName:      deployment.StackName,
        ChangesetID:    deployment.CapturedChangeset.ID,
        PlannedChanges: deployment.CapturedChangeset.Changes,
        StackOutputs:   extractOutputs(deployment.FinalStackState),
        StartTime:      deployment.DeploymentStart,
        EndTime:        deployment.DeploymentEnd,
        Duration:       deployment.DeploymentEnd.Sub(deployment.DeploymentStart).String(),
    }

    doc := output.New().
        Table("Deployment Summary", []map[string]any{
            {
                "Status":      summary.Status,
                "Stack ARN":   summary.StackARN,
                "Changeset":   summary.ChangesetID,
                "Start Time":  summary.StartTime.Format(time.RFC3339),
                "End Time":    summary.EndTime.Format(time.RFC3339),
                "Duration":    summary.Duration,
            },
        }, output.WithKeys("Status", "Stack ARN", "Changeset", "Start Time", "End Time", "Duration")).
        Table("Planned Changes", convertChangesToMaps(summary.PlannedChanges),
            output.WithKeys("Action", "LogicalID", "Type", "ResourceID", "Replacement")).
        Table("Stack Outputs", summary.StackOutputs,
            output.WithKeys("OutputKey", "OutputValue", "Description")).
        Build()

    out := output.NewOutput(settings.GetOutputOptions()...)
    return out.Render(context.Background(), doc)
}
```

#### 5.3 No-Changes Output

```go
type NoChangesResult struct {
    Message     string
    StackName   string
    StackStatus string
    StackARN    string
    LastUpdated time.Time
}

func outputNoChangesResult(deployment *DeployInfo) error {
    os.Stderr.Sync()
    fmt.Println("\n=== Deployment Summary ===")

    result := NoChangesResult{
        Message:     "No changes to deploy - stack is already up to date",
        StackName:   deployment.StackName,
        StackStatus: string(deployment.RawStack.StackStatus),
        StackARN:    deployment.StackArn,
        LastUpdated: *deployment.RawStack.LastUpdatedTime,
    }

    doc := output.New().
        Text(result.Message).
        Table("Stack Information", []map[string]any{
            {
                "Stack Name":   result.StackName,
                "Status":       result.StackStatus,
                "ARN":          result.StackARN,
                "Last Updated": result.LastUpdated.Format(time.RFC3339),
            },
        }, output.WithKeys("Stack Name", "Status", "ARN", "Last Updated")).
        Build()

    out := output.NewOutput(settings.GetOutputOptions()...)
    return out.Render(context.Background(), doc)
}
```

#### 5.4 Failed Deployment Output

```go
type DeploymentFailure struct {
    ErrorMessage    string
    StackStatus     string
    StatusReason    string
    FailedResources []FailedResource
    StackARN        string
    Timestamp       time.Time
}

type FailedResource struct {
    LogicalID      string
    ResourceStatus string
    StatusReason   string
    ResourceType   string
}

func outputFailureResult(deployment *DeployInfo) error {
    os.Stderr.Sync()
    fmt.Println("\n=== Deployment Summary ===")

    failure := DeploymentFailure{
        ErrorMessage:    deployment.DeploymentError.Error(),
        StackStatus:     string(deployment.FinalStackState.StackStatus),
        StatusReason:    aws.ToString(deployment.FinalStackState.StackStatusReason),
        FailedResources: extractFailedResources(deployment),
        StackARN:        deployment.StackArn,
        Timestamp:       time.Now(),
    }

    doc := output.New().
        Text(fmt.Sprintf("Deployment failed: %s", failure.ErrorMessage)).
        Table("Stack Status", []map[string]any{
            {
                "Stack ARN":     failure.StackARN,
                "Status":        failure.StackStatus,
                "Status Reason": failure.StatusReason,
                "Timestamp":     failure.Timestamp.Format(time.RFC3339),
            },
        }, output.WithKeys("Stack ARN", "Status", "Status Reason", "Timestamp")).
        Table("Failed Resources", convertFailedResourcesToMaps(failure.FailedResources),
            output.WithKeys("Logical ID", "Type", "Status", "Reason")).
        Build()

    out := output.NewOutput(settings.GetOutputOptions()...)
    return out.Render(context.Background(), doc)
}
```

### 6. Modified Deploy Flow

The main deployment flow in `cmd/deploy.go` is restructured:

```go
func deployTemplate(cmd *cobra.Command, args []string) error {
    // 1. Prepare deployment (unchanged)
    deployment, err := prepareDeployment(cmd, args)
    if err != nil {
        // Pre-deployment errors go to stderr only
        return err
    }

    deployment.DeploymentStart = time.Now()

    // 2. Run prechecks (unchanged, outputs to stderr)
    if err := runPrechecks(deployment, flags.Quiet); err != nil {
        return err
    }

    // 3. Create and show changeset (modified for data capture)
    if err := createAndCaptureChangeset(deployment, flags.Quiet); err != nil {
        return err
    }

    // 4. Handle dry-run mode
    if deployment.IsDryRun {
        return outputDryRunResult(deployment)
    }

    // 5. Handle create-changeset mode
    if flags.CreateChangeset {
        return outputDryRunResult(deployment)  // Same output, but don't delete changeset
    }

    // 6. Confirm and deploy (modified for quiet mode)
    if err := confirmAndDeployChangeset(deployment, flags.Quiet); err != nil {
        // Check for no-changes scenario
        if isNoChangesError(err) {
            return outputNoChangesResult(deployment)
        }

        // Deployment failed - capture state and output error
        deployment.DeploymentError = err
        deployment.DeploymentEnd = time.Now()

        // Capture final stack state
        if stack, stackErr := getStackState(deployment.StackName); stackErr == nil {
            deployment.FinalStackState = stack
        }

        // Output failure details to stdout
        if outputErr := outputFailureResult(deployment); outputErr != nil {
            fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", outputErr)
        }

        return err  // Return original deployment error
    }

    // 7. Deployment successful - capture final state
    deployment.DeploymentEnd = time.Now()

    stack, err := getStackState(deployment.StackName)
    if err != nil {
        return err
    }
    deployment.FinalStackState = stack

    // 8. Output success summary to stdout
    if err := outputSuccessResult(deployment); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)
    }

    return nil
}
```

### 7. Quiet Mode Implementation

Quiet mode suppresses all stderr output and auto-enables non-interactive mode:

```go
func initializeQuietMode() {
    if flags.Quiet {
        // Auto-enable non-interactive mode
        flags.NonInteractive = true

        // All functions that output to stderr check the quiet flag
        // No global state needed - just parameter passing
    }
}

// Example usage in progress functions
func showDeploymentInfo(deployment *DeployInfo, quiet bool) {
    if quiet {
        return  // Skip all stderr output
    }

    // Normal stderr output...
}
```

### 8. Changeset Data Capture

Modify `createAndShowChangeset()` to capture changeset data:

```go
func createAndCaptureChangeset(deployment *DeployInfo, quiet bool) error {
    // Create changeset (existing logic)
    changeset, err := createChangeset(deployment)
    if err != nil {
        return err
    }

    // Capture changeset immediately for final output
    deployment.CapturedChangeset = changeset
    deployment.Changeset = changeset  // Maintain existing field

    // Show changeset on stderr if not quiet
    if !quiet {
        if err := showChangeset(changeset, deployment.RawStack); err != nil {
            return err
        }
    }

    return nil
}
```

## Data Models

All output formats (JSON, CSV, YAML, Markdown, table) will use existing data structures from the codebase. The structure mirrors the current table-based output to maintain consistency (Decision D7).

**Existing data structures to reuse:**

1. **lib.ChangesetInfo** - For changeset data (already defined in `lib/changesets.go`)
2. **lib.ChangesetChanges** - For individual resource changes
3. **types.Stack** - AWS SDK stack representation (already used throughout)
4. **types.Output** - Stack outputs from AWS SDK

The go-output library will serialize these structures directly to JSON/YAML/CSV/Markdown without additional mapping layers. These are stable types that have been in use for years.

## Error Handling

### Error Categories

1. **Pre-Deployment Errors** (Out of scope for this feature)
   - Authentication failures
   - Invalid templates
   - Validation errors
   - Flag parsing errors
   - Write error messages to stderr only
   - Exit with non-zero code
   - No formatted stdout output

2. **Deployment Errors** (In scope)
   - CloudFormation failures
   - Resource creation/update failures
   - Rollback scenarios
   - Capture error details
   - Output to stdout in requested format
   - Exit with non-zero code

3. **Output Generation Errors** (In scope)
   - Treat as command failures (non-zero exit code)
   - If deployment succeeded but output generation fails, the command fails
   - Rationale: If user requests `--output json` and JSON generation fails, they expected JSON but got nothing - this is a failure
   - Write error to stderr and exit with non-zero code
   - Example: `fog deploy --output json > result.json` with JSON marshal error should fail, not silently succeed

### Error Handling Flow

```go
func deployTemplate(cmd *cobra.Command, args []string) error {
    // ... deployment logic ...

    if deploymentErr != nil {
        // Capture error details
        deployment.DeploymentError = deploymentErr

        // Try to output error in formatted style
        if outputErr := outputFailureResult(deployment); outputErr != nil {
            // Output generation is a critical failure - user requested specific format
            return fmt.Errorf("deployment failed: %w, additionally failed to generate %s output: %v",
                deploymentErr, viper.GetString("output"), outputErr)
        }

        // Return original deployment error (determines exit code)
        return deploymentErr
    }

    // Success path
    deployment.DeploymentEnd = time.Now()

    // ... capture final state ...

    // Output generation failure is a command failure
    if err := outputSuccessResult(deployment); err != nil {
        return fmt.Errorf("deployment succeeded but failed to generate %s output: %w",
            viper.GetString("output"), err)
    }

    return nil  // Only return nil if BOTH deployment AND output succeed
}
```

**Rationale:** If a user runs `fog deploy --output json > result.json` and JSON marshaling fails, they expect a non-zero exit code. Otherwise CI/CD pipelines think the command succeeded when it actually failed to produce the requested output.

### Pre-Deployment Error Validation

Existing error handling for pre-deployment scenarios remains unchanged:

- Changeset creation failures
- Template validation errors
- AWS API errors during preparation
- User cancellation during changeset creation

These scenarios continue to output errors to stderr and exit without producing formatted stdout output.

## Testing Strategy

### Unit Tests

1. **Output Builder Tests** (`cmd/deploy_output_test.go`)
   - Test each output builder function (success, failure, no-changes, dry-run)
   - Verify data structure correctness
   - Test with different output formats
   - Test with missing data fields (error handling)

2. **Stream Separation Tests**
   - Verify stderr output goes to stderr
   - Verify stdout output goes to stdout
   - Test quiet mode suppression
   - Test stream ordering (stderr flushed before stdout)

3. **Quiet Mode Tests**
   - Verify stderr suppression
   - Verify auto-enable of non-interactive mode
   - Verify stdout still produced

### Integration Tests

Integration tests will validate the end-to-end deployment flow with output generation:

```go
//go:build integration
// +build integration

func TestDeploy_SuccessfulWithJSONOutput(t *testing.T) {
    // Test successful deployment with JSON output
    // Verify stdout contains valid JSON
    // Verify stderr contains progress (if not quiet)
}

func TestDeploy_FailureWithCSVOutput(t *testing.T) {
    // Test failed deployment with CSV output
    // Verify stdout contains error details in CSV format
}

func TestDeploy_DryRunWithMarkdownOutput(t *testing.T) {
    // Test dry-run with Markdown output
    // Verify changeset output matches describe changeset
}

func TestDeploy_QuietMode(t *testing.T) {
    // Test quiet mode suppresses stderr
    // Verify only stdout is produced
}

func TestDeploy_NoChanges(t *testing.T) {
    // Test no-changes scenario
    // Verify appropriate message in output
}
```

### Golden File Testing

**Critical for preventing output corruption.** Use golden file tests to ensure output formats remain stable:

```go
func TestOutputSuccessResult_JSON(t *testing.T) {
    deployment := createTestDeployment()

    var buf bytes.Buffer
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := outputSuccessResult(deployment)
    require.NoError(t, err)

    w.Close()
    os.Stdout = oldStdout
    io.Copy(&buf, r)

    // Compare against golden file
    testutil.AssertGolden(t, "testdata/success-output.json", buf.Bytes())
}

func TestOutputSuccessResult_YAML(t *testing.T) {
    // Similar structure for YAML
    testutil.AssertGolden(t, "testdata/success-output.yaml", buf.Bytes())
}

func TestOutputFailureResult_JSON(t *testing.T) {
    // Test failure scenarios
    testutil.AssertGolden(t, "testdata/failure-output.json", buf.Bytes())
}

func TestOutputNoChanges_CSV(t *testing.T) {
    // Test no-changes scenario
    testutil.AssertGolden(t, "testdata/no-changes-output.csv", buf.Bytes())
}
```

**Golden files to create:**
- `testdata/success-output.json`
- `testdata/success-output.yaml`
- `testdata/success-output.csv`
- `testdata/success-output.md`
- `testdata/failure-output.json`
- `testdata/failure-output.yaml`
- `testdata/no-changes-output.json`
- `testdata/dry-run-output.json`

**Why this matters:** A small logic change could accidentally:
- Add/remove a newline breaking JSON parsing
- Change field name breaking API contract
- Modify structure breaking downstream tools

Golden file tests catch these regressions immediately.

### Manual Testing Scenarios

1. **Stream Redirection**
   ```bash
   fog deploy --template stack.yaml --output json > result.json
   fog deploy --template stack.yaml --output json 2> /dev/null
   fog deploy --template stack.yaml --output json > result.json 2> progress.log
   ```

2. **Format Validation**
   ```bash
   fog deploy --template stack.yaml --output json | jq .
   fog deploy --template stack.yaml --output yaml | yq .
   fog deploy --template stack.yaml --output csv | column -t -s,
   ```

3. **Quiet Mode**
   ```bash
   fog deploy --template stack.yaml --output json --quiet > result.json
   # Verify no stderr output, only stdout
   ```

4. **Error Scenarios**
   ```bash
   fog deploy --template invalid.yaml --output json
   # Verify pre-deployment errors go to stderr only

   fog deploy --template failing-stack.yaml --output json > error.json
   # Verify deployment errors output to stdout in JSON format
   ```

## Implementation Plan

### Phase 1: Infrastructure Setup
**Files:** `cmd/deploy.go`, `cmd/flaggroups.go`

1. Add `--quiet` flag to DeployFlags
2. Add fields to DeployInfo for data capture
3. Create helper function for stderr output creation

### Phase 2: Stream Separation
**Files:** `cmd/deploy.go`, `cmd/deploy_helpers.go`

1. Update `showEvents()` to use stderr writer
2. Update `showDeploymentInfo()` to use stderr writer
3. Update `printBasicStackInfo()` to use stderr writer
4. Update all `printMessage()` calls to use stderr
5. Implement quiet mode checks in all progress functions

### Phase 3: Data Capture
**Files:** `cmd/deploy.go`, `cmd/deploy_helpers.go`

1. Modify `createAndShowChangeset()` to capture changeset
2. Add timestamps to deployment flow
3. Capture final stack state after deployment
4. Capture error details for failed deployments

### Phase 4: Final Output Builders
**Files:** `cmd/deploy_output.go` (new file)

1. Implement `outputDryRunResult()`
2. Implement `outputSuccessResult()`
3. Implement `outputNoChangesResult()`
4. Implement `outputFailureResult()`
5. Add helper functions for data conversion

### Phase 5: Integration
**Files:** `cmd/deploy.go`

1. Restructure `deployTemplate()` flow
2. Add output generation calls
3. Implement error handling for output generation
4. Add stderr flush before stdout output
5. Add stdout header separation

### Phase 6: Testing
**Files:** `cmd/deploy_test.go`, `cmd/deploy_output_test.go`, `cmd/deploy_integration_test.go`

1. Unit tests for output builders
2. Unit tests for stream separation
3. Integration tests for deployment scenarios
4. Manual testing with different formats

### Phase 7: Cleanup
**Files:** `cmd/deploy.go`

1. Remove `viper.Set("output", "table")` override (line 77)
2. Verify all output paths use correct streams
3. Code cleanup and refactoring
4. Final linting and formatting

## Dependencies

### External Dependencies

- **go-output v2.6.0**: Provides StderrWriter and StdoutWriter support
  - Already available and documented
- **github.com/mattn/go-isatty**: TTY detection for conditional formatting
  - Used to prevent ANSI codes in redirected output
  - Lightweight, widely used dependency

### Internal Dependencies

- **Existing fog modules**: No changes to core CloudFormation operations
- **Configuration system**: Uses existing Viper configuration for format selection
- **Flag system**: Extends existing flag groups pattern

## Backwards Compatibility

### BREAKING CHANGES

**This feature introduces breaking changes for existing automation:**

**Current behavior (fog v1.x):**
- All output (progress + results) goes to stdout
- Scripts can parse combined output stream: `fog deploy ... | grep "Status"`

**New behavior (fog v2.x):**
- Progress output goes to stderr (real-time streaming)
- Final results go to stdout (formatted data)
- Scripts parsing stdout will now get different content

**Impact:**
- Scripts that parse stdout will break because progress messages move to stderr
- Scripts that redirect output to files need updating
- CI/CD pipelines may need modifications

**Why this is necessary:**
This change follows Unix conventions (stderr=progress, stdout=data) and enables proper programmatic use. While breaking, it's a positive change that makes the tool more composable.

### Migration Guide

**For scripts parsing combined output:**
```bash
# Old (v1.x):
fog deploy ... | grep "Status"

# New (v2.x) - Option 1: Combine streams like v1.x
fog deploy ... 2>&1 | grep "Status"

# New (v2.x) - Option 2: Parse structured output
fog deploy ... --output json | jq '.status'
```

**For scripts capturing output to files:**
```bash
# Old (v1.x):
fog deploy ... > deployment.log

# New (v2.x) - Capture both streams:
fog deploy ... > result.json 2> progress.log

# Or use quiet mode for CI/CD:
fog deploy ... --quiet --output json > result.json
```

### Default Behavior

- Default output format remains `table` (Decision D8)
- Streaming progress output goes to stderr in table format
- Final results always go to stdout for all deployment outcomes
- No special cases for table format - all formats behave consistently

### Version Guidance

- **v1.x**: Current behavior (all output to stdout)
- **v2.x**: New behavior (stderr/stdout separation)
- Recommend major version bump to signal breaking change
- Consider deprecation period with warning messages

## Security Considerations

### No New Security Risks

- Output generation uses existing data already available to the command
- No new AWS API calls or permissions required
- No new file system access beyond existing `--file` flag
- Stream separation follows standard Unix security model

### Error Message Sanitization

Error messages from CloudFormation may contain sensitive information:
- Stack parameter values
- Resource properties
- IAM policy details

**Mitigation**: This is existing behavior. The feature does not introduce new exposure, as the same information is already visible in streaming output. Users should follow existing best practices for protecting sensitive CloudFormation templates and parameters.

## Performance Considerations

### Minimal Performance Impact

1. **Data Capture**: Negligible overhead - only storing references to existing objects
2. **Output Generation**: Occurs after deployment completes - no impact on deployment time
3. **Stream Buffering**: Stderr flush before stdout adds <1ms latency
4. **go-output Rendering**: Efficient serialization with streaming support for large datasets

### Memory Usage

- Changeset data already loaded for streaming output
- Stack state fetched once at end (already done for current table output)
- No significant increase in memory footprint

## Future Enhancements

Potential improvements not in current scope:

1. **Progress Indicators**: Add spinner or progress bar to stderr output
2. **Colored Output Control**: Add `--no-color` flag for plain output
3. **Custom Templates**: Support custom output templates for formatting
4. **Webhooks**: Send formatted output to webhooks for automation
5. **Real-time Streaming**: Stream events in JSON format to stdout during deployment

These enhancements can be added later without breaking compatibility with this design.

## References

- [Requirements Document](./requirements.md)
- [Decision Log](./decision_log.md)
- [go-output v2 API Documentation](../../docs/research/go-output-v2/API.md)
- [Current Deploy Command](../../cmd/deploy.go)
- [Describe Changeset Command](../../cmd/describe_changeset.go)
