# Fog Testing Guide

This document describes the testing strategy, patterns, and procedures for the Fog project.

## Table of Contents

- [Overview](#overview)
- [Running Tests](#running-tests)
- [Testing Strategy](#testing-strategy)
- [Test Patterns](#test-patterns)
- [Coverage Targets](#coverage-targets)
- [Test Infrastructure](#test-infrastructure)
- [Writing New Tests](#writing-new-tests)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)

## Overview

Fog uses a comprehensive testing approach with:
- **Unit tests** for core business logic in the `lib` and `config` packages
- **Integration tests** with build tags for complex workflows
- **Golden file tests** for output validation
- **Mock-based testing** using dependency injection with focused interfaces

All tests run without requiring AWS credentials, enabling local development and CI/CD testing.

## Running Tests

### Basic Test Commands

```bash
# Run all unit tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Run tests with race detection
go test -race ./...

# Run integration tests (requires INTEGRATION=1)
INTEGRATION=1 go test ./...
```

### Validation Scripts

The project provides scripts for comprehensive test validation:

```bash
# Run complete validation suite (formatting, tests, race detection, linting)
./test/validate_tests.sh

# Generate detailed coverage report
./test/coverage_report.sh
```

### Coverage Reports

```bash
# Generate HTML coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# View coverage in browser
open coverage.html
```

## Testing Strategy

### Package-Specific Approaches

#### lib Package (Target: 80% coverage)
- **Approach**: Pure unit testing with dependency injection
- **Focus**: Core business logic, CloudFormation operations, helper functions
- **Patterns**: Map-based table-driven tests, mock AWS clients via interfaces
- **Key Files**: All `*_test.go` files in `lib/`

#### config Package (Target: 80% coverage)
- **Approach**: Pure unit testing with mock file system and AWS config
- **Focus**: Configuration loading, parsing, validation
- **Patterns**: Test fixtures in `testdata/config/`, multiple format support
- **Key Files**: `config/config_test.go`, `config/awsconfig_test.go`

#### cmd Package (Target: 75% for helper functions)
- **Approach**: Unit tests for helpers, integration tests for workflows
- **Focus**: Testable helper functions, validation logic
- **Exclusions**: Large orchestration functions (350+ lines) flagged for refactoring
- **Patterns**: Golden file testing for output, build tags for integration tests
- **Key Files**: `cmd/deploy_helpers_test.go`, `cmd/deploy_integration_test.go`

### Integration Testing

Integration tests use build tags and are excluded from default test runs:

```go
//go:build integration
// +build integration

package cmd

func TestDeployIntegration(t *testing.T) {
    if os.Getenv("INTEGRATION") != "1" {
        t.Skip("Skipping integration test")
    }
    // Test implementation
}
```

Run integration tests with:
```bash
INTEGRATION=1 go test ./...
```

## Test Patterns

### Map-Based Table-Driven Tests

All tests use map-based tables for uniqueness and randomization:

```go
func TestFunction(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        input   string
        want    string
        wantErr bool
    }{
        "valid input": {
            input: "test",
            want:  "TEST",
        },
        "empty input": {
            input:   "",
            wantErr: true,
        },
    }

    for name, tc := range tests {
        tc := tc // Capture range variable
        t.Run(name, func(t *testing.T) {
            t.Parallel()

            got, err := Function(tc.input)

            if tc.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tc.want, got)
        })
    }
}
```

### Mock Implementations

Mocks are structs with function fields:

```go
type mockCFNClient struct {
    describeStacksFn func(context.Context, *cloudformation.DescribeStacksInput, ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error)
}

func (m *mockCFNClient) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
    if m.describeStacksFn != nil {
        return m.describeStacksFn(ctx, params, optFns...)
    }
    return &cloudformation.DescribeStacksOutput{}, nil
}
```

### Golden File Testing

For complex output validation:

```go
func TestOutput(t *testing.T) {
    golden := testutil.NewGoldenFile(t)

    output := generateOutput()

    golden.Assert("test-output", []byte(output))
}

// Update golden files with:
go test ./... -update
```

### Dependency Injection

Functions accept interfaces for testability:

```go
// Before
func GetStack(stackName string) (*Stack, error) {
    client := cloudformation.NewFromConfig(cfg)
    // ...
}

// After
func GetStack(ctx context.Context, client CloudFormationDescribeStacksAPI, stackName string) (*Stack, error) {
    // ...
}
```

## Coverage Targets

| Package | Target | Current | Status |
|---------|--------|---------|--------|
| lib     | 80%    | 74.5%   | ⚠️ Need +5.5% |
| config  | 80%    | 59.0%   | ⚠️ Need +21.0% |
| cmd (helpers) | 75% | N/A | 📝 Helper functions only |
| Overall | 80%    | 41.7%   | ⚠️ In progress |

### Excluded Functions

The following functions are excluded from coverage requirements:

1. **cmd/deploy.go (409 lines)**
   - `deployTemplate()` - Large orchestration function
   - Reason: >80% AWS API calls, heavy Cobra integration
   - Recommendation: Refactor into smaller functions

2. **cmd/report.go (360 lines)**
   - `reportCmd()` - Complex reporting orchestration
   - Reason: Heavy AWS API usage, output formatting
   - Recommendation: Extract report generation logic

3. **cmd/drift.go**
   - `detectDrift()` - Large orchestration function
   - Reason: Minimal testable logic, heavy AWS integration
   - Recommendation: Extract drift analysis into lib

4. **cmd/describe_changeset.go**
   - `describeChangeset()`, `showChangeset()`
   - Reason: Cobra command orchestration
   - Recommendation: Extract formatting into lib

5. **Other cmd main command functions**
   - Reason: Framework integration code
   - Recommendation: Focus on helper functions

## Test Infrastructure

### testutil Package (`lib/testutil/`)

Shared testing utilities:

- **builders.go** - Mock AWS client builders
- **fixtures.go** - Test data builders with defaults
- **golden.go** - Golden file testing support
- **helpers.go** - Common test helpers
- **assertions.go** - Custom assertion utilities

Example usage:

```go
// Mock client builder
mockClient := testutil.NewMockCFNClient().
    WithStack(testutil.NewStackBuilder("test-stack").
        WithStatus(types.StackStatusCreateComplete).
        Build())

// Golden file testing
golden := testutil.NewGoldenFile(t)
golden.Assert("output", []byte(actualOutput))
```

### testdata Directory

Test fixtures and golden files:

```
testdata/
├── templates/        # Sample CloudFormation templates
├── config/          # Configuration file fixtures
├── golden/          # Golden files for output validation
└── fixtures/        # Other test data
```

## Writing New Tests

### 1. Create Test File

Place `*_test.go` files alongside source code:

```bash
# For lib/newfile.go
lib/newfile_test.go

# For config/config.go
config/config_test.go
```

### 2. Use Map-Based Tables

```go
func TestNewFunction(t *testing.T) {
    t.Parallel()

    tests := map[string]struct {
        // Test case fields
    }{
        "descriptive name": {
            // Test case
        },
    }

    for name, tc := range tests {
        tc := tc
        t.Run(name, func(t *testing.T) {
            t.Parallel()
            // Test implementation
        })
    }
}
```

### 3. Add Helper Functions

```go
func setupTest(t *testing.T) *testContext {
    t.Helper()
    // Setup code
}
```

### 4. Use Modern Patterns

- ✅ Use `any` instead of `interface{}`
- ✅ Use `t.Cleanup()` instead of `defer`
- ✅ Use `got`/`want` naming convention
- ✅ Use `cmp.Diff()` for struct comparisons
- ✅ Call `t.Parallel()` where safe
- ✅ Call `t.Helper()` in test helpers

### 5. Mock Dependencies

```go
mockClient := &mockService{
    operationFn: func(...) (..., error) {
        return expectedResult, nil
    },
}

result := functionUnderTest(mockClient)
```

## Common Patterns

### Testing Error Cases

```go
tests := map[string]struct {
    setup   func(*mock)
    wantErr bool
    errMsg  string
}{
    "API error": {
        setup: func(m *mock) {
            m.WithError(errors.New("API error"))
        },
        wantErr: true,
        errMsg:  "API error",
    },
}

// In test
if tc.wantErr {
    require.Error(t, err)
    if tc.errMsg != "" {
        assert.Contains(t, err.Error(), tc.errMsg)
    }
    return
}
```

### Comparing Complex Structs

```go
if diff := cmp.Diff(want, got); diff != "" {
    t.Errorf("mismatch (-want +got):\n%s", diff)
}
```

### Testing with Context

```go
ctx := context.Background()
result, err := Function(ctx, client, params)
```

### Parallel Testing

```go
func TestParallel(t *testing.T) {
    t.Parallel() // Test function runs in parallel

    for name, tc := range tests {
        tc := tc // Capture variable!
        t.Run(name, func(t *testing.T) {
            t.Parallel() // Subtest runs in parallel
            // Test implementation
        })
    }
}
```

**Note**: Don't use `t.Parallel()` if the test modifies global state (e.g., `viper` settings, global variables).

## Troubleshooting

### Race Conditions

```bash
# Run race detector
go test -race ./...

# Common causes:
# - Tests with t.Parallel() modifying global variables
# - Concurrent access to shared test data
# - Solution: Remove t.Parallel() or protect shared state
```

### Failed Tests

```bash
# Run specific test
go test ./lib -run TestFunctionName

# Run with verbose output
go test ./lib -v -run TestFunctionName

# Debug with print statements
t.Logf("Debug info: %v", value)
```

### Coverage Issues

```bash
# Check coverage for specific package
go test ./lib -cover

# See which lines are not covered
go test ./lib -coverprofile=lib.out
go tool cover -html=lib.out
```

### Golden File Mismatches

```bash
# Update golden files
go test ./... -update

# Compare manually
diff testdata/golden/expected.golden actual.txt
```

### Integration Test Issues

```bash
# Ensure INTEGRATION=1 is set
INTEGRATION=1 go test ./...

# Check build tags
grep -r "//go:build integration" .

# Verify skip logic
if os.Getenv("INTEGRATION") != "1" {
    t.Skip("Skipping integration test")
}
```

## Best Practices

1. **Write tests first** - Consider TDD for new features
2. **Test behavior, not implementation** - Focus on what, not how
3. **Keep tests simple** - Easy to understand and maintain
4. **Use descriptive names** - Test names should explain what's being tested
5. **Test edge cases** - Empty strings, nil values, boundary conditions
6. **Mock external dependencies** - Never call real AWS APIs in unit tests
7. **Avoid test interdependence** - Tests should be independent and isolated
8. **Clean up resources** - Use `t.Cleanup()` for resource cleanup
9. **Document complex scenarios** - Add comments for non-obvious test cases
10. **Run tests frequently** - Before commits, after changes

## CI/CD Integration

Tests run automatically in CI/CD:

```bash
# CI pipeline runs
go fmt ./...
go test ./...
go test -race ./...
golangci-lint run
```

Ensure all these pass locally before pushing:

```bash
./test/validate_tests.sh
```

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [go-cmp Documentation](https://github.com/google/go-cmp)
- [Fog Design Document](../specs/test-coverage-improvement/design.md)
- [Fog Requirements](../specs/test-coverage-improvement/requirements.md)
