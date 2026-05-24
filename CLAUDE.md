# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Testing
- `go test ./...` - Run all tests in the project
- `go test ./... -v` - Run all tests with verbose output
- `go test ./... -cover` - Run tests with coverage report
- `INTEGRATION=1 go test ./...` - Run integration tests (includes unit tests)
- `INTEGRATION=1 go test ./... -v` - Run integration tests with verbose output
- After updating .go files, always run `go fmt` followed by `go test ./...`

#### Integration Tests
Integration tests are skipped during default test runs and require the `INTEGRATION=1` environment variable to execute. Gating is done at runtime via `testutil.SkipIfIntegration(t)` (which checks the `INTEGRATION` env var) — **not** via a `//go:build integration` build tag. This means the integration test files still compile under the default `go test ./...` build (so build/signature drift is caught immediately) while their bodies are skipped unless `INTEGRATION=1` is set. They are located alongside their corresponding source files.

Integration tests validate complex workflows and end-to-end scenarios using mocked AWS clients. They do not require AWS credentials or network access.

**Running integration tests:**
```bash
# Run all tests including integration tests
INTEGRATION=1 go test ./...

# Run only integration tests with verbose output
INTEGRATION=1 go test ./... -v

# Run integration tests for a specific package
INTEGRATION=1 go test ./cmd -v

# Run integration tests with coverage
INTEGRATION=1 go test ./... -cover
```

**Writing integration tests:**
- Call `testutil.SkipIfIntegration(t)` as the first line of each integration test function — this skips the test unless `INTEGRATION=1` is set
- Do **not** add a `//go:build integration` build tag. Without it the tests compile in the default build (catching signature/build drift), while still skipping at runtime
- Use mock clients from `lib/testutil` for AWS service interactions
- Focus on testing workflows and interactions between components

### Build and Run
- `go build` - Build the fog binary
- `go run main.go [command]` - Run fog directly with go run
- `./fog [command]` - Run the compiled binary

### Linting
- The project uses `golangci-lint` v2 in CI/CD (the repo's `.golangci.yml` uses the v2 schema)
- Install locally with: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- Run with: `golangci-lint run` (or `make lint`, which runs a version preflight via `scripts/check-golangci-lint.sh`)

## Architecture Overview

Fog is a CLI tool for managing AWS CloudFormation deployments built with Go and Cobra. The architecture follows a layered approach:

### Core Components

**CMD Layer** (`cmd/`):
- `root.go` - Main Cobra command setup and global flags
- `commands/` - Individual command implementations (deploy, report, etc.)
- `flags/` - Modular flag system with validation groups
- `middleware/` - Request validation, error handling, and recovery
- `registry/` - Command registration and dependency injection
- `services/` - Business logic services (deployment, AWS operations)
- `ui/` - Output formatting and user interaction

**Lib Layer** (`lib/`):
- Core CloudFormation operations (stacks, changesets, drift detection)
- AWS resource management utilities
- File and template processing

**Config Layer** (`config/`):
- Configuration file handling (supports YAML, JSON, TOML)
- AWS configuration management

### Key Architectural Patterns

**Service Layer**: Business logic is organized into services (DeploymentService, AWS clients) with dependency injection through the registry system.

**Flag Groups**: Commands use modular flag groups (`cmd/flags/groups/`) that provide shared validation rules. Each group can be registered with commands and validation is aggregated.

**Error Handling**: Structured error system using `FogError` type with codes and categories for consistent formatting. Errors are handled through middleware.

**Middleware Chain**: Commands go through validation, error handling, and recovery middleware before execution.

**Template Processing**: Supports standard CloudFormation templates (YAML/JSON) and AWS stack deployment files. Templates can use placeholders like `$TEMPLATEPATH`.

### AWS Integration

- Uses AWS SDK v2 for all AWS operations
- Supports multiple output formats (table, CSV, JSON, dot graphs)
- Handles CloudFormation changesets, drift detection, exports, and dependencies
- S3 integration for large template uploads

### Testing Structure

Tests are distributed throughout the codebase with `*_test.go` files:
- Unit tests for individual components
- Integration tests for command workflows
- Mock implementations for AWS services
- The project has extensive test coverage across all layers

### Configuration

Fog uses Viper for configuration management with support for:
- Global config files (`fog.yaml`, `fog.json`, `fog.toml`)
- Environment-specific settings
- AWS profile and region configuration
- Template preprocessing and validation rules