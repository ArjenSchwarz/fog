---
references:
    - specs/deploy-output/requirements.md
    - specs/deploy-output/design.md
    - specs/deploy-output/decision_log.md
---
# Deploy Output Feature - Multi-Format Output Support

- [ ] 1. Add go-isatty dependency for TTY detection
  - This enables conditional formatting based on whether stderr is a TTY

## Phase 1: Infrastructure Setup

- [ ] 2. Add --quiet flag to DeployFlags struct

- [ ] 3. Register --quiet flag in deploy command

- [ ] 4. Add new fields to DeployInfo struct for data capture

- [ ] 5. Create createStderrOutput() helper with TTY detection

## Phase 2: Stream Separation

- [ ] 6. Update showEvents() to use stderr with quiet mode support

- [ ] 7. Update showDeploymentInfo() to use stderr with quiet mode support

- [ ] 8. Update printBasicStackInfo() to use stderr

- [ ] 9. Update all progress printMessage() calls to use stderr

- [ ] 10. Update interactive prompts to write to stderr

- [ ] 11. Capture deployment start timestamp

## Phase 3: Data Capture

- [ ] 12. Modify createAndShowChangeset() to capture changeset data

- [ ] 13. Capture deployment end timestamp and final stack state on success

- [ ] 14. Capture error details and stack state on deployment failure

- [ ] 15. Create outputDryRunResult() function

## Phase 4: Final Output Builders

- [ ] 16. Create outputSuccessResult() function

- [ ] 17. Create outputNoChangesResult() function

- [ ] 18. Create outputFailureResult() function

- [ ] 19. Create helper function extractFailedResources()

- [ ] 20. Integrate outputDryRunResult() into deployment flow

## Phase 5: Integration

- [ ] 21. Integrate create-changeset mode output

- [ ] 22. Integrate outputSuccessResult() into deployment flow

- [ ] 23. Integrate outputNoChangesResult() for no-changes scenario

- [ ] 24. Integrate outputFailureResult() for deployment failures

- [ ] 25. Implement quiet mode auto-approval logic

- [ ] 26. Pass quiet flag through all progress output functions

- [ ] 27. Write unit tests for createStderrOutput() TTY detection

## Phase 6: Testing

- [ ] 28. Write unit tests for output builder functions

- [ ] 29. Create golden files for output formats

- [ ] 30. Write golden file tests for all output formats

- [ ] 31. Write integration test for successful deployment with JSON output

- [ ] 32. Write integration test for failed deployment with formatted output

- [ ] 33. Write integration test for quiet mode

- [ ] 34. Write integration test for dry-run with multiple formats

- [ ] 35. Write integration test for no-changes scenario

- [ ] 36. Remove viper.Set("output", "table") override

## Phase 7: Cleanup

- [ ] 37. Verify all output paths use correct streams

- [ ] 38. Run go fmt on all modified files

- [ ] 39. Run go test ./... to verify all tests pass

- [ ] 40. Run golangci-lint to ensure code quality
