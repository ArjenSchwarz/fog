Unreleased
===========

  * Document deploy workflow
  * Use interfaces for CloudFormation exports functions and add tests
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
