---
references:
    - specs/deploy-output/requirements.md
    - specs/deploy-output/design.md
    - specs/deploy-output/decision_log.md
---
# Deploy Output Feature - Multi-Format Output Support

- [x] 1. Add go-isatty dependency for TTY detection
  - This enables conditional formatting based on whether stderr is a TTY

## Phase 1: Infrastructure Setup

- [x] 2. Add --quiet flag to DeployFlags struct

- [x] 3. Register --quiet flag in deploy command

- [x] 4. Add new fields to DeployInfo struct for data capture

- [x] 5. Create createStderrOutput() helper with TTY detection

## Phase 2: Stream Separation

- [x] 6. Update showEvents() to use stderr with quiet mode support
  - Modify showEvents() function in cmd/deploy_helpers.go to use stderr output writer
  - Add quiet parameter to function signature
  - Return early if quiet mode is enabled
  - Use createStderrOutput() helper for output creation
  - Ensure all event streaming writes to stderr, not stdout
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [5.2](requirements.md#5.2)
  - References: cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 7. Update showDeploymentInfo() to use stderr with quiet mode support
  - Modify showDeploymentInfo() function to use stderr output writer
  - Add quiet parameter to function signature
  - Return early if quiet mode is enabled
  - Use createStderrOutput() helper
  - Ensure stack information writes to stderr
  - Requirements: [3.3](requirements.md#3.3), [4.6](requirements.md#4.6), [5.3](requirements.md#5.3)
  - References: cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 8. Update printBasicStackInfo() to use stderr
  - Modify printBasicStackInfo() function to write to stderr
  - Use createStderrOutput() helper for output creation
  - Maintain current table format for stack information
  - Ensure consistent formatting with other stderr output
  - Requirements: [3.3](requirements.md#3.3), [4.6](requirements.md#4.6)
  - References: cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 9. Update all progress printMessage() calls to use stderr
  - Search for all printMessage() calls in deployment flow
  - Update each call to write to stderr instead of stdout
  - Add quiet mode checks where appropriate
  - Verify no progress messages leak to stdout
  - Requirements: [3.1](requirements.md#3.1), [4.1](requirements.md#4.1)
  - References: cmd/deploy.go, cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 10. Update interactive prompts to write to stderr
  - Modify confirmation prompts to write questions to stderr
  - Ensure prompts read responses from stdin
  - Add quiet mode logic to auto-approve prompts
  - Test that redirected stdout does not affect prompts
  - Requirements: [3.5](requirements.md#3.5), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5)
  - References: cmd/deploy.go, specs/deploy-output/design.md

- [x] 11. Capture deployment start timestamp
  - Add DeploymentStart field to DeployInfo struct
  - Set timestamp at beginning of deployment flow in deployTemplate()
  - Ensure timestamp is captured before any AWS operations
  - Use time.Now() for consistent timing
  - Requirements: [7.4](requirements.md#7.4)
  - References: lib/stacks.go, cmd/deploy.go, specs/deploy-output/design.md

## Phase 3: Data Capture

- [x] 12. Modify createAndShowChangeset() to capture changeset data
  - Modify createAndShowChangeset() function to capture changeset immediately after creation
  - Store changeset in deployment.CapturedChangeset field
  - Maintain existing deployment.Changeset field for backwards compatibility
  - Add quiet parameter to suppress stderr output
  - Show changeset overview to stderr only when not in quiet mode
  - Requirements: [12.2](requirements.md#12.2), [6.3](requirements.md#6.3)
  - References: cmd/deploy_helpers.go, lib/stacks.go, specs/deploy-output/design.md

- [x] 13. Capture deployment end timestamp and final stack state on success
  - Add DeploymentEnd and FinalStackState fields to DeployInfo struct
  - Set DeploymentEnd timestamp when deployment completes successfully
  - Call getStackState() to fetch final stack information
  - Store final stack in deployment.FinalStackState
  - Ensure this happens before output generation
  - Requirements: [7.4](requirements.md#7.4), [12.6](requirements.md#12.6)
  - References: lib/stacks.go, cmd/deploy.go, specs/deploy-output/design.md

- [x] 14. Capture error details and stack state on deployment failure
  - Add DeploymentError field to DeployInfo struct
  - Set DeploymentError when deployment fails
  - Set DeploymentEnd timestamp on failure
  - Call getStackState() to capture final stack state after failure
  - Store failed stack state in deployment.FinalStackState
  - Handle cases where stack state fetch fails gracefully
  - Requirements: [9.2](requirements.md#9.2), [9.4](requirements.md#9.4), [12.6](requirements.md#12.6)
  - References: lib/stacks.go, cmd/deploy.go, specs/deploy-output/design.md

- [x] 15. Create outputDryRunResult() function
  - Create new outputDryRunResult() function in cmd/deploy_output.go
  - Call os.Stderr.Sync() to flush stderr before stdout output
  - Reuse buildAndRenderChangeset() from cmd/describe_changeset.go
  - Pass deployment.CapturedChangeset data to buildAndRenderChangeset()
  - Function should handle both --dry-run and --create-changeset modes
  - Return error if output generation fails
  - Requirements: [6.1](requirements.md#6.1), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [10.5](requirements.md#10.5)
  - References: cmd/deploy_output.go, cmd/describe_changeset.go, specs/deploy-output/design.md

## Phase 4: Final Output Builders

- [x] 16. Create outputSuccessResult() function
  - Create outputSuccessResult() function in cmd/deploy_output.go
  - Call os.Stderr.Sync() and print header before output
  - Build DeploymentSummary struct with status, ARN, changeset ID, planned changes, outputs, timestamps
  - Create go-output document with three tables: summary, planned changes, stack outputs
  - Use settings.GetOutputOptions() to respect user format preference
  - Render document to stdout
  - Return error if output generation fails
  - Requirements: [7.1](requirements.md#7.1), [7.2](requirements.md#7.2), [7.3](requirements.md#7.3), [7.4](requirements.md#7.4), [7.5](requirements.md#7.5), [7.6](requirements.md#7.6), [10.5](requirements.md#10.5), [10.6](requirements.md#10.6)
  - References: cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 17. Create outputNoChangesResult() function
  - Create outputNoChangesResult() function in cmd/deploy_output.go
  - Call os.Stderr.Sync() and print header before output
  - Build NoChangesResult struct with message, stack name, status, ARN, last updated time
  - Create go-output document with text message and stack information table
  - Use settings.GetOutputOptions() to respect user format preference
  - Render document to stdout
  - Return error if output generation fails
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4), [10.5](requirements.md#10.5), [10.6](requirements.md#10.6)
  - References: cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 18. Create outputFailureResult() function
  - Create outputFailureResult() function in cmd/deploy_output.go
  - Call os.Stderr.Sync() and print header before output
  - Build DeploymentFailure struct with error message, stack status, status reason, failed resources, ARN, timestamp
  - Create go-output document with error text, stack status table, and failed resources table
  - Use settings.GetOutputOptions() to respect user format preference
  - Render document to stdout
  - Return error if output generation fails
  - Requirements: [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.3](requirements.md#9.3), [9.4](requirements.md#9.4), [9.5](requirements.md#9.5), [9.6](requirements.md#9.6), [10.5](requirements.md#10.5), [10.6](requirements.md#10.6)
  - References: cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 19. Create helper function extractFailedResources()
  - Create extractFailedResources() helper function in cmd/deploy_output.go
  - Function takes DeployInfo as input and returns []FailedResource
  - Query stack events to find resources with failed statuses
  - Extract LogicalID, ResourceStatus, StatusReason, ResourceType for each failed resource
  - Handle cases where stack events are unavailable
  - Return empty slice if no failed resources found
  - Requirements: [9.3](requirements.md#9.3)
  - References: cmd/deploy_output.go, lib/stacks.go, specs/deploy-output/design.md

- [x] 20. Integrate outputDryRunResult() into deployment flow
  - Modify deployTemplate() in cmd/deploy.go to call outputDryRunResult()
  - Call after changeset creation when deployment.IsDryRun is true
  - Pass awsConfig to outputDryRunResult()
  - Handle output generation errors - return error to fail command
  - Ensure changeset is deleted after output for dry-run mode
  - Exit successfully after output generation
  - Requirements: [6.1](requirements.md#6.1), [11.5](requirements.md#11.5), [12.3](requirements.md#12.3)
  - References: cmd/deploy.go, cmd/deploy_output.go, specs/deploy-output/design.md

## Phase 5: Integration

- [x] 21. Integrate create-changeset mode output
  - Modify deployTemplate() to handle --create-changeset flag
  - Call outputDryRunResult() when flags.CreateChangeset is true
  - Pass awsConfig to outputDryRunResult()
  - Do NOT delete changeset after output for create-changeset mode
  - Handle output generation errors - return error to fail command
  - Exit successfully after output generation
  - Requirements: [6.2](requirements.md#6.2), [6.6](requirements.md#6.6), [11.5](requirements.md#11.5), [12.4](requirements.md#12.4)
  - References: cmd/deploy.go, cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 22. Integrate outputSuccessResult() into deployment flow
  - Modify deployTemplate() to call outputSuccessResult() after successful deployment
  - Capture DeploymentEnd timestamp before calling output function
  - Capture final stack state using getStackState()
  - Handle output generation errors - return error to fail command
  - Return nil only if both deployment and output generation succeed
  - Ensure function is called for all successful deployment paths
  - Requirements: [7.1](requirements.md#7.1), [10.1](requirements.md#10.1), [11.5](requirements.md#11.5)
  - References: cmd/deploy.go, cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 23. Integrate outputNoChangesResult() for no-changes scenario
  - Identify no-changes error in deployment flow using isNoChangesError()
  - Call outputNoChangesResult() when no changes are detected
  - Handle output generation errors - write warning to stderr but treat as success
  - Return nil exit code for no-changes scenario
  - Ensure this is treated as successful outcome (exit code 0)
  - Requirements: [8.1](requirements.md#8.1), [8.5](requirements.md#8.5), [10.1](requirements.md#10.1)
  - References: cmd/deploy.go, cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 24. Integrate outputFailureResult() for deployment failures
  - Capture deployment error in deployment.DeploymentError field
  - Set DeploymentEnd timestamp when deployment fails
  - Attempt to capture final stack state (may fail gracefully)
  - Call outputFailureResult() with captured data
  - Handle output generation errors - return combined error message
  - Return original deployment error to maintain non-zero exit code
  - Ensure this happens for all deployment failure paths
  - Requirements: [9.1](requirements.md#9.1), [10.1](requirements.md#10.1), [11.5](requirements.md#11.5)
  - References: cmd/deploy.go, cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 25. Implement quiet mode auto-approval logic
  - Add initializeQuietMode() function or inline logic in deployTemplate()
  - When flags.Quiet is true, set flags.NonInteractive to true
  - Document that quiet mode implies non-interactive mode
  - Add check at start of deployment flow
  - Ensure all interactive prompts respect NonInteractive flag
  - Requirements: [5.5](requirements.md#5.5)
  - References: cmd/deploy.go, specs/deploy-output/design.md

- [x] 26. Pass quiet flag through all progress output functions
  - Add quiet bool parameter to all functions that output progress to stderr
  - Update function signatures for showEvents(), showDeploymentInfo(), printBasicStackInfo()
  - Pass quiet flag from deployTemplate() through all call sites
  - Ensure all progress functions check quiet flag and return early if true
  - Verify no progress output escapes to stderr in quiet mode
  - Requirements: [5.2](requirements.md#5.2), [5.3](requirements.md#5.3)
  - References: cmd/deploy.go, cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 27. Write unit tests for createStderrOutput() TTY detection
  - Create test file cmd/deploy_helpers_test.go if not exists
  - Test createStderrOutput() returns output with table format
  - Test TTY detection adds color and emoji transformers when stderr is TTY
  - Test TTY detection skips transformers when stderr is not TTY
  - Mock os.Stderr to simulate different TTY states
  - Verify output writer is configured for stderr
  - Requirements: [1.2](requirements.md#1.2), [4.7](requirements.md#4.7)
  - References: cmd/deploy_helpers_test.go, cmd/deploy_helpers.go, specs/deploy-output/design.md

## Phase 6: Testing

- [x] 28. Write unit tests for output builder functions
  - Create test file cmd/deploy_output_test.go
  - Test outputSuccessResult() with mock deployment data
  - Test outputNoChangesResult() with mock stack data
  - Test outputFailureResult() with mock error data
  - Test outputDryRunResult() integration with buildAndRenderChangeset
  - Test each output format (JSON, CSV, YAML, Markdown, table)
  - Verify data structure correctness
  - Test error handling for missing fields
  - Requirements: [7.6](requirements.md#7.6), [8.4](requirements.md#8.4), [9.6](requirements.md#9.6), [11.5](requirements.md#11.5)
  - References: cmd/deploy_output_test.go, cmd/deploy_output.go, specs/deploy-output/design.md

- [x] 29. Create golden files for output formats
  - Create testdata directory for golden files
  - Generate golden files for successful deployment: success-output.json, success-output.yaml, success-output.csv, success-output.md
  - Generate golden files for failed deployment: failure-output.json, failure-output.yaml
  - Generate golden files for no-changes: no-changes-output.json
  - Generate golden files for dry-run: dry-run-output.json
  - Use realistic sample data that matches actual deployment structures
  - Document golden file update process in test comments
  - Requirements: [7.6](requirements.md#7.6), [8.4](requirements.md#8.4), [9.6](requirements.md#9.6)
  - References: cmd/testdata/, specs/deploy-output/design.md

- [x] 30. Write golden file tests for all output formats
  - Add golden file test functions to cmd/deploy_output_test.go
  - Test each output format against corresponding golden file
  - Use testutil.AssertGolden() or implement golden file comparison
  - Capture stdout output using os.Pipe() or bytes.Buffer
  - Test successful deployment output for all formats
  - Test failure output for JSON and YAML
  - Test no-changes output for JSON
  - Test dry-run output for JSON
  - Add flag to regenerate golden files when needed
  - Requirements: [7.6](requirements.md#7.6), [8.4](requirements.md#8.4), [9.6](requirements.md#9.6), [12.1](requirements.md#12.1)
  - References: cmd/deploy_output_test.go, cmd/testdata/, specs/deploy-output/design.md

- [x] 31. Write integration test for successful deployment with JSON output
  - Create or update cmd/deploy_integration_test.go with integration build tag
  - Create test function TestDeploy_SuccessfulWithJSONOutput
  - Use mock CloudFormation client from lib/testutil
  - Set up test stack and changeset
  - Execute deployment with --output json flag
  - Capture stdout and stderr separately
  - Verify stdout contains valid JSON with expected fields
  - Verify stderr contains progress output (unless quiet)
  - Parse JSON and validate deployment summary structure
  - Requirements: [1.3](requirements.md#1.3), [3.2](requirements.md#3.2), [7.1](requirements.md#7.1), [7.6](requirements.md#7.6)
  - References: cmd/deploy_integration_test.go, lib/testutil/, specs/deploy-output/design.md

- [x] 32. Write integration test for failed deployment with formatted output
  - Create test function TestDeploy_FailureWithFormattedOutput in cmd/deploy_integration_test.go
  - Use mock CloudFormation client that simulates deployment failure
  - Set up test stack that will fail during deployment
  - Execute deployment with --output json flag
  - Capture stdout and stderr separately
  - Verify stdout contains error details in JSON format
  - Verify stderr contains progress output up to failure point
  - Validate error structure includes stack status, failed resources, status reason
  - Verify command returns non-zero exit code
  - Requirements: [3.2](requirements.md#3.2), [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.6](requirements.md#9.6)
  - References: cmd/deploy_integration_test.go, lib/testutil/, specs/deploy-output/design.md

- [x] 33. Write integration test for quiet mode
  - Create test function TestDeploy_QuietMode in cmd/deploy_integration_test.go
  - Use mock CloudFormation client
  - Execute deployment with --quiet flag
  - Capture stdout and stderr separately
  - Verify stderr is empty (no progress output)
  - Verify stdout contains formatted output
  - Verify non-interactive mode is auto-enabled
  - Test that no prompts are displayed
  - Validate quiet mode works with different output formats
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6)
  - References: cmd/deploy_integration_test.go, specs/deploy-output/design.md

- [x] 34. Write integration test for dry-run with multiple formats
  - Create test function TestDeploy_DryRunMultipleFormats in cmd/deploy_integration_test.go
  - Use mock CloudFormation client
  - Test dry-run with JSON, YAML, CSV, and Markdown formats
  - Verify changeset output matches describe changeset command output
  - Verify no actual deployment occurs
  - Verify changeset is deleted after output
  - Capture and validate output for each format
  - Ensure output structure is consistent across formats
  - Requirements: [6.1](requirements.md#6.1), [6.3](requirements.md#6.3), [6.5](requirements.md#6.5), [12.3](requirements.md#12.3)
  - References: cmd/deploy_integration_test.go, specs/deploy-output/design.md

- [x] 35. Write integration test for no-changes scenario
  - Create test function TestDeploy_NoChanges in cmd/deploy_integration_test.go
  - Use mock CloudFormation client that returns no changes
  - Execute deployment
  - Verify no-changes output is produced
  - Verify output includes current stack information
  - Verify exit code is 0 (success)
  - Test with different output formats
  - Validate no-changes message appears in output
  - Requirements: [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.5](requirements.md#8.5)
  - References: cmd/deploy_integration_test.go, specs/deploy-output/design.md

- [x] 36. Remove viper.Set("output", "table") override
  - Locate viper.Set("output", "table") call in cmd/deploy.go (around line 77)
  - Remove this override to allow user format preferences
  - Verify default format is still "table" through configuration
  - Test that --output flag works without the override
  - Ensure backwards compatibility is maintained
  - Update any comments that reference this override
  - Requirements: [2.6](requirements.md#2.6), [1.1](requirements.md#1.1)
  - References: cmd/deploy.go, specs/deploy-output/design.md

## Phase 7: Cleanup

- [x] 37. Verify all output paths use correct streams
  - Audit all output calls in cmd/deploy.go and cmd/deploy_helpers.go
  - Verify progress output goes to stderr using createStderrOutput()
  - Verify final formatted output goes to stdout
  - Check that interactive prompts write to stderr
  - Verify no stdout output occurs during deployment progress
  - Test stream separation with redirected output
  - Document any edge cases or special handling
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.4](requirements.md#3.4), [10.5](requirements.md#10.5)
  - References: cmd/deploy.go, cmd/deploy_helpers.go, specs/deploy-output/design.md

- [x] 38. Run go fmt on all modified files
  - Run go fmt ./cmd/... on command files
  - Run go fmt ./lib/... on library files
  - Verify all code follows Go formatting standards
  - Check that imports are properly organized
  - Ensure consistent spacing and indentation
  - References: cmd/, lib/

- [x] 39. Run go test ./... to verify all tests pass
  - Run go test ./... without INTEGRATION flag for unit tests
  - Run INTEGRATION=1 go test ./... for integration tests
  - Verify all unit tests pass
  - Verify all integration tests pass
  - Fix any test failures
  - Check test coverage to ensure adequate coverage
  - Run tests with -v flag to see detailed output if needed
  - References: cmd/, lib/

- [x] 40. Run golangci-lint to ensure code quality
  - Run golangci-lint run to check for code quality issues
  - Fix any linting errors reported
  - Address any warnings that are relevant
  - Verify no new linting issues were introduced
  - Ensure code follows project linting standards
  - Document any intentional linting suppressions with comments
  - References: cmd/, lib/
