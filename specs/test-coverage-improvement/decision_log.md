# Decision Log: Test Coverage Improvement

## Overview
This document records key decisions made during the requirements gathering phase for improving test coverage in the Fog project.

---

## Decision 1: Coverage Target Strategy

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Initial proposal suggested uniform coverage targets across all packages (80% overall, 75% cmd, 80% config, 75% lib).

### Decision
Adopted a differentiated, pragmatic approach:
- **Overall target**: 80% (weighted by package size)
- **lib package**: 80% target (primary focus, up from 66.6%)
- **config package**: 80% target (up from 0.0%)
- **cmd package**: Focus on helper functions only, exclude large orchestration functions

### Rationale
- The lib package contains core business logic and should have the highest quality bar
- Config package is small and pure logic, easy to achieve 80%
- Cmd package has large orchestration functions (350+ lines) that are untestable without major refactoring
- Testing helper functions in cmd provides better ROI than attempting to test large Cobra command orchestrators

### Consequences
- Need to identify and document which cmd functions are excluded from coverage
- Large orchestration functions will be flagged for future refactoring
- More realistic and achievable targets

---

## Decision 2: Dependency Injection with Interfaces

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Initial proposal suggested "minimal interface abstractions" but was unclear about the implementation approach. The existing codebase uses global function variables for mocking in some tests.

### Decision
Use dependency injection with small, focused interfaces throughout the codebase.

### Rationale
- Provides proper testability without requiring AWS credentials
- Small, focused interfaces are easier to mock and maintain
- Aligns with Go best practices (accept interfaces, return structs)
- Public API changes to accept interfaces are acceptable architectural improvements
- More sustainable than global function variable swapping

### Consequences
- Function signatures will change to accept interface parameters
- May affect existing code that calls these functions
- Requires defining interfaces in packages that use them
- Breaking changes to exported functions are acceptable for this improvement

---

## Decision 3: Testing Approach by Package

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Different packages have different characteristics and testing needs.

### Decision
Adopt package-specific testing strategies:

**lib package:**
- Comprehensive unit tests
- Dependency injection with interfaces
- 80% coverage target
- Focus on testing business logic

**cmd package:**
- Unit tests for helper functions
- Integration tests with build tags for complex workflows
- Golden file testing for output validation
- Exclude large orchestration functions from coverage requirements

**config package:**
- Pure unit tests
- Mock AWS SDK configuration
- Test fixtures in testdata/config/
- 80% coverage target

### Rationale
- Each package has different testability characteristics
- lib is pure business logic and highly testable
- cmd has heavy framework integration and orchestration
- config is small and focused on parsing/loading

### Consequences
- Need to create test infrastructure (testutil package, testdata directory)
- Need to implement golden file testing support
- Integration tests must use build tags and INTEGRATION environment variable
- Documentation must explain testing strategy per package

---

## Decision 4: No AWS Authentication Required

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Tests need to run in local development and CI/CD environments without AWS credentials.

### Decision
All tests must run without AWS authentication:
- Unit tests use mocked AWS clients via dependency injection
- Integration tests use build tags and are excluded from default runs
- No network calls to AWS services

### Rationale
- Enables local development without AWS access
- Simplifies CI/CD setup (no credential management)
- Faster test execution (no network calls)
- More reliable tests (no external dependencies)
- Aligns with unit testing best practices

### Consequences
- Must mock all AWS SDK clients
- Integration tests requiring real AWS are excluded from regular runs
- Need comprehensive mocking infrastructure
- May need to verify real AWS behavior manually or in separate test environments

---

## Decision 5: Uplift All Existing Tests

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
The codebase has 12 existing test files with varying patterns and quality. Initial proposal was unclear about whether to refactor existing tests.

### Decision
All existing tests SHALL be uplifted to follow the new standards and best practices.

### Rationale
- Ensures consistency across the entire test suite
- Provides opportunity to improve existing test quality
- Prevents technical debt accumulation
- Makes the codebase easier to maintain long-term
- User explicitly stated: "All existing tests should be uplifted"

### Consequences
- More work than only adding new tests
- Existing tests in lib/stacks_test.go, cmd/deploy_helpers_test.go, etc. need refactoring
- Need to convert slice-based table tests to map-based
- Need to update naming conventions and patterns
- All changes must not break existing test functionality

---

## Decision 6: Test Infrastructure Requirements

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** Design Review

### Context
Need shared infrastructure to support testing across packages without duplication.

### Decision
Create comprehensive test infrastructure:
- **testutil package** at `lib/testutil` for shared test utilities
- **testdata directory** at project root for test fixtures
- **Golden file testing** support with update flag
- **Mock builders** for common AWS client patterns

### Rationale
- Reduces test code duplication
- Provides consistent testing patterns
- Makes tests easier to write and maintain
- Golden file testing ideal for complex output validation (changesets, drift reports)
- Centralized mock builders ensure consistency

### Consequences
- Need to design and implement testutil package API
- Need to organize testdata directory structure
- Need to implement golden file test helper functions
- May need to migrate existing test helpers to testutil

---

## Decision 7: Map-Based Table-Driven Tests

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** Requirements (Go Best Practices)

### Context
Existing tests use slice-based table tests. Go best practices recommend map-based tables.

### Decision
All table-driven tests SHALL use `map[string]struct` instead of slice-based tables.

### Rationale
- Ensures unique test case names (compiler error for duplicates)
- Catches test interdependencies (map iteration is randomized)
- Aligns with modern Go testing best practices
- Forces developers to create descriptive test names

### Consequences
- All existing slice-based table tests need conversion
- Test execution order becomes non-deterministic (good for catching dependencies)
- Slightly more verbose syntax than slices
- Need to educate team on pattern if not familiar

---

## Decision 8: Small, Focused Interfaces

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Initial proposal mentioned "minimal interfaces" but didn't specify size or scope.

### Decision
Interfaces SHALL be small and focused (ideally single-method or single-responsibility).

### Rationale
- Easier to understand and maintain
- Easier to mock in tests
- Aligns with Interface Segregation Principle
- Follows Go conventions (e.g., `io.Reader`, `io.Writer`)
- Reduces coupling between components

### Consequences
- May need multiple interfaces instead of one large interface
- Functions may accept multiple interface parameters
- Need to document interface patterns for consistency
- AWS SDK already provides focused interfaces (e.g., `CloudFormationDescribeStacksAPI`)

---

## Decision 9: Exclude Large Orchestration Functions

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** User

### Context
Some cmd functions exceed 350 lines and are primarily AWS API orchestration with Cobra integration.

### Decision
Large orchestration functions (350+ lines, >80% AWS calls) SHALL be:
- Flagged for future refactoring
- Excluded from coverage requirements
- Documented with justification

### Rationale
- Testing these functions without refactoring provides minimal value
- Would require extensive mocking of Cobra framework and AWS SDK
- Tests would be brittle and hard to maintain
- User explicitly stated: "flag the larger functions in the cmd package as requiring refactoring and skip testing them for now"
- Better to focus effort on testable business logic

### Consequences
- Need to identify and document excluded functions
- Coverage metrics won't include these functions
- Future work item to refactor these functions
- May require extracting testable logic into separate functions over time

---

## Decision 10: Use github.com/google/go-cmp for Comparisons

**Date:** 2025-10-02
**Status:** Accepted
**Decision Maker:** Requirements (Go Best Practices)

### Context
Complex struct comparisons need better tooling than reflect.DeepEqual.

### Decision
Tests SHALL use `github.com/google/go-cmp/cmp` for complex struct comparisons.

### Rationale
- Provides clear, readable diff output on test failures
- Handles unexported fields, nil values, and complex types better
- Industry standard for Go testing
- Better error messages than reflect.DeepEqual
- Supports custom comparers for special types

### Consequences
- Need to add dependency to go.mod
- Need to update existing tests using reflect.DeepEqual
- Need to document cmp usage patterns for team
- Test output becomes more verbose but more informative

---

## Resolved Questions

### Q1: Integration Test Scope
Should we define specific scenarios for integration tests, or leave this to the implementation phase?

**Status:** Resolved - Define during design phase
**Decision:** Integration test scenarios will be defined during the design phase when we have better understanding of what needs testing.

### Q2: Golden File Update Mechanism
Should golden file update use a flag like `-update` or an environment variable like `UPDATE_GOLDEN=true`?

**Status:** Resolved - Use best practice
**Decision:** Follow Go community best practices for golden file testing (typically `-update` flag pattern used by tools like `go test -update`).

### Q3: Coverage Enforcement in CI
Should CI fail builds if coverage drops below targets?

**Status:** Resolved - Report only
**Decision:** CI should report coverage metrics but not fail builds. This allows gradual improvement and avoids blocking legitimate changes.

### Q4: Testutil Package API Design
What specific helpers and builders should testutil provide?

**Status:** Resolved - Discover during design/implementation
**Decision:** The testutil package API will be discovered and designed during the design and implementation phases based on common patterns and actual needs.

---

## Rejected Alternatives

### Alternative 1: Use testify/mock Framework
**Rejected:** User explicitly requested no third-party mocking frameworks
**Reasoning:** Adds dependency, overkill for simple mocking needs, function-based mocks sufficient

### Alternative 2: Test All Cmd Functions Equally
**Rejected:** User requested pragmatic approach focusing on helpers
**Reasoning:** Large orchestration functions untestable without major refactoring, low ROI

### Alternative 3: Keep Existing Test Patterns
**Rejected:** User explicitly requested uplifting all existing tests
**Reasoning:** Creates inconsistency, perpetuates technical debt, prevents standardization

### Alternative 4: Allow Tests to Require AWS Credentials
**Rejected:** User explicitly requested no AWS authentication
**Reasoning:** Complicates local dev and CI, slows tests, adds external dependencies

### Alternative 5: Focus on Cmd Package First
**Rejected:** User requested lib package as primary focus
**Reasoning:** Lib has core business logic, cmd has orchestration challenges, lib provides better ROI
