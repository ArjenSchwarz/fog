# CMD Folder Refactoring Plan

## Overview

This document outlines a comprehensive refactoring plan for improving the code quality, readability, and flexibility of the `cmd` folder in the fog CloudFormation management tool.

## Current State Analysis

The `cmd` folder contains a Cobra-based CLI application with several architectural issues:

### Key Problems Identified

1. **Monolithic Files**: Files like `deploy.go` (500+ lines) and `drift.go` contain too many responsibilities
2. **Mixed Abstractions**: Business logic mixed with UI/presentation code
3. **Global State**: Heavy reliance on global variables (`deployFlags`, `outputsettings`, etc.)
4. **Poor Testability**: Direct dependencies on AWS clients and file system operations
5. **Code Duplication**: Similar patterns repeated across commands
6. **Inconsistent Error Handling**: Mix of `log.Fatal`, `os.Exit`, and error returns

### Current File Structure

```
cmd/
├── demo.go                    # Demo command
├── demosettings.go           # Demo settings
├── dependencies.go           # Dependencies listing
├── deploy_helpers_test.go    # Deploy helper tests
├── deploy_helpers.go         # Deploy helper functions
├── deploy.go                 # Main deploy command (monolithic)
├── describe_changeset.go     # Changeset description
├── describe.go               # Stack description
├── drift.go                  # Drift detection (monolithic)
├── exports.go                # Export listing
├── flaggroups.go             # Flag group definitions
├── groups.go                 # Command group setup
├── helpers.go                # Utility functions
├── history.go                # Deployment history
├── report.go                 # Stack reporting
├── resources.go              # Resource listing
├── root.go                   # Root command setup
├── tables.go                 # Table demo
└── version.go                # Version command
```

## Refactoring Objectives

### Primary Goals

- **Improve Code Quality**: Reduce complexity, eliminate duplication, standardize patterns
- **Enhance Readability**: Smaller, focused files with clear responsibilities
- **Increase Flexibility**: Pluggable services and configurable behavior
- **Better Testability**: Mockable dependencies and isolated business logic
- **Consistent Error Handling**: Structured error types with proper error chains

### Target Architecture

```
cmd/
├── commands/          # Individual command implementations
├── services/          # Business logic services
├── ui/               # User interface components
├── registry/         # Command registration and routing
├── middleware/       # Common middleware (validation, error handling)
└── testing/          # Test utilities and mocks
```

## Refactoring Tasks

This refactoring is broken down into 6 parallel tasks that can be worked on independently:

### Task 1: Command Structure Reorganization
**Goal**: Clean separation of concerns and consistent command patterns
- **File**: `cmd-refactor-task-1-command-structure-reorg.md`
- **Priority**: High (Foundation for other tasks)
- **Estimated Effort**: Medium

### Task 2: Business Logic Extraction
**Goal**: Separate business logic from CLI presentation
- **File**: `cmd-refactor-task-2-business-logic-extraction.md`
- **Priority**: High
- **Estimated Effort**: Large

### Task 3: Flag Management Refactoring
**Goal**: Improve flag handling consistency and validation
- **File**: `cmd-refactor-task-3-flag-management.md`
- **Priority**: Medium
- **Estimated Effort**: Small

### Task 4: Output and UI Standardization
**Goal**: Consistent user interface patterns
- **File**: `cmd-refactor-task-4-output-ui-standardization.md`
- **Priority**: Medium
- **Estimated Effort**: Medium

### Task 5: Error Handling Improvement
**Goal**: Consistent and testable error handling
- **File**: `cmd-refactor-task-5-error-handling.md`
- **Priority**: High
- **Estimated Effort**: Medium

### Task 6: Testing Infrastructure
**Goal**: Make the code more testable
- **File**: `cmd-refactor-task-6-testing-infrastructure.md`
- **Priority**: High
- **Estimated Effort**: Large

## Implementation Strategy

### Phase 1: Foundation (Tasks 1, 5)
- Command structure reorganization
- Error handling improvement
- Creates stable foundation for other changes

### Phase 2: Core Services (Tasks 2, 6)
- Business logic extraction
- Testing infrastructure
- Builds on foundation to create testable services

### Phase 3: User Experience (Tasks 3, 4)
- Flag management refactoring
- Output and UI standardization
- Improves user-facing aspects using refactored services

### Migration Approach

1. **Maintain Backward Compatibility**: All existing CLI commands continue to work
2. **Gradual Transition**: New structure introduced alongside existing code
3. **Integration Testing**: Ensure no regressions in functionality
4. **Feature Flags**: Use configuration to enable new behavior gradually

## Expected Benefits

### Code Quality Improvements
- **Reduced Complexity**: Smaller, focused files (target: <200 lines per file)
- **Eliminated Duplication**: Shared services and utilities
- **Consistent Patterns**: Standardized approaches across commands

### Maintainability Enhancements
- **Clear Separation of Concerns**: Business logic separated from UI
- **Better Error Handling**: Structured errors with context
- **Improved Documentation**: Self-documenting code structure

### Testing Improvements
- **Mockable Dependencies**: Testable business logic
- **Isolated Components**: Unit testable services
- **Integration Test Support**: Testable command flows

### Developer Experience
- **Easier Onboarding**: Clearer code organization
- **Safer Changes**: Better test coverage
- **Faster Development**: Reusable components and services

## Success Metrics

### Quantitative Goals
- Reduce average file size from ~250 lines to <200 lines
- Achieve >80% test coverage for business logic
- Eliminate all `log.Fatal` and `os.Exit` calls from business logic
- Reduce cyclomatic complexity of individual functions

### Qualitative Goals
- Clear separation between CLI concerns and business logic
- Consistent error handling patterns across all commands
- Easy to add new commands using established patterns
- Improved code readability and maintainability

## Risk Mitigation

### Potential Risks
- **Breaking Changes**: Refactoring might introduce bugs
- **Timeline Impact**: Large scope might delay other features
- **Integration Complexity**: AWS SDK integration challenges

### Mitigation Strategies
- **Comprehensive Testing**: Integration tests for all commands
- **Gradual Rollout**: Feature flags for new behavior
- **Code Review**: Peer review for all changes
- **Backup Plans**: Ability to rollback changes if needed

## Conclusion

This refactoring plan provides a structured approach to significantly improving the `cmd` folder's architecture while maintaining backward compatibility and minimizing risk. The parallel task structure allows for efficient resource allocation and faster completion.

Each detailed task document provides step-by-step implementation instructions, making this plan actionable for development teams.
