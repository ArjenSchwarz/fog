Unreleased
===========

### Added
- Enhanced error handling in GetTransitGatewayRouteTableRoutes with AWS API error type assertions
- Context timeout handling (30 seconds) for Transit Gateway route retrieval API calls
- State filters for SearchTransitGatewayRoutes API (active and blackhole states only)
- Specific error handling for InvalidRouteTableID.NotFound and UnauthorizedOperation AWS errors
- Additional test cases for AWS error scenarios (InvalidRouteTableID.NotFound, UnauthorizedOperation, context timeout)
- Context validation test to ensure proper context passing

### Changed
- Updated GetTransitGatewayRouteTableRoutes to use context.WithTimeout for API call protection
- Enhanced GetTransitGatewayRouteTableRoutes with errors.As for smithy.APIError type assertions
- Improved error messages with specific context (route table ID, timeout duration, IAM permissions)

### Added
- Unit tests for Transit Gateway route table helper functions in lib/tgw_routetables_test.go
- TestGetTGWRouteDestination with tests for CIDR block extraction, prefix list extraction, nil handling, and precedence
- TestGetTGWRouteTarget with tests for attachment ID extraction, blackhole state handling, empty array handling, nil pointer handling, and ECMP behavior
- TestGetTransitGatewayRouteTableRoutes with tests for successful retrieval, empty tables, API errors, and parameter validation
- Godoc comments for all Transit Gateway test functions following Go documentation standards
- Mock EC2SearchTransitGatewayRoutesAPI implementation for isolated unit testing

### Added
- AWS SDK interface for Transit Gateway route table operations (EC2SearchTransitGatewayRoutesAPI)
- Core Transit Gateway route table helper functions in lib/tgw_routetables.go
- GetTransitGatewayRouteTableRoutes function for retrieving routes from AWS
- GetTGWRouteDestination function for extracting route destination identifiers
- GetTGWRouteTarget function for extracting route target identifiers with ECMP support

### Added
- Transit Gateway drift detection specification including requirements, design, tasks, and decision log
- Requirements document defining 11 requirement categories covering Transit Gateway route table detection, template parsing, AWS API integration, route comparison, output formatting, and error handling
- Design document specifying technical architecture, component interfaces, data models, testing strategy, performance considerations, and known limitations for Transit Gateway drift detection
- Implementation tasks document with 28 numbered tasks organized into foundation setup, core functions, template parsing, route comparison, command integration, testing, and validation phases
- Decision log documenting 16 key architectural and implementation decisions including VPC route pattern reuse, propagated route exclusion, API choice, error handling strategy, and ECMP limitations

### Added
- Documentation comments for all exported types and functions across cmd, lib, and config packages
- Godoc-compliant comments for DeployInfo, CfnStack, StackEvent, and ResourceEvent types
- Function documentation for stack operations, changeset management, and configuration utilities

### Changed
- Improved code formatting with consistent comment spacing throughout codebase
- Refactored conditional logic in deploy commands using switch statements for better readability
- Enhanced code maintainability with proper documentation following Go best practices

### Changed
- Modernized codebase to use Go 1.25 built-in functions and types across all packages
- Replaced `interface{}` with `any` type throughout codebase for improved readability
- Updated drift detection to use `maps.Copy()` instead of manual map copying loops
- Updated stack operations to use `slices.Contains()` instead of manual slice iteration
- Simplified file conversion functions using modern type aliases
- Cleaned up test files by removing redundant blank imports
- Updated test utilities and assertions to use `any` type
- Modernized template processing functions with cleaner type handling

### Added
- Integration tests for deployment workflows including creation, updates, dry runs, changeset validation, and rollback scenarios
- Integration tests for precheck execution with pass, fail, and stop-on-fail behavior validation
- Integration tests for changeset creation, execution, and handling of empty changesets
- Integration tests for rollback scenarios including new stack failures and update rollbacks
- Test validation script (`test/validate_tests.sh`) for running format checks, unit tests, race detection, and linting
- Test coverage reporting script (`test/coverage_report.sh`) with per-package and weighted coverage analysis
- Test documentation (`test/README.md`) covering testing strategy, patterns, coverage targets, and troubleshooting

### Changed
- Updated CLAUDE.md with integration test documentation including build tags, environment variables, and usage examples
- Updated README.md with development section covering building, testing, linting, and project structure
- Disabled parallel execution for tests using global state (viper configuration and deployFlags)
- Updated `.claude/settings.local.json` to allow execution of coverage reporting script and git rev-parse command

### Added
- Comprehensive unit tests for template body processing with mock S3 clients testing body-only, URL-only, and S3 URL handling
- Unit tests for GetTemplateContents, GetRawTemplateBody, and IsFilePathURI functions covering various input scenarios

### Changed
- Refactored drift detection functions to use interface-based dependency injection for improved testability
- Updated GetDefaultStackDrift to use manual pagination instead of AWS paginator for better test control
- Enhanced drift detection tests with modern Go patterns including map-based table tests, parallel execution, and dedicated mock implementations
- Improved test coverage for drift detection functions including StartDriftDetection, WaitForDriftDetectionToFinish, GetDefaultStackDrift, and GetUncheckedStackResources

### Added (Previous Unreleased)
- Golden file testing framework with Makefile targets for validating and updating output fixtures
- Golden test files for deployment output validation including changesets, events, and stack outputs
- Comprehensive unit tests for deploy helper functions with mock implementations and table-driven patterns
- Test coverage for deployment output formatting including `formatAccountDisplay` and `determineDeploymentMethod`
- Test data directory structure (`cmd/testdata/golden/cmd/`) for maintaining golden file test fixtures

### Changed
- Refactored `showDeploymentInfo` to use extracted helper functions for better testability
- Refactored `runPrechecks` signature to remove unused `cfg` parameter
- Updated `showFailedEvents` to use modern `any` type instead of `interface{}`
- Modernized string operations using `strings.SplitSeq` in `setDeployTags` and `setDeployParameters`
- Enhanced deploy helper functions with extracted utilities: `validateStackReadiness`, `formatAccountDisplay`, `determineDeploymentMethod`

### Added (Previous Unreleased)
- golangci-lint configuration with modern linters (govet, staticcheck, revive, gocritic)
- Unit tests for config package covering GetLCString, GetString, LoadConfigFile, and GetOutputSettings functions
- Unit tests for AWS config operations including GetAWSConfig, GetAccountDetails, GetCallerIdentity, and GetAccountAliases
- Test fixtures for config package in testdata/config/ with valid configurations in YAML, JSON, and TOML formats
- Test fixtures for invalid and minimal configuration scenarios
- Mock implementations for Config, STSClient, and IAMClient interfaces to enable isolated unit testing

  * Added test pattern validator utility to verify test files follow modern Go patterns including table-driven tests, parallel execution, assertion libraries, and proper test helper usage
  * Added comprehensive refactored unit tests for changesets with modern Go testing patterns including table-driven tests, parallel execution, and dependency injection for DeleteChangeset, DeployChangeset, AddChange, GetStack, GenerateChangesetUrl, and GetDangerDetails functions
  * Added comprehensive refactored unit tests for stacks with modern patterns covering GetStack, StackExists, CreateChangeSet, WaitUntilChangesetDone, GetChangeset, GetEvents, DeleteStack, and other stack operations with mock implementations
  * Added CloudFormationCreateChangeSetAPI and CloudFormationDescribeChangeSetAPI interfaces to lib/interfaces.go for improved testability
  * Refactored stack operation functions to use interface-based dependencies (CreateChangeSet, WaitUntilChangesetDone, GetChangeset, DeleteStack, GetEvents) enabling better unit testing with mock implementations
  * Enhanced testutil package with additional builder methods for creating test stacks and mock clients with comprehensive error handling
  * Added interface definitions for improved testability:
    - Added AWS service interfaces in lib/interfaces.go for S3Upload and S3Head operations
    - Added comprehensive config package interfaces in config/interfaces.go including AWSConfigLoader, STSGetCallerIdentityAPI, IAMListAccountAliasesAPI, ConfigReader, and ViperConfigAPI
    - Added extensive unit tests for both lib and config interface implementations with mock clients and error handling scenarios
  * Added comprehensive unit tests for mock client builders in lib/testutil/builders_test.go covering MockCFNClient, MockEC2Client, MockS3Client, StackBuilder, and StackEventBuilder with error injection and builder pattern validation
  * Added comprehensive unit tests for test data fixtures in lib/testutil/fixtures_test.go covering sample templates, configurations, stack responses, changesets, events, and helper functions with fixture consistency validation
  * Added comprehensive test utilities package (lib/testutil) with assertion helpers, test builders, fixtures, golden file testing, and test helpers to improve test maintainability and coverage
  * Added golden file testing framework for validating complex output with automatic update capabilities
  * Added test assertion utilities for common patterns including AWS error handling, stack operations, and changeset validations
  * Added test builders for creating mock AWS resources (stacks, changesets, parameters, tags) with fluent interfaces
  * Added comprehensive test fixtures for CloudFormation templates, deployment files, and configurations
  * Added test helper utilities for temporary files, directories, environment management, and AWS client mocking
  * Added test data files for configuration and template validation
  * Add test coverage improvement specification with comprehensive requirements, design, and implementation plan for achieving 80% test coverage across the codebase
  * Add Claude Code configuration files for AI-assisted development with approved tool permissions and project-specific guidance
  * Configure gitignore to exclude Claude scripts directory
  * Document deploy workflow
  * Add stack utility tests
  * Expand design for refactoring deploy command
  * Refactor deploy command using helper functions
  * Add tests for deploy helper functions
  * Restore deployment comments removed during refactor

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
