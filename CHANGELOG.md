Unreleased
===========

### Fixed
- Removed hardcoded `=== Deployment Summary ===` headers from deploy output functions that were breaking JSON/YAML parsing
- Added nil checks for `FinalStackState` before accessing `StackStatus` and `Outputs` to prevent potential nil pointer dereference in success output
- Improved duration calculation with zero-time validation to avoid incorrect time calculations
- Changed failure output timestamp handling to use "N/A" instead of `time.Now()` when `DeploymentEnd` is not available for more accurate output
- Replaced deprecated `github.com/mitchellh/go-homedir` dependency with standard library `os.UserHomeDir()` (Audit Issue 11.1)

## BREAKING CHANGES

**Stream Separation for Deploy Command**

The `fog deploy` command now follows Unix conventions by separating progress output from structured data output:

- **Progress output** (stack information, changeset details, deployment status, interactive prompts) → **stderr**
- **Structured results** (deployment summary in JSON/YAML/CSV/etc.) → **stdout**

**Impact on existing scripts:**
- Scripts using `fog deploy ... | grep` will see different content (only final results, not progress)
- Scripts using `fog deploy ... > file` will only capture results, not progress messages
- CI/CD pipelines parsing combined output need updates

**Migration examples:**
```bash
# Old (v1.x) - combined output to stdout
fog deploy --stack mystack | grep "Status"

# New (v2.x) - Option 1: Combine streams
fog deploy --stack mystack 2>&1 | grep "Status"

# New (v2.x) - Option 2: Parse structured output (recommended)
fog deploy --stack mystack --output json | jq '.status'

# New (v2.x) - Option 3: Suppress progress with --quiet
fog deploy --stack mystack --quiet --output json | jq '.status'
```

For more details, see the [deployment output specification](specs/deploy-output/design.md).

---

### Changed
- Updated deployment output to write all changeset and deployment information to stderr while reserving stdout for structured JSON results
- Moved deployment end timestamp setting from deploy.go to printDeploymentResults() for more accurate timing
- Modified all output operations in showChangeset() and related functions to use stderr with table format
- Updated printLog() to conditionally use stderr for deploy context while maintaining stdout for history command
- Added time rounding to deployment duration calculation for cleaner output
- Improved error handling for zero-value timestamps in deployment output
- Refined deployment messages to be more concise and less conversational
- Updated test assertions to match new message formats

### Added
- Stream separation test suite in `cmd/stream_separation_test.go` with comprehensive tests for stderr/stdout separation covering printMessage, createStderrOutput, output functions, stream separation, format helpers, quiet mode, and stderr sync behavior
- Stream verification report documenting audit of all output paths in deploy command (`specs/deploy-output/stream_verification.md`) confirming correct stderr/stdout usage, edge cases, and providing manual verification commands

### Added
- Integration tests for deployment output scenarios in `cmd/deploy_output_integration_test.go` covering successful deployment with JSON output, failed deployment with formatted output, quiet mode, dry-run with multiple formats, and no-changes scenario
- Unit tests for deploy output builder functions in `cmd/deploy_output_test.go` covering success, failure, and no-changes scenarios across all output formats (JSON, YAML, CSV, Markdown, table)
- Golden file test infrastructure for deploy output validation with generator test in `cmd/generate_golden_files_test.go` and golden files in `cmd/testdata/deploy-output/`
- Golden file tests for deployment output covering success (JSON, YAML, CSV, Markdown), failure (JSON, YAML), and no-changes (JSON) scenarios
- Helper functions `compareWithGolden()`, `createGoldenTestDeployment()`, `createGoldenFailedDeploymentTest()`, and `createGoldenNoChangesDeploymentTest()` for golden file testing
- Golden file README documenting structure, update process, and data format conventions
- Support for `UPDATE_GOLDEN=1` environment variable to regenerate golden files when output format changes

### Changed
- Deploy command no longer enforces table output format, allowing users to specify output format via `--output` flag while maintaining table as default
- `outputFailureResult()` now uses `DeploymentEnd` timestamp when available instead of always using `time.Now()` for consistent golden file testing

### Added
- Unit tests for `createStderrOutput()` helper function in `cmd/deploy_helpers_test.go` verifying basic functionality and table format usage

### Added
- `outputSuccessResult()` function in `cmd/deploy_output.go` to output deployment summary for successful deployments with deployment metadata, planned changes, and stack outputs tables
- `outputNoChangesResult()` function in `cmd/deploy_output.go` to output no-changes message with stack information when CloudFormation determines there are no changes to apply
- `outputFailureResult()` function in `cmd/deploy_output.go` to output deployment failure details with error messages, stack status, and failed resources information
- `extractFailedResources()` helper function in `cmd/deploy_output.go` to query stack events and extract failed resource details (LogicalID, ResourceStatus, StatusReason, ResourceType)
- `FailedResource` struct to represent resources that failed during deployment

### Changed
- `printDeploymentResults()` in `cmd/deploy_helpers.go` now calls `outputSuccessResult()` for successful deployments to output formatted summary to stdout after printing success message to stderr
- `printDeploymentResults()` now calls `outputFailureResult()` for failed deployments to output formatted failure details to stdout after showing failed events to stderr
- No-changes scenario in `createChangeset()` now calls `outputNoChangesResult()` to output formatted message to stdout before exiting
- Success and failure paths now write progress messages to stderr (if not quiet) and final formatted output to stdout, completing stream separation for all deployment outcomes
- Output generation errors are now treated as warnings written to stderr rather than command failures

### Added
- `outputDryRunResult()` function in `cmd/deploy_output.go` to output changeset results for dry-run and create-changeset modes
- Changeset data capture in `createAndShowChangeset()` function, storing changeset in both `CapturedChangeset` and `Changeset` fields for backwards compatibility
- Final stack state capture in `printDeploymentResults()` function, storing in `FinalStackState` field
- Deployment error capture in failure path, storing in `DeploymentError` field
- Deployment end timestamp capture after deployment completes (both success and failure)

### Changed
- `createAndShowChangeset()` function now accepts `quiet` parameter to suppress stderr output when quiet mode is enabled
- Changeset overview is now shown to stderr only when not in quiet mode
- Dry-run mode now calls `outputDryRunResult()` for formatted output after changeset creation
- Create-changeset mode now calls `outputDryRunResult()` for formatted output after changeset creation
- Changeset deletion for dry-run mode now happens in main deployment flow after output generation
- `confirmAndDeployChangeset()` function no longer handles create-changeset mode (moved to main flow)
- Deployment flow now captures final stack state and deployment end timestamp for both success and failure paths

### Fixed
- Test `TestCreateAndShowChangeset` updated to reflect new behavior where changeset deletion is handled in main flow
- Test `TestConfirmAndDeployChangeset` updated to remove create-changeset mode test case (now handled in main flow)

### Changed
- Deploy command progress output (stack information, changeset overview, deployment status messages, event streaming, interactive prompts) now writes to stderr instead of stdout following Unix conventions
- `showDeploymentInfo()` function now accepts `quiet` parameter and returns early when quiet mode is enabled, suppressing all output to stderr
- `showEvents()` function now accepts `quiet` parameter for conditional event streaming suppression
- `printBasicStackInfo()` now uses `createStderrOutput()` helper for stderr rendering with TTY detection
- `printMessage()` helper now uses `createStderrOutput()` for stderr rendering instead of stdout
- Interactive confirmation prompts in `askForConfirmation()` now write to stderr using `fmt.Fprintf(os.Stderr, ...)`
- Error messages and progress indicators throughout deployment flow now consistently use stderr via `fmt.Fprintln(os.Stderr, ...)` and `fmt.Fprintf(os.Stderr, ...)`
- Quiet mode (`--quiet` flag) now automatically enables non-interactive mode to auto-approve all prompts
- `showFailedEvents()` now uses `createStderrOutput()` for rendering failed events table to stderr

### Added
- Deployment start timestamp capture in `deployTemplate()` function using `deployment.DeploymentStart = time.Now()` before any AWS operations
- Quiet mode checks in deployment progress functions (`deployChangeset()`, `createChangeset()`) to suppress informational messages when `--quiet` flag is enabled

### Added
- Comprehensive user documentation in `docs/user-guide/` directory including:
  - Complete user guide with installation, quick start, feature overview, and best practices
  - Configuration reference documenting all configuration options with examples for development, production, and CI/CD scenarios
  - Deployment files specification with field reference, examples, and best practices
  - Advanced usage guide covering multi-stack deployments, cross-stack references, multi-region deployments, CI/CD integration (GitHub Actions, GitLab CI, Jenkins), advanced drift detection, template preprocessing, complex tagging strategies, and environment management
  - Troubleshooting guide with solutions for deployment issues, configuration problems, AWS credentials and permissions, template and parameter issues, drift detection issues, output and display issues, and debug mode usage
- Architecture diagrams in `docs/`:
  - `architecture-overview.drawio.svg` visualizing the layered system architecture
  - `configuration-flow.drawio.svg` showing configuration precedence flow
- Documentation section in main README.md with quick links to all user guides
- "Getting Help" section in README.md with documentation links, built-in help commands, and community support information
- `--quiet` flag to DeployFlags struct for suppressing progress output (stderr) while showing only final result
- New fields to DeployInfo struct for deployment tracking: CapturedChangeset, FinalStackState, DeploymentError, DeploymentStart, and DeploymentEnd
- `createStderrOutput()` helper function with TTY detection that conditionally enables colors and emojis based on whether stderr is a TTY, preventing ANSI codes in redirected output
- `go-isatty` dependency (v0.0.20) for terminal detection functionality
- Created specification for deploy command multi-format output support feature in `specs/deploy-output/`
  - Requirements document with 13 user stories and 70+ acceptance criteria covering stream separation, output formats, quiet mode, error handling, backwards compatibility, and testing
  - Design document with output flow diagram, dual-output architecture (stderr for progress, stdout for data), component specifications, data models, error handling strategy, and 7-phase implementation plan
  - Decision log documenting 8 key design decisions including stream separation, format consistency, backwards compatibility acknowledgment, quiet mode behavior, and output generation error handling
  - Task list with 40 implementation tasks organized into 7 phases: infrastructure setup, stream separation, data capture, final output builders, integration, testing, and cleanup
- Enhanced go-output v2 API documentation with details on StderrWriter and StdoutWriter support from v2.6.0
- Unit tests for empty changesets across all output formats (table, CSV, JSON, YAML, Markdown, HTML) verifying proper handling of zero-change scenarios with format-appropriate empty indicators
- Unit tests for all output formats in describe changeset command covering table, CSV, JSON, YAML, Markdown, and HTML rendering
- Unit tests for empty changeset handling across all formats
- Unit tests for ANSI code behavior verification in structured formats
- Unit tests for JSON structure validation matching go-output v2 array format
- Unit tests for dangerous changes detection with Remove actions, Conditional replacements, and True replacements
- Helper functions `contains()` and `captureStdout()` in describe_changeset_test.go for test validation

### Changed
- Describe changeset command now respects global `--output` flag instead of enforcing table format
- Refactored describe changeset command to use `buildAndRenderChangeset()` orchestration function for single document rendering
- Replacement type string literals replaced with constants (`replacementConditional`, `replacementTrue`) for better maintainability

### Removed
- Hardcoded `viper.Set("output", "table")` enforcement in describe changeset command
- Unused `viper` import from describe_changeset.go

### Added
- New `addStackInfoSection()` function in describe changeset command that accepts builder parameter and includes console URL as a field in stack information table
- New `buildChangesetData()` helper function that separates data preparation from rendering logic, returning changeRows, summaryContent, and dangerRows
- New `addChangesetSections()` function that adds changeset tables to builder using empty table for dangerous changes when none exist

### Changed
- Stack information table now includes ConsoleURL field when not a dry run, making it accessible in all output formats
- Dangerous changes section now uses empty table with headers instead of text message when no dangerous changes exist, allowing proper format-specific rendering

### Added
- Created spec for changeset output format support feature in `specs/changeset-output-format/`
  - Requirements document with 33 acceptance criteria across 7 sections covering output format configuration, content completeness, format-specific rendering, scope limitation, backward compatibility, data structure specification, and error handling
  - Design document with architecture diagrams, component specifications, data structure definitions for JSON/YAML/CSV formats, testing strategy, and implementation checklist
  - Decision log documenting 10 key design decisions including removal of DOT format, console URL as table field, acceptance of go-output v2's tables array structure, and preservation of shared functions
  - Task list with 20 implementation tasks organized into 4 phases: refactor output functions, update command function, testing, and cleanup

### Added
- Created `cmd/output_helpers.go` with formatting helper functions (formatInfo, formatSuccess, formatError, formatPositive, formatBold) that replicate v1 output styling behavior with emojis and colors
- Added `printMessage()` helper function for rendering formatted messages through go-output document builder

### Changed
- Refactored deploy command to use unified go-output document building for success messages and outputs table instead of mixing `fmt.Print()` with go-output rendering
- Updated `printLog()` function to accept an optional message parameter for better integration with formatted output
- Migrated all precheck and deployment status messages to use new formatting helpers instead of plain text prefixes

### Fixed
- Missing emoji and color formatting in deployment status messages (info ℹ️, success ✅, error 🚨)
- Missing visual spacing before prechecks start in deployment workflow (added leading newline)
- Missing spacing between changeset summary table and AWS Console link (added newline before console URL)
- Table spacing issues caused by mixing `fmt.Print()` and go-output rendering (now using unified document building)

### Changed
- Upgraded go-output dependency from v2.4.0 to v2.5.0 for thread-safe format functions and parallel test support
- Migrated all format references from variables to functions (e.g., `output.JSON` → `output.JSON()`) per v2.5.0 breaking changes
- Re-enabled `t.Parallel()` for tests that don't use viper global state (deploy_test.go, report_test.go)
- Updated all test files to use format function calls in both `WithFormat()` calls and struct literals

### Changed
- Simplified inline styling in deploy command by removing unnecessary wrapper functions (stringFailure, stringSuccess, stringInfo, stringWarning, stringPositive, stringBold) and calling output.Style*() functions directly
- Simplified inline styling in drift command by removing format-checking wrapper functions (styleForFormat, styleWarning, stylePositive) and calling output.Style*() functions directly
- Updated config to use EnhancedColorTransformer instead of ColorTransformer for format-aware color handling

### Known Issues
- Inline styling functions (StyleWarning, StylePositive) embed ANSI codes in data which appear in JSON/CSV/DOT output - this will be addressed in a future go-output v2 update

### Fixed
- Race conditions in parallel test execution causing `fatal error: concurrent map writes` in cmd and config packages (removed `t.Parallel()` from tests that use global state and concurrent rendering)
- Linting issue resolved by extracting "html" string constant in report command
- File output now works correctly for all commands (drift, report, etc.) by using `WithFormats()` instead of multiple `WithFormat()` calls
- Report command `--file` flag now works correctly (removed local flag that was shadowing global persistent flag)
- HTML format output now renders correctly (added missing `case "html"` in format switch)
- Empty directory paths in file output now default to current directory
- File format now correctly defaults to console format when `--file-format` is not specified
- ANSI color codes no longer appear in JSON/CSV/YAML output from drift command (styling now only applied to table/markdown/html formats)
- File writer creation errors are now logged with warning messages instead of being silently swallowed (config.go)
- Report frontmatter now properly attached to v2 output via `WithFrontMatter()` option
- Mermaid timeline diagrams now render as proper Gantt charts using v2 `GanttChart()` API instead of plain tables
- Improved error message for S3 template upload failures in deploy command (changed from "this failed" to "Failed to upload template to S3")

### Changed
- Updated go-output dependency from v2.3.2 to v2.3.3 (includes Mermaid rendering fixes)
- Report help text now clarifies that `--file` uses exact filename (no placeholders), while S3 output via Lambda supports placeholders
- Report command uses v2 `createMermaidGanttChart()` returning GanttTask objects instead of map data
- Test `TestReportMermaidTableGeneration` renamed to `TestReportMermaidGanttChartGeneration` with updated assertions for GanttTask structure
- Test expectations for file output updated from 4 options to 3 options (console writer + file writer + formats)

### Removed
- Obsolete "capture range variable" and "capture loop variable" comments from all test files (Go 1.22+ automatically captures loop variables)
- Unused `checkedResources` variable and associated workaround code from drift.go

### Changed
- Modernized test patterns for Go 1.22+ compatibility (removed explicit loop variable captures `tc := tc` as loop variables are now automatically captured)
- Updated drift detection output formatting to use newlines instead of dynamic separators for better readability
- Added explanatory comments in report.go for manual sorting approach in events and mermaid tables

### Removed
- `GetSeparator()` method from config.Config as separator logic is now handled directly in drift detection
- Unused config tests for GetSeparator method

### Added
- Migration completion documentation in decision log (Decision 8) summarizing v2 migration success, implementation decisions, deviations from design, lessons learned, and migration metrics

### Changed
- Replaced `interface{}` with `any` in deploy.go helper functions per Go 1.18+ best practices
- Removed duplicate import alias in config/config.go (consolidated to single `output` alias)
- Simplified loop in cmd/history.go using `append(...slice...)` pattern instead of manual iteration
- Refactored deploy.go helper functions from one-line to multi-line format for better readability
- Removed `t.Parallel()` from deploy_helpers_test.go tests to avoid test timeout issues with global state

### Added
- Golden file test infrastructure with ANSI code stripping for test validation
- `StripAnsi()` helper function to remove ANSI escape codes from strings
- `AssertStringWithoutAnsi()` method for validating output content without formatting codes
- Manual validation results documented in decision log (Decision 7) confirming functional equivalence with v1
- Windows cross-compilation verification confirming v2 resolves v1 compilation issues

### Changed
- Test validation philosophy updated to focus on data correctness rather than byte-for-byte matching
- Config tests no longer use parallel execution to avoid viper global state race conditions

### Fixed
- Golden file tests now strip ANSI codes before comparison to validate content structure
- Config test race conditions resolved by removing `t.Parallel()` from tests using viper global state
- Test assertions changed from `SetDefault()` to `Set()` for consistent viper configuration

### Changed
- Updated go-output dependency from v1.4.0 to v2.3.0 with new v2 package structure
- Updated all import paths from `github.com/ArjenSchwarz/go-output` to `github.com/ArjenSchwarz/go-output/v2` across 15 Go files
- Mermaid/gantt chart support now uses v2.3.0 native APIs (ChartContent, NewGanttChart) instead of separate mermaid subpackage
- **resources command**: Migrated from v1 OutputArray pattern to v2 Builder pattern with modern Go patterns (slices.SortFunc for sorting)
- **deploy command**: Migrated from v1 OutputSettings to v2 Builder pattern with simplified string formatting helpers
- **drift command**: Migrated from v1 OutputArray to v2 Builder pattern with incremental row building
- **report command**: Migrated to v2 Output API with context-based rendering
- **describe changeset command**: Migrated from v1 OutputArray to v2 Builder pattern with multiple tables support
- **demo tables command**: Migrated from v1 OutputArray/OutputHolder to v2 Builder pattern with explicit style list
- **history command**: Migrated from v1 OutputArray to v2 Builder pattern with settings-based configuration

### Added
- go-output v2 specification and research documentation in specs/go-output-v2 directory
- API documentation covering all v2 public interfaces and agent implementation patterns
- Migration guide detailing v1 to v2 upgrade path with breaking changes and code examples
- Evaluation document comparing v2 against alternatives (pterm, lipgloss, glamour, charmbracelet/log)
- Design document outlining v2 architecture, threading model, and collapsible content system
- Requirements specification defining functional, technical, and user experience requirements
- Task breakdown with sprint planning for go-output v2 implementation
- Decision log tracking architectural and design choices
- Golden file baseline tests for exports command v1 output (table, CSV, JSON formats)
- Test coverage for verbose and non-verbose exports output modes
- **Comprehensive unit tests for resources command**: Tests for v2 Builder pattern, column ordering (basic and verbose), sorting by Type, multiple output formats (table, CSV, JSON, Markdown), array field handling, and empty results
- **Integration tests for deploy command**: Tests for deployment preparation, S3 uploads, and error handling
- **Integration tests for drift command**: Tests for drift detection scenarios and output formatting
- **Integration tests for report command**: Tests for report generation with different output formats
- **Comprehensive unit tests for describe changeset command**: Tests for stack info, changeset changes, danger table, summary table, multiple output formats, sorting, empty changesets, and action/replacement variations
- **Comprehensive unit tests for demo tables command**: Tests for different table styles (Default, Bold, ColoredBright, Light, Rounded), long descriptions with column wrapping, sorted output, multiple output formats, boolean value handling, and column ordering
- **Comprehensive unit tests for history command**: Tests for deployment history, multiple output formats, column ordering, and log formatting

### Fixed
- **helpers.go**: Replaced v1 `settings.NewOutputSettings().StringFailure()` with v2 `output.StyleNegative()` for error messages
- **deploy_helpers_test.go**: Removed obsolete v1 `outputsettings` initialization from test cases
- **config.go**: Removed deprecated `NewOutputSettings()` method and all references to global `outputsettings` variable from test files
- **deploy_integration_test.go**: Removed global `outputsettings` variable assignments no longer needed with v2 API
- **drift_integration_test.go**: Updated `TestTransitGatewayDrift_SeparatePropertiesFlag` to use v2 API patterns instead of deprecated `OutputArray`

1.11.0 / 2025-10-17
===================

### Added
- Transit Gateway drift detection for route tables with support for `Fn::ImportValue` resolution
- Filtering of propagated routes and transient states in Transit Gateway drift detection
- Golden file testing framework for deployment output validation
- Test utilities package (lib/testutil) with assertion helpers, builders, fixtures, and mock clients
- Integration tests for deployment workflows, prechecks, changesets, and rollback scenarios
- Test validation and coverage reporting scripts
- Documentation comments for all exported types and functions

### Changed
- Drift detection resolves `Fn::ImportValue` references for route attachments
- Template parameter constraints now support both string and numeric values
- Modernized codebase to use Go 1.25 built-in functions (`any`, `maps.Copy()`, `slices.Contains()`)
- Refactored drift detection functions to use interface-based dependency injection
- Enhanced deploy helper functions with extracted utilities for better testability
- Updated README.md with development section covering building, testing, and linting

### Fixed
- Template parsing no longer fails when parameter constraints are strings instead of numbers
- Drift detection output properly handles properties with non-JSON values

1.10.2 / 2025-06-04
===================

  * Add unit tests for GetResources and document the scenarios covered.
  * Fix `golangci-lint` errors by checking file close errors and
    replacing `strings.Replace` with `strings.ReplaceAll`.

1.10.1 / 2025-06-04
===================

  * Document contribution guidelines and local validation steps.

1.10.0 / 2025-05-23
===================

  * Major drift detection enhancements:
    - Detect unmanaged AWS resources (e.g., SSO Permission Sets and Assignments)
    - Support for ignoring specific blackhole routes and unmanaged resources via config
    - Improved handling of IPv6 CIDR blocks in NACL resource parsing
    - New drift detection options in `fog.yaml` (`ignore-blackholes`, `detect-unmanaged-resources`, `ignore-unmanaged-resources`)
  * SSO/Identity Center support:
    - List SSO Permission Sets and Assignments, with helper functions for AWS SSO and Organizations APIs
  * Testing:
    - Added comprehensive unit tests for drift detection, file handling, logging, deployment messages, EC2, stacks, and changesets
  * Refactoring and improvements:
    - Refactored commands into command groups
    - Refactored handling of flags through flag groups
    - Refactored and added helper functions for string/map handling
    - Improved error handling and logging
  * Dependency and tooling updates:
    - Updated AWS SDKs and other dependencies in `go.mod` and `go.sum`
    - Bumped Go version to 1.24.0 in both `go.mod` and CI workflow
  * Other:
    - Improved release workflow
    - Minor configuration and code quality improvements
    - Add logo for fog

1.9.0 / 2024-02-27
==================

  * Add support for deployment files
  * Add support for ignoring certain tags in the drift detection.
  * Dependency updates
  * Update Go version and support for separate output files

1.8.0 / 2023-06-17
==================

  * Improved changesets and added drift detection

1.7.0 / 2023-05-01
==================

  * Changeset improvements: show all changes + summary

1.6.0 / 2022-09-12
==================

  * Add support for timezones
  * Support direct file names for template, tags and parameters (thanks to @mludvig)
  * Support passing the extension for source files

1.5.0 / 2022-08-28
==================

  * Show diagram above table
  * Clean up the stackname in the output when an arn is provided

1.4.0 / 2022-08-25
==================

  * Report: Add support for frontmatter and filename placeholders
  * Also includes some restructuring and an example template for a fog reports bucket

1.3.0 / 2022-08-14
==================

  * Support writing of report to S3 buckets.
  * Upgrade to Go 1.19

1.2.2 / 2022-06-06
==================

  * Unlimited retries to handle API rate limiting in larger accounts
  * Be able to loop over more than 100 stacks to handle larger accounts

1.2.1 / 2022-06-02
==================

  * Show the IDs of resources that have been added during an event
  * Show the IDs of both the original and new resource for resources that are replaced during an event
  * Show the cleanup action for failed resources in an event
  * Show the reason for a failure of a resource in an event
  * Show failed resources differently in the chart

1.2.0 / 2022-06-01
==================

  * Report: Add support for milestones and show replacement actions better
  * The graph now shows milestones for stack status changes like UPDATE_COMPLETE_CLEANUP_IN_PROGRESS and UPDATE_COMPLETE
  * Resources that are being replaced now have 2 entries, one where the new resource is created and one where the old one is cleaned up.


1.1.5 / 2022-06-01
==================

  * Bugfix for dealing with accounts without aliases

1.1.4 / 2022-05-31
==================

  * Support report functionality
  * Use separate go-output library for handling outputs

1.0.0 / 2021-10-07
==================

  * Show account alias when deploying
  * Add support for prechecks
  * Show modules in change sets

0.10.0 / 2021-09-14
===================

  * Update go version and modules
  * Add documentation to the README and example code
  * Add support for global tags and placeholder values

0.9.0 / 2021-09-01
==================

  * Add very basic README and deployment flow diagram
  * Fix descriptions of some flags
  * Add support for uploading the template to S3 before deployment
  * Add support for multiple tag and parameter files

0.8.0 / 2021-08-23
==================

  * Add support for showing dependencies between stacks
  * Use dev for the development version
  * Use built-in functionality for printing settings
  * Add changeset specific functionalities

0.7.0 / 2021-08-11
==================

  * Add non-interactive mode to deployments
  * Restructure of text messages into its own file
  * Missing quote in release workflow

0.6.0 / 2021-07-26
==================

  * Support for dry run
  * Use correct type for release workflow

0.5.3 / 2021-07-26
==================

  * Fix ldflags for version
  * Only run packaging when creating a release

0.5.0 / 2021-07-25
==================

  * Add GitHub Action for building binaries
  * Support for actual deployments
  * Add command to list all CloudFormation managed resources
  * Extra config info
  * Make things more reusable in exports cmd
  * Exports shows imported by in verbose mode
  * Add demo command
  * Add export name filter to exports command
  * Add ability for wildcard filter by stackname
  * Initial exports functionality
  * Initial commit
