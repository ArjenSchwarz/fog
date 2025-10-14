# Test Coverage Improvement Requirements

## Introduction

This feature focuses on improving the test coverage of the Fog codebase from the current **37.3%** to at least **80%** overall, while simultaneously uplifting all existing tests to follow Go testing best practices.

**Coverage Strategy:**
- **lib package**: Primary focus with 80% target (up from 66.6%) - comprehensive unit testing with dependency injection
- **cmd package**: Focus on helper functions and testable business logic; large orchestration functions (350+ lines) flagged for future refactoring and excluded from coverage targets
- **config package**: 80% target (up from 0.0%) - pure unit testing

**Testing Approach:**
- **lib**: Unit tests with small, focused interface abstractions for AWS SDK mocking via dependency injection
- **cmd**: Combination of unit tests for helpers, integration tests with build tags, and golden file testing for complex output validation
- **All tests**: Must run without AWS authentication to enable local development and CI/CD

**Architectural Changes:**
- Introduce dependency injection with small, focused interfaces for AWS clients and external dependencies
- Extract testable helper functions from large command orchestration functions where beneficial
- Create test infrastructure (testutil package, testdata directory, golden file support)

---

## Requirements

### 1. Coverage Baseline and Targets

**User Story:** As a developer, I want to establish clear coverage targets per package, so that I can measure progress and ensure comprehensive testing across the codebase.

**Acceptance Criteria:**
1.1. The system SHALL achieve a minimum of 80% overall test coverage across all packages (weighted by package size)
1.2. The `lib` package SHALL achieve a minimum of 80% coverage (up from 66.6%) as the primary testing focus
1.3. The `config` package SHALL achieve a minimum of 80% coverage (up from 0.0%)
1.4. The `cmd` package SHALL focus on helper functions with a pragmatic approach to coverage:
   - Helper functions (e.g., `deploy_helpers.go`) SHALL have 75%+ coverage
   - Large orchestration functions (350+ lines) SHALL be flagged for refactoring and excluded from coverage requirements
   - Testable business logic SHALL be prioritized over AWS API orchestration code
1.5. Coverage SHALL be measured using `go test ./... -cover` and reported per-package
1.6. The project SHALL maintain a list of excluded functions with justification (size, orchestration-heavy, requires refactoring)

### 2. Interface Extraction for Dependency Injection

**User Story:** As a developer, I want to use dependency injection with focused interfaces, so that I can write testable code with mockable dependencies.

**Acceptance Criteria:**
2.1. The system SHALL use dependency injection for AWS SDK clients and external dependencies
2.2. Interfaces SHALL be small and focused (ideally single-method or single-responsibility)
2.3. Interfaces SHALL be defined in the package that uses them (not in separate interface packages)
2.4. Interface names SHALL follow Go conventions (e.g., `CloudFormationDescribeStacksAPI` already exists)
2.5. Functions SHALL accept interface parameters instead of concrete AWS SDK types
2.6. The implementation SHALL document interface extraction patterns for consistency
2.7. Public APIs MAY change to accept interfaces, as this is an acceptable architectural improvement

### 3. Test Structure and Organization

**User Story:** As a developer, I want tests organized following Go best practices, so that the test suite is maintainable and follows community standards.

**Acceptance Criteria:**
3.1. All test files SHALL use the `*_test.go` naming convention and be placed alongside source files
3.2. Tests SHALL use map-based table-driven testing with `map[string]struct` for test cases
3.3. Each test case SHALL use `t.Run(name, func(t *testing.T) {...})` for subtests
3.4. Test case names SHALL clearly describe what is being tested
3.5. Tests SHALL use the `tc` variable name for individual test cases in table-driven tests
3.6. Helper functions SHALL call `t.Helper()` as their first statement
3.7. ALL existing tests SHALL be uplifted to follow these patterns

### 4. Modern Go Testing Patterns

**User Story:** As a developer, I want tests to use modern Go idioms and patterns, so that the codebase reflects current best practices and avoids outdated patterns.

**Acceptance Criteria:**
4.1. Tests SHALL use `any` instead of `interface{}` for type parameters
4.2. Tests SHALL use `t.Cleanup()` over `defer` for resource cleanup
4.3. Tests SHALL use the `got`/`want` naming convention for actual vs expected values
4.4. Tests SHALL use `github.com/google/go-cmp/cmp` for complex struct comparisons
4.5. Parallel tests SHALL call `t.Parallel()` as the first statement in the test function
4.6. Loop variables SHALL be properly captured before use in parallel subtests
4.7. ALL existing tests SHALL be refactored to use modern patterns where applicable

### 5. Mock Implementation Standards

**User Story:** As a developer, I want consistent, minimal mocking strategies, so that tests are simple to understand and maintain without heavy dependencies.

**Acceptance Criteria:**
5.1. Mock implementations SHALL implement the specific focused interfaces defined for dependency injection
5.2. Mocks SHALL be structs with function fields for flexibility (e.g., `type mockClient struct { describeStacksFn func(...) (...) }`)
5.3. The system SHALL NOT introduce third-party mocking frameworks (e.g., testify/mock, gomock)
5.4. Mock clients SHALL be defined in `*_test.go` files or in a `testutil` package if widely reused
5.5. Each mock SHALL have a clear, descriptive type name (e.g., `mockDescribeStacksClient`)
5.6. Mocks SHALL allow error injection for testing failure scenarios
5.7. Existing mocks SHALL be refactored to follow these patterns

### 6. Test Infrastructure and Helpers

**User Story:** As a developer, I want shared test infrastructure and helpers, so that I can write tests efficiently without duplicating setup code.

**Acceptance Criteria:**
6.1. The system SHALL create a `testutil` package under `lib/testutil` for shared test utilities
6.2. The `testutil` package SHALL include:
   - Common mock builders (e.g., `NewMockStacksClient()`)
   - Test data builders with sensible defaults
   - Assertion helpers for common patterns
6.3. The system SHALL create a `testdata/` directory at the project root for test fixtures
6.4. The `testdata/` directory SHALL contain:
   - Sample CloudFormation templates
   - Sample deployment configuration files
   - Expected output files for golden file testing
6.5. The system SHALL implement golden file testing support for complex output validation
6.6. Golden file tests SHALL support an update flag (e.g., `-update`) to regenerate expected outputs

### 7. Command Package Testing Strategy

**User Story:** As a developer, I want a pragmatic testing approach for the cmd package, so that I can achieve meaningful coverage without testing untestable orchestration code.

**Acceptance Criteria:**
7.1. The system SHALL identify and document large orchestration functions requiring refactoring:
   - Functions exceeding 350 lines
   - Functions with >80% AWS API calls
   - Functions with complex Cobra framework integration
7.2. Helper functions (e.g., in `deploy_helpers.go`) SHALL have comprehensive unit test coverage
7.3. Testable business logic SHALL be extracted into testable functions where beneficial (minimal refactoring)
7.4. Integration tests SHALL use build tags (`// +build integration`) and run only when `INTEGRATION=true`
7.5. Complex output formatting SHALL use golden file tests with fixtures in `testdata/`
7.6. Tests SHALL use mock AWS clients that don't require authentication
7.7. The system SHALL document which cmd functions are excluded from coverage with justification

### 8. Config Package Testing

**User Story:** As a developer, I want comprehensive tests for the config package, so that configuration loading and parsing is well-validated.

**Acceptance Criteria:**
8.1. The `config` package SHALL have tests for AWS configuration loading (`awsconfig.go`)
8.2. The `config` package SHALL have tests for configuration file parsing (`config.go`)
8.3. Tests SHALL use test fixtures in `testdata/config/` for configuration files
8.4. Tests SHALL mock AWS SDK configuration loading to avoid authentication requirements
8.5. Tests SHALL cover multiple configuration formats (YAML, JSON, TOML)
8.6. Tests SHALL validate error handling for invalid configurations
8.7. Tests SHALL achieve 80%+ coverage for the config package

### 9. Lib Package Testing

**User Story:** As a developer, I want comprehensive unit tests for the lib package, so that core business logic is well-validated.

**Acceptance Criteria:**
9.1. The `lib` package SHALL achieve 80%+ coverage with unit tests
9.2. AWS SDK clients SHALL be injected via small, focused interfaces
9.3. Each lib file SHALL have a corresponding `*_test.go` file
9.4. Tests SHALL cover success paths, error paths, and edge cases
9.5. Complex CloudFormation operations SHALL have detailed test coverage
9.6. Helper functions (parsing, formatting, data transformation) SHALL have 90%+ coverage
9.7. Existing lib tests SHALL be uplifted to modern patterns

### 10. Test Quality and Assertions

**User Story:** As a developer, I want meaningful test assertions, so that failures provide clear information about what went wrong.

**Acceptance Criteria:**
10.1. Tests SHALL NOT be assertion-free (no tests written solely for coverage metrics)
10.2. Error assertions SHALL check both error presence and error messages when relevant
10.3. Complex struct comparisons SHALL use `cmp.Diff()` and report differences clearly
10.4. Tests SHALL verify state changes, not just absence of errors
10.5. Mock call verification SHALL check that expected interactions occurred where relevant
10.6. Test failures SHALL provide context about what was being tested

### 11. Test Documentation

**User Story:** As a developer, I want clear test documentation, so that future maintainers understand test intent and coverage strategy.

**Acceptance Criteria:**
11.1. Test functions SHALL have comments describing what they test (especially for complex scenarios)
11.2. Complex test setups SHALL include inline comments explaining the scenario
11.3. Test files exceeding 500 lines SHALL be split by functionality (e.g., `stacks_parsing_test.go`, `stacks_validation_test.go`)
11.4. The system SHALL document architectural changes made for testability
11.5. The system SHALL provide a coverage report with excluded functions and justifications
11.6. Tests SHALL follow the naming convention `Test<FunctionName>` or `Test<FunctionName>_<Scenario>`

### 12. No AWS Authentication Required

**User Story:** As a developer, I want to run all tests without AWS credentials, so that I can develop locally and run tests in CI/CD without AWS access.

**Acceptance Criteria:**
12.1. All unit tests SHALL run without AWS credentials or authentication
12.2. AWS SDK clients SHALL be mocked using dependency injection and interfaces
12.3. Integration tests requiring AWS SHALL use build tags and be excluded from default test runs
12.4. Mock implementations SHALL simulate AWS API responses without network calls
12.5. Tests SHALL NOT depend on external services (AWS, S3, CloudFormation)
12.6. CI/CD SHALL be able to run `go test ./...` without AWS configuration

### 13. Validation and Quality Gates

**User Story:** As a developer, I want automated validation of test quality, so that the test suite maintains high standards over time.

**Acceptance Criteria:**
13.1. All tests SHALL pass with `go test ./...` before completion
13.2. All test code SHALL be formatted with `go fmt`
13.3. The system SHALL run `golangci-lint` on all code (including tests)
13.4. Coverage SHALL be verified with `go test ./... -cover`
13.5. Tests SHALL not introduce race conditions (verified with `go test -race ./...`)
13.6. The system SHALL generate a coverage report identifying gaps and excluded functions

### 14. Preservation of Existing Functionality

**User Story:** As a developer, I want to ensure that adding tests does not break existing functionality, so that the codebase remains stable.

**Acceptance Criteria:**
14.1. All existing tests SHALL continue to pass after modifications
14.2. Changes to production code SHALL be limited to:
   - Interface extraction and dependency injection
   - Extracting small helper functions where beneficial
   - Adding missing error handling where discovered
14.3. Public APIs MAY change to accept interfaces (acceptable architectural improvement)
14.4. Behavior of existing commands SHALL remain unchanged
14.5. Configuration handling SHALL maintain backward compatibility
14.6. The system SHALL flag any architectural changes that exceed "small, focused interface extraction"
