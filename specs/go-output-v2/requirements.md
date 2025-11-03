# Requirements: Go-Output v2 Migration

## Introduction

This feature migrates fog from go-output v1 to v2.2.1+, adopting modern patterns and improved architecture. The migration will replace the global state-based OutputSettings approach with v2's Builder pattern and functional options, while leveraging v2.2.1's newly added features that directly replace all v1 functionality used by fog.

**Key Goals:**
- Achieve complete feature parity with v1 (all output formats, styling, configuration)
- Adopt v2's superior patterns (Builder, functional options, thread safety)
- Leverage v2.2.1+ features (inline styling, column width, array handling)
- Maintain backward compatibility for user-facing CLI behavior
- Improve code maintainability and reduce global state

**Scope:**
- All commands using go-output (deploy, drift, report, exports, dependencies, history, etc.)
- Configuration layer (config/config.go)
- Output formatting across all formats (table, csv, json, dot)

## Requirements

### 1. Dependency Update

**User Story:** As a developer, I want to update the go-output dependency to v2.2.1+, so that I can use the latest features and improvements.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL update go.mod to use go-output v2.2.1 or later
2. <a name="1.2"></a>The system SHALL remove the v1 dependency after migration is complete
3. <a name="1.3"></a>The system SHALL verify no dependency conflicts exist with v2.2.1
4. <a name="1.4"></a>The system SHALL update import paths from `github.com/ArjenSchwarz/go-output` to `github.com/ArjenSchwarz/go-output/v2`

### 2. Inline Styling Migration

**User Story:** As a developer, I want to replace v1's inline styling methods with v2.2.1's stateless styling functions, so that I can color text within table cells without global state.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL replace all `outputsettings.StringWarningInline()` calls with `output.StyleWarning()`
2. <a name="2.2"></a>The system SHALL replace all `outputsettings.StringPositiveInline()` calls with `output.StylePositive()`
3. <a name="2.3"></a>The system SHALL replace all `outputsettings.StringNegativeInline()` calls with `output.StyleNegative()`
4. <a name="2.4"></a>The system SHALL preserve the visual appearance of styled output (colors, formatting)
5. <a name="2.5"></a>The system SHALL use conditional styling functions (e.g., `StyleWarningIf()`) where appropriate to improve code readability
6. <a name="2.6"></a>The system SHALL ensure styled text renders correctly in all supported output formats

### 3. Table Column Width Configuration

**User Story:** As a user, I want to configure maximum table column width, so that output remains readable in standard terminal windows.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL support table column width configuration via `table.max-column-width` setting
2. <a name="3.2"></a>The system SHALL use v2's `TableWithMaxColumnWidth()` constructor to apply width limits
3. <a name="3.3"></a>The system SHALL use `TableWithStyleAndMaxColumnWidth()` when both style and width are configured
4. <a name="3.4"></a>The system SHALL automatically wrap text within cells when content exceeds the configured width
5. <a name="3.5"></a>The system SHALL default to 50 characters maximum width if not configured
6. <a name="3.6"></a>The config.NewOutputSettings() method SHALL return appropriate v2 Format objects based on configuration

### 4. Output Format Migration

**User Story:** As a developer, I want to migrate from OutputArray to v2's Builder pattern, so that I can construct documents with better type safety and no global state.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL replace all OutputArray instantiations with v2's Builder pattern
2. <a name="4.2"></a>The system SHALL use `output.New().Table().Build()` pattern for single tables
3. <a name="4.3"></a>The system SHALL chain multiple `.Table()` calls for documents with multiple tables
4. <a name="4.4"></a>The system SHALL use `output.WithKeys()` to specify exact column ordering
5. <a name="4.5"></a>The system SHALL preserve existing key ordering from v1 implementation
6. <a name="4.6"></a>The system SHALL use `Header()` method for document titles instead of Settings.Title
7. <a name="4.7"></a>The system SHALL pass context.Background() to all Render() calls

### 5. Output Settings to Functional Options Migration

**User Story:** As a developer, I want to replace OutputSettings with v2's functional options pattern, so that I can configure output behavior in a more composable way.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL replace OutputSettings configuration with functional options
2. <a name="5.2"></a>The system SHALL use `WithFormat()` to specify output format (table, csv, json, dot)
3. <a name="5.3"></a>The system SHALL use `WithWriter(NewStdoutWriter())` for console output
4. <a name="5.4"></a>The system SHALL use `WithWriter(NewFileWriter())` for file output when configured
5. <a name="5.5"></a>The system SHALL support multiple formats and writers in a single output configuration
6. <a name="5.6"></a>The system SHALL use `WithTransformer(&EmojiTransformer{})` when emoji output is enabled
7. <a name="5.7"></a>The system SHALL use `WithTransformer(&ColorTransformer{})` when color output is enabled
8. <a name="5.8"></a>The config package SHALL provide helper methods to construct v2 Output configurations from viper settings

### 6. Array Handling Optimization

**User Story:** As a developer, I want to leverage v2's automatic array handling, so that multi-value cells render appropriately in each output format without manual separator logic.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL identify locations using `GetSeparator()` with `strings.Join()` for array values
2. <a name="6.2"></a>The system SHALL replace joined strings with direct array assignment where appropriate
3. <a name="6.3"></a>The system SHALL allow v2 to handle format-specific array rendering (newlines for table, semicolons for CSV, etc.)
4. <a name="6.4"></a>The system SHALL maintain the existing GetSeparator() method for cases where manual joining is preferred
5. <a name="6.5"></a>The system SHALL verify array rendering produces equivalent or better output across all formats
6. <a name="6.6"></a>The drift detection output SHALL correctly display multi-line property differences

### 7. File Output Configuration

**User Story:** As a user, I want to save output to files in different formats, so that I can process results with other tools.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL support the `--file` flag to specify output file path
2. <a name="7.2"></a>The system SHALL support the `--file-format` flag to specify file output format
3. <a name="7.3"></a>The system SHALL use v2's `NewFileWriter(dir, pattern)` for file output
4. <a name="7.4"></a>The system SHALL support simultaneous console and file output
5. <a name="7.5"></a>The system SHALL support different formats for console vs file (e.g., table to console, json to file)
6. <a name="7.6"></a>The system SHALL create parent directories if they don't exist when writing files

### 8. Table Sorting

**User Story:** As a developer, I want to sort table output by specific columns, so that results are presented in a logical order.

**Acceptance Criteria:**

1. <a name="8.1"></a>The system SHALL support the SortKey configuration for table sorting
2. <a name="8.2"></a>The system SHALL use v2's data pipeline `.SortBy()` method instead of byte transformers
3. <a name="8.3"></a>The system SHALL default to ascending sort order
4. <a name="8.4"></a>The system SHALL apply sorting before rendering to avoid parse/render cycles
5. <a name="8.5"></a>The system SHALL preserve sort functionality in all commands that currently use it

### 9. Multiple Table Support

**User Story:** As a developer, I want to output multiple tables with different column sets in a single command, so that I can present related but differently structured data.

**Acceptance Criteria:**

1. <a name="9.1"></a>The system SHALL support multiple tables in a single document
2. <a name="9.2"></a>The system SHALL allow each table to have independent column ordering
3. <a name="9.3"></a>The system SHALL preserve the ability to add tables incrementally (e.g., in loops)
4. <a name="9.4"></a>The system SHALL render multiple tables correctly in all output formats
5. <a name="9.5"></a>The system SHALL maintain table separation in table format output

### 10. Drift Detection Output

**User Story:** As a user, I want to see drift detection results with clear visual indicators, so that I can quickly identify infrastructure changes.

**Acceptance Criteria:**

1. <a name="10.1"></a>The drift command SHALL use v2's inline styling for change type indicators
2. <a name="10.2"></a>The drift command SHALL highlight DELETED resources with warning styling
3. <a name="10.3"></a>The drift command SHALL highlight property differences with appropriate styling
4. <a name="10.4"></a>The drift command SHALL display multi-line property values correctly
5. <a name="10.5"></a>The drift command SHALL handle NACL, Route Table, and Transit Gateway route differences
6. <a name="10.6"></a>The drift output SHALL be readable in both table and CSV formats

### 11. Configuration Layer Update

**User Story:** As a developer, I want the config package to provide v2-compatible helper methods, so that command code can easily construct v2 output objects from viper settings.

**Acceptance Criteria:**

1. <a name="11.1"></a>The config package SHALL provide a method to construct v2 Format objects from settings
2. <a name="11.2"></a>The config package SHALL handle table style configuration (Default, Bold, ColoredBright, etc.)
3. <a name="11.3"></a>The config package SHALL handle table max column width configuration
4. <a name="11.4"></a>The config package SHALL provide a method to construct v2 Output objects with appropriate writers
5. <a name="11.5"></a>The config package SHALL handle emoji and color transformer configuration
6. <a name="11.6"></a>The config package SHALL maintain backward compatibility with existing viper key names

### 12. Testing and Validation

**User Story:** As a developer, I want comprehensive testing of the migration, so that I can be confident the v2 implementation matches v1 behavior.

**Acceptance Criteria:**

1. <a name="12.1"></a>The system SHALL pass all existing unit tests after migration
2. <a name="12.2"></a>The system SHALL pass all existing integration tests after migration
3. <a name="12.3"></a>The system SHALL validate output format correctness for table, csv, json, and dot formats
4. <a name="12.4"></a>The system SHALL validate inline styling appears correctly in terminal output
5. <a name="12.5"></a>The system SHALL validate table column width limiting works as expected
6. <a name="12.6"></a>The system SHALL validate file output writes correctly to specified paths
7. <a name="12.7"></a>The system SHALL test on Windows, macOS, and Linux platforms

### 13. Backward Compatibility

**User Story:** As a user, I want the CLI interface and output to remain unchanged, so that my existing scripts and workflows continue to work.

**Acceptance Criteria:**

1. <a name="13.1"></a>The system SHALL maintain all existing CLI flags and their behavior
2. <a name="13.2"></a>The system SHALL maintain all existing configuration file keys
3. <a name="13.3"></a>The system SHALL produce functionally equivalent output to v1 for all commands
4. <a name="13.4"></a>The system SHALL maintain the same default values for all settings
5. <a name="13.5"></a>The system SHALL maintain support for all currently supported output formats
6. <a name="13.6"></a>The system SHALL not introduce breaking changes to user-facing behavior

### 14. Code Quality and Maintainability

**User Story:** As a developer, I want clean, maintainable code following v2 best practices, so that future modifications are easier.

**Acceptance Criteria:**

1. <a name="14.1"></a>The system SHALL eliminate global state from output configuration
2. <a name="14.2"></a>The system SHALL use v2's Builder pattern consistently across all commands
3. <a name="14.3"></a>The system SHALL use functional options for configuration instead of settings objects
4. <a name="14.4"></a>The system SHALL follow Go best practices for error handling with context
5. <a name="14.5"></a>The system SHALL pass linting checks (golangci-lint)
6. <a name="14.6"></a>The system SHALL format all code with `go fmt`
7. <a name="14.7"></a>The code SHALL be self-documenting with clear variable names and minimal comments

### 15. Documentation

**User Story:** As a developer or user, I want updated documentation, so that I understand the changes and can reference the new implementation.

**Acceptance Criteria:**

1. <a name="15.1"></a>The CHANGELOG SHALL document the migration to go-output v2.2.1
2. <a name="15.2"></a>The CHANGELOG SHALL note any user-visible changes (even if minimal)
3. <a name="15.3"></a>The README SHALL be updated if there are new dependencies or requirements
4. <a name="15.4"></a>Code comments SHALL be added where v2 patterns differ significantly from v1
5. <a name="15.5"></a>The migration SHALL be documented in the decision log

### 16. Error Handling Improvements

**User Story:** As a user, I want clear error messages when file output fails, so that I can understand and fix configuration issues.

**Acceptance Criteria:**

1. <a name="16.1"></a>The system SHALL log a warning when NewFileWriter() fails to create a file writer
2. <a name="16.2"></a>The system SHALL continue with console output even if file writer creation fails
3. <a name="16.3"></a>The error message SHALL indicate which file path failed and why
4. <a name="16.4"></a>The system SHALL NOT silently swallow file writer creation errors

### 17. Code Comments Cleanup

**User Story:** As a developer, I want accurate code comments that reflect current Go behavior, so that the codebase is maintainable.

**Acceptance Criteria:**

1. <a name="17.1"></a>The system SHALL remove or update obsolete loop variable capture comments
2. <a name="17.2"></a>Comments about Go 1.22+ automatic loop variable capture SHALL be removed unless explaining why old pattern is preserved
3. <a name="17.3"></a>Test files SHALL NOT contain misleading comments about manual variable capture

### 18. Report Command Frontmatter Support

**User Story:** As a user, I want frontmatter in markdown reports when requested, so that I can integrate reports with static site generators.

**Acceptance Criteria:**

1. <a name="18.1"></a>The report command SHALL support --frontmatter flag for markdown output
2. <a name="18.2"></a>The system SHALL generate frontmatter metadata including stack name, region, account
3. <a name="18.3"></a>The frontmatter SHALL be properly attached to the v2 output rendering
4. <a name="18.4"></a>The frontmatter SHALL appear as a YAML block at the beginning of markdown output

### 19. Report Command Mermaid Timeline Support

**User Story:** As a user, I want Mermaid Gantt charts in markdown/HTML reports, so that I can visualize deployment timelines.

**Acceptance Criteria:**

1. <a name="19.1"></a>The report command SHALL render Mermaid timelines for markdown and HTML output formats
2. <a name="19.2"></a>The system SHALL use Mermaid code blocks with ganttChart syntax, not plain tables
3. <a name="19.3"></a>The Mermaid chart SHALL include all resource events with start times and durations
4. <a name="19.4"></a>The chart SHALL be properly formatted within markdown/HTML output
