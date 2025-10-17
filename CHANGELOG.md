Unreleased
===========

### Added
- go-output v2 specification and research documentation in specs/go-output-v2 directory
- API documentation covering all v2 public interfaces and agent implementation patterns
- Migration guide detailing v1 to v2 upgrade path with breaking changes and code examples
- Evaluation document comparing v2 against alternatives (pterm, lipgloss, glamour, charmbracelet/log)
- Design document outlining v2 architecture, threading model, and collapsible content system
- Requirements specification defining functional, technical, and user experience requirements
- Task breakdown with sprint planning for go-output v2 implementation
- Decision log tracking architectural and design choices

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
