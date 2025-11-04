# Requirements: Changeset Output Format Support

## Introduction

The `describe changeset` command currently has hardcoded table output, which prevents users from consuming changeset information in other formats like JSON or CSV for programmatic processing, automation, or analysis. This feature will remove the hardcoded table enforcement and enable users to control the output format through the global `--output` flag, supporting all formats that fog provides: table, csv, json, yaml, markdown, html, and dot.

This enhancement aligns the describe changeset command with fog's other commands that respect the global output format setting, providing a consistent user experience across the CLI.

## Requirements

### 1. Output Format Configuration

**User Story:** As a fog user, I want to control the output format of the describe changeset command using the global `--output` flag, so that I can consume changeset information in the format that best suits my workflow.

**Acceptance Criteria:**

1. <a name="1.1"></a>The system SHALL remove the hardcoded `viper.Set("output", "table")` enforcement from the describe changeset command
2. <a name="1.2"></a>The system SHALL respect the global `--output` flag setting when rendering changeset information
3. <a name="1.3"></a>The system SHALL support these fog output formats: table, csv, json, yaml, markdown, html
4. <a name="1.4"></a>The system SHALL use the output format as determined by the configuration system (CLI flag, config file, environment variable, or default), maintaining backward compatibility

### 2. Content Completeness Across Formats

**User Story:** As a user consuming changeset data programmatically, I want all changeset information included in non-table formats, so that I have access to complete data for analysis and automation.

**Acceptance Criteria:**

1. <a name="2.1"></a>The system SHALL include stack information (stack name, account, region, action type, console URL) in all output formats
2. <a name="2.2"></a>The system SHALL include all changeset changes with full details (action, logical ID, resource type, physical ID, replacement status, module path if present) in all output formats
3. <a name="2.3"></a>The system SHALL include dangerous changes information (resources with Remove action or Conditional/True replacement) in all output formats
4. <a name="2.4"></a>The system SHALL include summary statistics (total changes, added, removed, modified, replacements, conditionals) in all output formats
5. <a name="2.5"></a>The system SHALL include the AWS console changeset URL as a field in the stack information table (not for dry runs)
6. <a name="2.6"></a>The system SHALL structure output to match the current table organization: stack information section, changes section, dangerous changes section, and summary section
7. <a name="2.7"></a>The system SHALL handle empty changesets (zero changes) by including stack information and an appropriate indication of no changes for each format

### 3. Format-Specific Rendering

**User Story:** As a user, I want changeset information to be properly structured for each output format, so that the data is usable and follows format conventions.

**Acceptance Criteria:**

1. <a name="3.1"></a>The system SHALL render all changeset data through a single go-output v2 Builder document to ensure consistent formatting
2. <a name="3.2"></a>The system SHALL structure JSON output using go-output v2's standard format with a `tables` array containing table objects with `title` and `data` fields
3. <a name="3.3"></a>The system SHALL structure CSV output as multiple sections (one per table) with section titles and headers separating each section
4. <a name="3.4"></a>The system SHALL structure YAML output matching JSON structure (tables array with title and data fields)
5. <a name="3.5"></a>The system SHALL preserve color and emoji formatting in table output while stripping ANSI codes from non-terminal formats (JSON, CSV, YAML)
6. <a name="3.6"></a>The system SHALL maintain the existing table format structure with multiple tables (stack info, changes, dangerous changes, summary) when table output is selected
7. <a name="3.7"></a>The system SHALL use empty tables with headers but no data rows to represent missing sections (e.g., no dangerous changes), which will render appropriately per format
8. <a name="3.8"></a>The system SHALL include console URL as a field in the stack information table, making it accessible in all formats (table column, JSON/YAML field, CSV column)

### 4. Scope Limitation

**User Story:** As a developer maintaining fog, I want this feature limited to the describe changeset command, so that we can implement and test changes incrementally without affecting other commands.

**Acceptance Criteria:**

1. <a name="4.1"></a>The system SHALL modify only the `describe changeset` command (cmd/describe_changeset.go)
2. <a name="4.2"></a>The system SHALL NOT modify the deploy command's changeset display functionality
3. <a name="4.3"></a>The system SHALL NOT affect other describe commands or report commands

### 5. Backward Compatibility

**User Story:** As an existing fog user, I want the describe changeset command to maintain its current behavior by default, so that my existing scripts and workflows continue to work without modification.

**Acceptance Criteria:**

1. <a name="5.1"></a>The system SHALL maintain the existing table layout and presentation for table format output
2. <a name="5.2"></a>The system SHALL continue to display all existing information (stack info, changes, dangerous changes, summary, console URL) in table format
3. <a name="5.3"></a>The system SHALL continue to support the `--url` and `--changeset` flags without changes to their behavior
4. <a name="5.4"></a>The system SHALL maintain the same default behavior as other fog commands (respecting configuration precedence order)

### 6. Data Structure Specification

**User Story:** As a developer implementing this feature, I want a clear specification of the output structure for each format, so that the implementation is consistent and predictable.

**Acceptance Criteria:**

1. <a name="6.1"></a>The system SHALL structure JSON and YAML output using go-output v2's standard format with a `tables` array containing table objects, each with:
   - `title`: String containing the table title
   - `data`: Array of row objects
   - Stack information table SHALL include: stackName, account, region, action, and consoleUrl (when not dry run) fields
   - Changes table SHALL include: action, logicalId, resourceType, physicalId, replacement, and optional module fields
   - Dangerous changes table SHALL include: action, logicalId, resourceType, physicalId, replacement, details, and optional module fields
   - Summary table SHALL include: total, added, removed, modified, replacements, conditionals count fields
2. <a name="6.2"></a>The system SHALL structure CSV output as multiple sections (one per table) with section titles and headers, where each section contains its respective data rows with appropriate columns
3. <a name="6.3"></a>The system SHALL represent "no dangerous changes" as an empty table with headers but no data rows, which renders appropriately per format (empty table in table format, empty data array in JSON/YAML, headers only in CSV)
4. <a name="6.4"></a>The system SHALL represent empty changesets with stack information populated and changes/dangerousChanges tables with empty data arrays (JSON/YAML) or no data rows (CSV/table), with summary counts all set to zero

### 7. Error Handling

**User Story:** As a user, I want clear error messages when changeset output fails, so that I can troubleshoot issues effectively.

**Acceptance Criteria:**

1. <a name="7.1"></a>The system SHALL display clear error messages if the output format is unsupported
2. <a name="7.2"></a>The system SHALL display clear error messages if rendering fails for any output format
3. <a name="7.3"></a>The system SHALL continue to display existing error messages for changeset retrieval failures
4. <a name="7.4"></a>The system SHALL maintain the current error handling for invalid changeset names or URLs
