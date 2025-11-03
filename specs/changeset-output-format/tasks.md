---
references:
    - specs/changeset-output-format/requirements.md
    - specs/changeset-output-format/design.md
    - specs/changeset-output-format/decision_log.md
---
# Changeset Output Format Support Implementation

## Phase 1: Refactor Output Functions

- [ ] 1. Create addStackInfoSection() from buildBasicStackInfo()
  - Accept builder, deployment, awsConfig, changeset, showDryRunInfo parameters
  - Return builder after adding stack info table
  - Include ConsoleURL field when not dry run
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [2.5](requirements.md#2.5), [3.8](requirements.md#3.8)
  - References: cmd/describe_changeset.go

- [ ] 2. Create buildChangesetData() helper for data preparation
  - Extract data preparation logic from mixed builder/data code
  - Return changeRows, summaryContent, dangerRows
  - Separate data transformation from rendering logic
  - Requirements: [2.2](requirements.md#2.2), [3.1](requirements.md#3.1)
  - References: cmd/describe_changeset.go

- [ ] 3. Create addChangesetSections() from buildChangesetDocument()
  - Accept builder and changeset parameters
  - Return builder after adding all changeset sections
  - Use empty table for no dangerous changes instead of text message
  - Handle empty changesets appropriately
  - Requirements: [2.2](requirements.md#2.2), [2.3](requirements.md#2.3), [2.4](requirements.md#2.4), [2.6](requirements.md#2.6), [2.7](requirements.md#2.7), [3.1](requirements.md#3.1), [3.7](requirements.md#3.7), [6.3](requirements.md#6.3), [6.4](requirements.md#6.4)
  - References: cmd/describe_changeset.go

- [ ] 4. Update existing tests to use new functions
  - Modify cmd/describe_changeset_test.go to call new functions
  - Ensure tests still pass with new implementation
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2)
  - References: cmd/describe_changeset_test.go

## Phase 2: Update Command Function

- [ ] 5. Remove viper.Set("output", "table") from describeChangeset()
  - Delete line 56 in cmd/describe_changeset.go
  - Allows respecting global output flag
  - Core change enabling multi-format support
  - Requirements: [1.1](requirements.md#1.1), [1.2](requirements.md#1.2), [1.3](requirements.md#1.3), [1.4](requirements.md#1.4)
  - References: cmd/describe_changeset.go

- [ ] 6. Create buildAndRenderChangeset() function
  - Create orchestration function
  - Build complete document with addStackInfoSection() and addChangesetSections()
  - Render and handle errors with clear messages
  - Exit with code 1 on failure
  - Requirements: [3.1](requirements.md#3.1), [7.1](requirements.md#7.1), [7.2](requirements.md#7.2)
  - References: cmd/describe_changeset.go

- [ ] 7. Update describeChangeset() to call new function
  - Replace lines 82-83 with call to buildAndRenderChangeset()
  - Remove the builder variable
  - Requirements: [4.1](requirements.md#4.1)
  - References: cmd/describe_changeset.go

- [ ] 8. Test with table format (verify backward compatibility)
  - Run existing tests
  - Manually verify table output matches current behavior exactly
  - Check spacing, newlines, and formatting
  - Requirements: [5.1](requirements.md#5.1), [5.2](requirements.md#5.2), [5.3](requirements.md#5.3), [5.4](requirements.md#5.4)
  - References: cmd/describe_changeset_test.go

## Phase 3: Testing

- [ ] 9. Add unit tests for all output formats
  - Create table-driven tests
  - Cover table, csv, json, yaml, markdown, html formats
  - Use sample changeset data
  - Verify each format renders without error
  - Requirements: [1.3](requirements.md#1.3), [3.1](requirements.md#3.1), [3.2](requirements.md#3.2), [3.3](requirements.md#3.3), [3.4](requirements.md#3.4)
  - References: cmd/describe_changeset_test.go

- [ ] 10. Add tests for empty changesets
  - Test each format with zero changes
  - Verify text for table/markdown/html
  - Verify empty arrays for JSON/YAML
  - Verify no data rows for CSV
  - Requirements: [2.7](requirements.md#2.7), [6.4](requirements.md#6.4)
  - References: cmd/describe_changeset_test.go

- [ ] 11. Add tests for console URL inclusion
  - Verify URL appears as field in stack info table
  - Test all formats: table column, JSON field, CSV column
  - Verify URL is omitted for dry runs
  - Requirements: [2.5](requirements.md#2.5), [3.8](requirements.md#3.8)
  - References: cmd/describe_changeset_test.go

- [ ] 12. Add ANSI code stripping tests
  - Verify JSON/YAML/CSV contain no ANSI escape sequences
  - Verify table output DOES contain ANSI codes
  - Test bold formatting on Remove actions
  - Requirements: [3.5](requirements.md#3.5)
  - References: cmd/describe_changeset_test.go

- [ ] 13. Add JSON structure tests
  - Test JSON has tables array as top-level key
  - Verify each table has title and data fields
  - Test console URL accessible via tables[0].data[0].ConsoleURL
  - Verify data structure matches specification
  - Requirements: [3.2](requirements.md#3.2), [6.1](requirements.md#6.1)
  - References: cmd/describe_changeset_test.go

- [ ] 14. Add integration tests
  - Create integration tests with mocked AWS clients
  - Test full command execution
  - Test with different formats via --output flag
  - Requirements: [1.3](requirements.md#1.3), [1.4](requirements.md#1.4)
  - References: cmd/describe_changeset_integration_test.go

- [ ] 15. Verify golden file tests if they exist
  - Check for golden file tests
  - Update golden files if format changes are expected
  - Requirements: [5.1](requirements.md#5.1)

## Phase 4: Cleanup

- [ ] 16. Remove ONLY printChangeset() function
  - Delete printChangeset() from cmd/describe_changeset.go
  - DO NOT remove showChangeset(), buildChangesetDocument(), or printBasicStackInfo()
  - These are used by deploy and history commands
  - Requirements: [4.1](requirements.md#4.1), [4.2](requirements.md#4.2), [4.3](requirements.md#4.3)
  - References: cmd/describe_changeset.go

- [ ] 17. Remove unused imports
  - Clean up imports no longer needed
  - Run go fmt to organize imports
  - References: cmd/describe_changeset.go

- [ ] 18. Run go fmt on modified files
  - Format cmd/describe_changeset.go
  - Format cmd/describe_changeset_test.go
  - Format cmd/describe_changeset_integration_test.go if it exists

- [ ] 19. Run golangci-lint run
  - Verify no linting issues
  - Address any warnings or errors found

- [ ] 20. Update CHANGELOG.md
  - Add entry for changeset output format support feature
  - Document that describe changeset now respects --output flag
  - Document support for all fog formats
  - References: CHANGELOG.md
