---
references:
    - specs/go-output-v2/requirements.md
    - specs/go-output-v2/design.md
    - specs/go-output-v2/decision_log.md
---
# Go-Output v2 Migration - Implementation Tasks

## Phase 0: Pre-Migration Baseline

- [x] 1. Create golden file baseline for current v1 output
  - Before making any code changes, capture v1 output as golden files
  - Create testdata directory structure for golden files
  - Run exports command with sample data and save table format output
  - Run exports command with sample data and save CSV format output
  - Run exports command with sample data and save JSON format output
  - Run exports command with sample data and save dot format output
  - These golden files serve as the comparison baseline for v2 migration
  - Add update flag support to regenerate golden files when needed
  - Requirements: [12.3](requirements.md#12.3), [13.3](requirements.md#13.3)
  - References: cmd/exports_test.go, testdata/

## Phase 1: Dependency Update

- [x] 2. Update dependencies and import paths
  - Update go.mod to require github.com/ArjenSchwarz/go-output/v2 v2.2.1 or later
  - Update all import statements from github.com/ArjenSchwarz/go-output to github.com/ArjenSchwarz/go-output/v2
  - Run go mod tidy to resolve dependencies
  - Verify no dependency conflicts exist
  - Requirements: [1.1](requirements.md#1.1), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4)
  - References: go.mod

## Phase 2: Configuration Layer

- [x] 3. Write unit tests for config layer Format helper methods
  - Create test cases for GetTableFormat() method
  - Test table style name mapping (Default, Bold, ColoredBright, etc.)
  - Verify TableWithStyleAndMaxColumnWidth() is called with correct parameters
  - Test that max column width is read from viper configuration
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [11.2](requirements.md#11.2), [11.3](requirements.md#11.3), [12.1](requirements.md#12.1)
  - References: config/config_test.go

- [x] 4. Implement config layer helper methods for v2 Format objects
  - Add GetTableFormat() method to config.Config that creates v2 Format objects with style and max column width
  - Implement table style name mapping (Default, Bold, ColoredBright, etc.)
  - Use TableWithStyleAndMaxColumnWidth() constructor with values from viper
  - Keep existing NewOutputSettings() method temporarily for backward compatibility during migration
  - Requirements: [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.6](requirements.md#3.6), [11.1](requirements.md#11.1), [11.2](requirements.md#11.2), [11.3](requirements.md#11.3)
  - References: config/config.go

- [x] 5. Write unit tests for config layer Output options helper methods
  - Create test cases for GetOutputOptions() method
  - Test format name mapping (table, csv, json, dot) to v2 Format objects
  - Test console output configuration with WithFormat() and WithWriter()
  - Test file output configuration when --file flag is set
  - Test transformer addition when emoji/color settings are enabled
  - Verify correct number and types of options are returned
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [7.3](requirements.md#7.3), [11.4](requirements.md#11.4), [11.5](requirements.md#11.5), [12.1](requirements.md#12.1)
  - References: config/config_test.go

- [x] 6. Implement config layer helper methods for v2 Output options
  - Add GetOutputOptions() method that returns []output.Option based on viper settings
  - Add getFormatForOutput() helper that maps format names (table, csv, json, dot) to v2 Format objects
  - Handle console output with WithFormat() and WithWriter(NewStdoutWriter())
  - Handle file output with WithWriter(NewFileWriter()) when --file flag is configured
  - Add emoji transformer when use-emoji setting is enabled
  - Add color transformer when use-colors setting is enabled
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4), [5.5](requirements.md#5.5), [5.6](requirements.md#5.6), [5.7](requirements.md#5.7), [7.3](requirements.md#7.3), [11.4](requirements.md#11.4), [11.5](requirements.md#11.5)
  - References: config/config.go

## Phase 3: Inline Styling Migration

- [x] 7. Migrate inline styling calls to v2 functions
  - Replace all outputsettings.StringWarningInline() calls with output.StyleWarning()
  - Replace all outputsettings.StringPositiveInline() calls with output.StylePositive()
  - Replace all outputsettings.StringNegativeInline() calls with output.StyleNegative()
  - Use StyleWarningIf(), StylePositiveIf() for conditional styling where appropriate
  - Update imports to include v2 output package for styling functions
  - Remove outputsettings variable references from styling calls
  - Requirements: [2.1](requirements.md#2.1), [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.5](requirements.md#2.5), [2.6](requirements.md#2.6)
  - References: cmd/drift.go, cmd/deploy.go, cmd/report.go

## Phase 4: Command Migration - Exports

- [x] 8. Write unit tests for exports command v2 migration
  - Create test cases validating Builder pattern usage
  - Test WithKeys() preserves column order (Export, Value)
  - Test array handling for ImportingStacks field
  - Verify output renders correctly with settings.GetOutputOptions()
  - Requirements: [4.5](requirements.md#4.5), [6.5](requirements.md#6.5), [12.1](requirements.md#12.1)
  - References: cmd/exports_test.go

- [x] 9. Migrate exports command to v2 Builder pattern
  - Replace OutputArray instantiation with output.New().Table().Build()
  - Use settings.GetTableFormat() for table configuration
  - Use WithKeys() to specify column order: Export, Value
  - Pass ImportingStacks as array directly (let v2 handle separator logic)
  - Use settings.GetOutputOptions() to create Output instance
  - Call Render(context.Background(), doc.Build())
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [4.6](requirements.md#4.6), [4.7](requirements.md#4.7), [6.2](requirements.md#6.2), [6.3](requirements.md#6.3), [13.3](requirements.md#13.3)
  - References: cmd/exports.go

## Phase 5: Command Migration - Dependencies

- [x] 10. Write unit tests for dependencies command v2 migration
  - Create test cases for Builder pattern with sorting
  - Test SortBy() data pipeline method
  - Test column ordering (Name, DependedOnBy)
  - Verify sorted output matches v1 behavior
  - Requirements: [8.5](requirements.md#8.5), [12.1](requirements.md#12.1)
  - References: cmd/dependencies_test.go

- [x] 11. Migrate dependencies command with sorting
  - Replace OutputArray with Builder pattern
  - Use doc.Pipeline().SortBy(sortKey, output.Ascending).Execute() for sorting
  - Use WithKeys() for column order
  - Remove Settings.SortKey usage (replaced by data pipeline)
  - Render with settings.GetOutputOptions()
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [8.1](requirements.md#8.1), [8.2](requirements.md#8.2), [8.3](requirements.md#8.3), [8.4](requirements.md#8.4)
  - References: cmd/dependencies.go

## Phase 6: Command Migration - Drift

- [ ] 12. Write unit tests for drift command v2 migration
  - Create test cases for multiple tables with different column sets
  - Test inline styling in drift output (StyleWarning, StylePositive)
  - Test array handling for property differences
  - Test NACL, Route Table, Transit Gateway route difference handling
  - Verify multi-line property values render correctly
  - Requirements: [9.2](requirements.md#9.2), [10.1](requirements.md#10.1), [10.2](requirements.md#10.2), [10.3](requirements.md#10.3), [10.4](requirements.md#10.4), [10.5](requirements.md#10.5), [12.1](requirements.md#12.1)
  - References: cmd/drift_test.go

- [ ] 13. Migrate drift command to v2 Builder pattern
  - Replace OutputArray with output.New().Table() for main drift table
  - Chain additional .Table() calls for multiple tables (if needed)
  - Use WithKeys() to specify independent column ordering for each table
  - Ensure inline styling (StyleWarning, StylePositive) is applied correctly
  - Handle array values for property differences (let v2 format appropriately)
  - Render all tables with settings.GetOutputOptions()
  - Requirements: [4.1](requirements.md#4.1), [4.3](requirements.md#4.3), [9.1](requirements.md#9.1), [9.2](requirements.md#9.2), [9.4](requirements.md#9.4), [10.6](requirements.md#10.6)
  - References: cmd/drift.go

## Phase 7: Command Migration - Deploy

- [ ] 14. Write unit tests for deploy command v2 migration
  - Create test cases for multiple tables (events + outputs)
  - Test each table has independent column sets
  - Test tables are added incrementally in loops
  - Verify table separation in output
  - Requirements: [9.1](requirements.md#9.1), [9.3](requirements.md#9.3), [9.5](requirements.md#9.5), [12.1](requirements.md#12.1)
  - References: cmd/deploy_test.go

- [ ] 15. Migrate deploy command with multiple tables
  - Use Builder pattern with multiple .Table() calls
  - Add events table with WithKeys() for event columns
  - Add outputs table with WithKeys() for output columns
  - Ensure tables can be added incrementally in loops
  - Use Header() method for document title instead of Settings.Title
  - Render with settings.GetOutputOptions()
  - Requirements: [4.1](requirements.md#4.1), [4.3](requirements.md#4.3), [4.6](requirements.md#4.6), [9.1](requirements.md#9.1), [9.3](requirements.md#9.3)
  - References: cmd/deploy.go

## Phase 8: Remaining Commands

- [ ] 16. Write unit tests for resources command
  - Create test cases for resources command Builder pattern usage
  - Test column ordering matches v1
  - Test array handling for multi-value fields
  - Verify output renders correctly
  - Requirements: [4.5](requirements.md#4.5), [12.1](requirements.md#12.1)
  - References: cmd/resources_test.go

- [ ] 17. Migrate resources command
  - Replace OutputArray with output.New().Table().Build()
  - Use settings.GetTableFormat() and settings.GetOutputOptions()
  - Use WithKeys() for column ordering
  - Handle any array fields appropriately
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [4.7](requirements.md#4.7)
  - References: cmd/resources.go

- [ ] 18. Write unit tests for report command
  - Create test cases for report command Builder pattern usage
  - Test column ordering matches v1
  - Test inline styling if used in report output
  - Verify output renders correctly
  - Requirements: [4.5](requirements.md#4.5), [12.1](requirements.md#12.1)
  - References: cmd/report_test.go

- [ ] 19. Migrate report command
  - Replace OutputArray with output.New().Table().Build()
  - Use settings.GetTableFormat() and settings.GetOutputOptions()
  - Use WithKeys() for column ordering
  - Ensure inline styling uses output.StyleWarning() etc.
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [4.7](requirements.md#4.7), [2.6](requirements.md#2.6)
  - References: cmd/report.go

- [ ] 20. Write unit tests for history command
  - Create test cases for history command Builder pattern usage
  - Test column ordering matches v1
  - Verify output renders correctly
  - Requirements: [4.5](requirements.md#4.5), [12.1](requirements.md#12.1)
  - References: cmd/history_test.go

- [ ] 21. Migrate history command
  - Replace OutputArray with output.New().Table().Build()
  - Use settings.GetTableFormat() and settings.GetOutputOptions()
  - Use WithKeys() for column ordering
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [4.7](requirements.md#4.7)
  - References: cmd/history.go

- [ ] 22. Write unit tests for describe_changeset command
  - Create test cases for describe_changeset command Builder pattern usage
  - Test column ordering matches v1
  - Verify output renders correctly
  - Requirements: [4.5](requirements.md#4.5), [12.1](requirements.md#12.1)
  - References: cmd/describe_changeset_test.go

- [ ] 23. Migrate describe_changeset command
  - Replace OutputArray with output.New().Table().Build()
  - Use settings.GetTableFormat() and settings.GetOutputOptions()
  - Use WithKeys() for column ordering
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.4](requirements.md#4.4), [4.7](requirements.md#4.7)
  - References: cmd/describe_changeset.go

- [ ] 24. Write unit tests for tables helper
  - Create test cases for tables.go helper functions
  - Test any OutputArray usage in helper functions
  - Verify helpers work with v2 Builder pattern
  - Requirements: [12.1](requirements.md#12.1)
  - References: cmd/tables_test.go

- [ ] 25. Migrate tables helper
  - Update any OutputArray usage in tables.go helper functions
  - Ensure helpers accept and return v2 compatible types
  - Update function signatures if needed for v2 compatibility
  - Requirements: [4.1](requirements.md#4.1)
  - References: cmd/tables.go

- [ ] 26. Write unit tests for deploy_helpers
  - Create test cases for deploy_helpers.go functions
  - Test any OutputArray usage in helper functions
  - Verify helpers work with v2 Builder pattern
  - Requirements: [12.1](requirements.md#12.1)
  - References: cmd/deploy_helpers_test.go

- [ ] 27. Migrate deploy_helpers
  - Update any OutputArray usage in deploy_helpers.go
  - Ensure helpers accept and return v2 compatible types
  - Update function signatures if needed for v2 compatibility
  - Requirements: [4.1](requirements.md#4.1)
  - References: cmd/deploy_helpers.go

## Phase 9: Global State Cleanup

- [ ] 28. Remove global outputsettings variable
  - Remove var outputsettings *format.OutputSettings from cmd/root.go
  - Remove initialization of outputsettings in root command
  - Verify no remaining references to global outputsettings exist
  - Search codebase for any lingering outputsettings usage
  - Remove NewOutputSettings() method from config package (if no longer needed)
  - Requirements: [1.2](requirements.md#1.2), [14.1](requirements.md#14.1)
  - References: cmd/root.go, config/config.go

## Phase 10: Testing and Validation

- [ ] 29. Run existing test suite after migration
  - Run go test ./... and verify all existing unit tests pass
  - Run INTEGRATION=1 go test ./... and verify integration tests pass
  - Fix any test failures related to output format changes
  - Ensure tests validate data correctness, not byte-for-byte output matching
  - Requirements: [12.1](requirements.md#12.1), [12.2](requirements.md#12.2)
  - References: All test files

- [ ] 30. Manual validation testing and golden file comparison
  - Compare v2 output against golden files created in Phase 0
  - Verify column ordering matches v1 for each command
  - Test inline styling appears correctly (fog drift command)
  - Test column width limiting with long content
  - Test file output with --file and --file-format flags
  - Test array handling renders with correct separators per format
  - Test multiple tables render correctly
  - Test sorting functionality
  - Document any functional differences from v1 in decision log
  - Update golden files if output is functionally equivalent but differs in acceptable ways
  - Requirements: [12.3](requirements.md#12.3), [12.4](requirements.md#12.4), [12.5](requirements.md#12.5), [12.6](requirements.md#12.6), [13.1](requirements.md#13.1), [13.2](requirements.md#13.2), [13.3](requirements.md#13.3), [13.4](requirements.md#13.4)
  - References: specs/go-output-v2/decision_log.md, testdata/

- [ ] 31. Verify Windows build with cross-compilation
  - Run GOOS=windows GOARCH=amd64 go build to verify Windows compilation succeeds
  - Successful build is sufficient - no Windows machine available for execution testing
  - This verifies the v2 dependency resolves the v1 Windows compilation issue
  - Requirements: [12.7](requirements.md#12.7)
  - References: go.mod

## Phase 11: Code Quality and Cleanup

- [ ] 32. Code quality checks
  - Run go fmt ./... on all modified files
  - Run golangci-lint run and fix any issues
  - Review code for v2 best practices (Builder pattern, functional options)
  - Ensure no global state remains in output configuration
  - Verify error handling uses proper context and wrapping
  - Requirements: [14.2](requirements.md#14.2), [14.3](requirements.md#14.3), [14.4](requirements.md#14.4), [14.5](requirements.md#14.5), [14.6](requirements.md#14.6), [14.7](requirements.md#14.7)
  - References: All modified Go files

## Phase 12: Final Cleanup

- [ ] 33. Remove v1 dependency
  - Remove v1 dependency from go.mod
  - Run go mod tidy to clean up module dependencies
  - Verify build succeeds without v1 dependency
  - Search for any remaining v1 import paths
  - Requirements: [1.2](requirements.md#1.2)
  - References: go.mod

## Phase 13: Documentation

- [ ] 34. Update README if needed
  - Check if README mentions go-output version
  - Update dependency versions if documented
  - Verify build instructions still accurate
  - Update any go-output-specific documentation
  - Requirements: [15.3](requirements.md#15.3)
  - References: README.md

- [ ] 35. Add migration documentation to decision log
  - Document the migration completion in decision_log.md
  - Note any decisions made during implementation
  - Record any deviations from original design
  - Document lessons learned for future migrations
  - Requirements: [15.5](requirements.md#15.5)
  - References: specs/go-output-v2/decision_log.md
