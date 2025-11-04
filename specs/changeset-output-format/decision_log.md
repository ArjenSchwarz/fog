# Decision Log: Changeset Output Format Support

## Overview
This document tracks all decisions made during the requirements, design, and implementation phases of the changeset output format feature.

---

## Requirements Phase Decisions

### Decision 1: Use Global --output Flag (2025-11-03)

**Context**: Need to decide between using global --output flag or creating a dedicated changeset-specific flag.

**Decision**: Use the global --output flag and remove the hardcoded table enforcement.

**Rationale**:
- Provides consistent UX across all fog commands
- Users expect global flags to work everywhere
- Simpler implementation and maintenance

**Status**: Approved by user

---

### Decision 2: Support All Fog Output Formats (2025-11-03)

**Context**: Determine which output formats should be supported.

**Decision**: Support all formats that fog provides: table, csv, json, yaml, markdown, html.

**Rationale**:
- Users mentioned markdown was missing from documentation but is supported
- Consistency with other fog commands
- Flexibility for different use cases

**Status**: Approved by user

**Critical Review Findings**: The `dot` format was initially included but both review agents (design-critic and peer-review-validator) identified that DOT format (GraphViz) is for graph relationships, not tabular data. A changeset is a flat list of changes with no inherent graph structure.

**Final Decision**: Remove `dot` from supported formats for this feature. Graph visualization would require parsing CloudFormation template dependencies - that would be a separate feature if needed.

**Status**: Approved by user - requirements updated to remove dot format

---

### Decision 3: Scope Limited to Describe Changeset Command (2025-11-03)

**Context**: Whether to modify both describe changeset and deploy commands.

**Decision**: Only modify describe changeset command. Deploy command changes will be addressed in a separate spec.

**Rationale**:
- Deploy command involves much more than just changeset output
- Incremental approach reduces risk
- Allows focused testing and validation

**Status**: Approved by user

---

### Decision 4: Include All Information in All Formats (2025-11-03)

**Context**: Determine content scope for non-table formats.

**Decision**: Include all information (stack info, changes, dangerous changes, summary, console URL) in all output formats.

**Rationale**:
- Users consuming data programmatically need complete context
- Omitting information would require multiple command calls
- Maintains feature parity across formats

**Status**: Approved by user

**Critical Review Findings**: Both review agents identified that the exact structure for this "all information" is not defined.

**Final Decision**: Structure all formats to match the current table organization with these sections:
- Stack information (stackName, account, region, action)
- Changes array (all resource changes)
- Dangerous changes (subset of changes that are Remove/Conditional/True replacement)
- Summary statistics (counts)
- Console URL

For JSON/YAML: Use nested objects and arrays matching this structure.
For CSV: One row per change with stack info columns, plus summary rows at the end.

Empty changesets should show appropriate behavior for each format (text messages for table/markdown/html, empty arrays for JSON/YAML, no data rows for CSV).

**Status**: Approved by user - requirements updated with new section 6 "Data Structure Specification"

---

## Critical Review Findings Summary (2025-11-03)

### Finding 1: Missing Data Schema Definition ⚠️ CRITICAL

**Issue**: Requirements specify "properly nested objects" for JSON and "flattened data rows" for CSV but don't define the actual structure.

**Impact**: Different implementations could produce incompatible outputs, defeating programmatic consumption purpose.

**Recommendation**: Add "Data Structure Specification" section with:
- Complete JSON schema example showing nested structure
- CSV structure definition (recommend changes-only with stack info repeated)
- YAML structure (similar to JSON)
- Empty changeset handling for each format

**Status**: PENDING - Requires user approval to add this specification

---

### Finding 2: DOT Format Doesn't Make Sense ⚠️ CRITICAL

**Issue**: DOT format (GraphViz) is for graph relationships with nodes and edges. Changesets are tabular data with no inherent graph structure.

**Analysis**:
- DOT is used in fog for stack dependencies (stack A exports to stack B)
- Changesets are a flat list of resource changes
- Creating a meaningful graph would require:
  - Parsing CloudFormation template
  - Building resource dependency tree
  - Overlaying changeset actions
  - This is a separate feature, not a simple format conversion

**Decision**: Remove `dot` from supported formats for this feature.

**Status**: APPROVED - Requirements updated (AC 1.3)

---

### Finding 3: Empty Changeset Handling Undefined ⚠️ CRITICAL

**Issue**: No specification for how different formats should handle changesets with zero changes.

**Current behavior**: Table format shows "No resource changes" text message, which doesn't translate to JSON/YAML.

**Decision**: Each format should show behavior that makes sense for that format:
- Table/Markdown/HTML: Display stack info + "No resource changes" message
- JSON/YAML: Return structured output with empty arrays and zero summary counts
- CSV: Output headers only with no data rows

**Status**: APPROVED - Requirements updated (AC 2.7, 3.7, 3.8, 6.4)

---

### Finding 4: Configuration Precedence Unclear ⚠️ MODERATE

**Issue**: After removing hardcoded `viper.Set("output", "table")`, precedence order is unclear.

**Decision**: Configuration precedence is OUT OF SCOPE for this feature. This is already handled by the configuration system (viper/cobra) before the command executes. The describe changeset command should simply respect whatever format is configured, just like other fog commands.

**Status**: RESOLVED - Out of scope, already handled. Requirements updated (AC 1.4, 5.4) to clarify this.

---

### Finding 5: ANSI Code Handling Assumption ⚠️ MODERATE

**Issue**: AC 3.5 assumes ANSI code stripping works, but current code uses `bold()` color formatting directly in data assembly (lines 189, 223).

**Technical concern**: Code pollutes data with ANSI codes during assembly, requiring go-output v2 to clean them. Better pattern: store raw data, apply formatting only during rendering.

**Recommendation**: Store changeset data without formatting codes and apply format-specific styling during rendering. Colors preserved for terminal outputs (table, markdown), omitted for data formats (JSON, CSV, YAML).

**Status**: PENDING - May require refactoring during implementation

---

### Finding 6: Missing Testing Requirements 📝 ENHANCEMENT

**Issue**: No specification for testing expectations.

**Recommendation**: Add testing requirements section covering:
- Unit tests for JSON structure validation
- Unit tests for CSV headers and row count
- Unit tests for table format preservation
- Integration tests for end-to-end formatting
- Tests for empty changeset handling
- Tests for error conditions

**Status**: PENDING - Nice to have, can be added

---

### Finding 7: Scope Limitation May Need Helper Functions 📝 ENHANCEMENT

**Issue**: AC 4.1 says "modify only cmd/describe_changeset.go" but may need shared helpers.

**Recommendation**: Clarify that refactoring shared output utility functions in `cmd/output_helpers.go` is acceptable if needed for consistent formatting.

**Status**: PENDING - Minor clarification

---

### Finding 8: Error Message Specificity 📝 ENHANCEMENT

**Issue**: "Clear error messages" is vague.

**Recommendation**: Specify exact error message patterns for unsupported formats and rendering failures.

**Status**: PENDING - Nice to have, can be added

---

## Peer Review Consensus

Both design-critic and peer-review-validator agents independently identified the same critical issues:
- Missing data schema (both flagged as critical)
- DOT format inappropriate for tabular data (both recommended removal)
- Empty changeset handling needed (both identified as gap)

This consensus strongly validates these findings.

---

---

## Decision 5: Single Document Output (2025-11-03)

**Context**: Ensure all output is properly formatted and consistent across formats.

**Decision**: All output must be gathered into a single go-output v2 Builder document before rendering. This ensures consistent formatting and prevents mixing of Builder-based output with direct printf statements.

**Rationale**:
- Prevents inconsistencies between formats
- Ensures transformers (ANSI stripping, emoji handling) apply to all content
- Makes testing easier with single output artifact
- Follows go-output v2 best practices

**Implementation note**: The console URL (currently a separate printf) needs to be included in the Builder document structure.

**Status**: Approved by user - requirements updated (AC 3.1)

---

## Resolution Summary

All critical findings from the review process have been addressed:

1. ✅ **DOT format removed** - Not appropriate for tabular changeset data (AC 1.3 updated)
2. ✅ **Data structure specified** - New section 6 defines JSON/YAML/CSV structure (AC 6.1-6.4 added)
3. ✅ **Empty changeset handling defined** - Format-appropriate behavior specified (AC 2.7, 3.7, 3.8, 6.4 added)
4. ✅ **Configuration precedence clarified** - Out of scope, handled by existing config system (AC 1.4, 5.4 updated)
5. ✅ **Single document output required** - All content through Builder pattern (AC 3.1 updated)

## Ready for Design Phase

Requirements are now complete and approved. All critical issues have been resolved. Ready to proceed to Phase 2: Design Creation.

---

## Design Phase Findings (2025-11-03)

### Design Review Summary

The design document was reviewed by both design-critic and peer-review-validator agents. Both agents identified similar core issues, validating their importance.

### Critical Finding 1: Console URL Structure ⚠️

**Issue**: The design proposes using `builder.Text()` for the console URL, which creates a generic text array in JSON/YAML. Requirements specify the URL should be a named field.

**Decision Required**: Update design to include console URL as a field in the stack information table.

**Proposed Solution**:
```go
// In addStackInfoSection
if !deployment.IsDryRun {
    content["ConsoleURL"] = changeset.GenerateChangesetUrl(awsConfig)
    keys = append(keys, "ConsoleURL")
}
```

**Status**: PENDING user decision

---

### Critical Finding 2: JSON/YAML Structure Mismatch ⚠️

**Issue**: Requirement 6.1 specifies a flat structure with named sections (`stackInfo`, `changes`, etc.), but go-output v2's actual behavior wraps all tables in a `tables` array:

```json
{
  "tables": [
    {"title": "CloudFormation stack information", "data": [...]},
    {"title": "Changes for changeset-name", "data": [...]}
  ]
}
```

**Options**:
1. Update Requirement 6.1 to accept the `tables` array structure (maintains consistency with other fog commands)
2. Implement custom JSON marshaling (more complex, breaks consistency)

**Peer Review Recommendation**: Update requirements to match go-output v2 behavior. This is consistent across all fog commands and still supports programmatic consumption.

**Status**: PENDING user decision

---

### Critical Finding 3: CSV Structure Needs Verification ⚠️

**Issue**: Requirement 6.2 and design example show a merged CSV with "Section" column, but go-output v2's actual behavior with multiple tables produces separate CSV sections with headers between them.

**Recommendation**: Test actual CSV output and update either requirements or design to match reality.

**Status**: PENDING verification - can be addressed during implementation

---

### Moderate Finding 4: Empty Dangerous Changes Handling

**Issue**: Design uses `builder.Text("No dangerous changes")` which adds text to all formats. Requirement 6.3 specifies empty array for JSON/YAML.

**Recommendation**: Don't add text message - use empty table. Go-output v2 will handle format-appropriately.

**Status**: PENDING user decision

---

### Moderate Finding 5: ANSI Code Stripping Verification

**Issue**: Design continues to apply ANSI color codes during data assembly (for bold "Remove" actions). Should verify these are stripped from JSON/CSV/YAML.

**Recommendation**: Accept current approach but add explicit test requirement verifying no ANSI codes in structured formats.

**Status**: ACCEPTED - add to testing requirements

---

## Design Phase Decisions (2025-11-03)

All design review findings have been resolved with user approval:

### Decision 6: Console URL as Stack Info Field

**Context**: Design originally used `builder.Text()` for console URL, creating generic text array in JSON/YAML.

**Decision**: Add console URL as a field in the stack information table with "ConsoleURL" column.

**Rationale**:
- Makes URL programmatically accessible in all formats
- Provides structured field in JSON/YAML instead of text array
- Adds one column to stack info table (acceptable trade-off)
- Works consistently across all formats

**Status**: APPROVED - Design and requirements updated (AC 2.1, 2.5, 3.8, 6.1)

---

### Decision 7: JSON/YAML Structure Format

**Context**: Requirements initially specified flat structure with named sections, but go-output v2 wraps all tables in a `tables` array.

**Decision**: Update requirements to accept go-output v2's standard `tables` array structure.

**Rationale**:
- Maintains consistency with all other fog commands using go-output v2
- Still fully supports programmatic consumption - consumers can iterate tables array and index by title
- Implementing custom JSON marshaling would be complex and break consistency
- This is the actual behavior of the library

**Status**: APPROVED - Requirements updated (AC 3.2, 3.4, 6.1)

---

### Decision 8: CSV Multi-Section Format

**Context**: Design example showed merged CSV with "Section" column, but go-output v2 produces separate sections with headers.

**Decision**: Accept go-output v2's multi-section CSV format with section titles and headers between tables.

**Rationale**:
- Standard go-output v2 behavior
- Consistent with other fog commands
- Provides clear separation of concerns for CSV consumers
- Each section has its own appropriate headers

**Status**: APPROVED - Requirements updated (AC 3.3, 6.2)

---

### Decision 9: Empty Dangerous Changes Representation

**Context**: Design used `builder.Text("No dangerous changes")` which adds text to all formats.

**Decision**: Use empty table with headers but no data rows for "no dangerous changes" scenario.

**Rationale**:
- Go-output v2 handles this appropriately per format:
  - Table: shows title with no rows
  - JSON/YAML: includes table with empty data array
  - CSV: shows headers only, no data rows
- Eliminates need for format-specific logic
- Provides structural representation in data formats

**Status**: APPROVED - Design updated

---

### Decision 10: ANSI Code Stripping Verification

**Context**: Design continues to apply ANSI color codes during data assembly (for bold "Remove" actions).

**Decision**: Accept current approach but add explicit test requirement verifying no ANSI codes in JSON/YAML/CSV output.

**Rationale**:
- Go-output v2 has transformers that strip ANSI codes from structured formats
- Refactoring to apply formatting at render time would be complex
- Testing will verify the transformers work correctly
- If issues arise, can be addressed in future refactoring

**Status**: APPROVED - Testing requirements updated in design

---

## Phase 3 Ready

All design review findings have been resolved. Design and requirements are aligned with go-output v2's actual behavior. Ready to proceed to Phase 3: Task Planning.

---

## Additional Clarification (2025-11-03)

### Shared Function Preservation

**Finding**: The functions marked for "deprecation" in the design are actually shared with other commands:
- `showChangeset()` used by deploy_helpers.go
- `buildChangesetDocument()` used by history.go
- `printBasicStackInfo()` used by deploy.go

**Decision**: These shared functions MUST be preserved. Only `printChangeset()` (used only in describe_changeset.go) can be removed.

**Rationale**:
- Scope limitation (AC 4.1-4.3) requires not affecting other commands
- Deploy and history commands depend on these shared functions
- New implementation will use new functions alongside the old ones

**Impact on Implementation**:
- Keep all shared functions unchanged
- Add new functions for describe changeset command
- Only remove `printChangeset()` in cleanup phase

**Status**: Clarified in design document

---

## Phase 3 Complete: Task Planning (2025-11-03)

Implementation tasks have been created using the rune CLI tool. The task file is located at `specs/changeset-output-format/tasks.md` and contains 20 tasks organized into 4 phases:

**Phase 1: Refactor Output Functions** (4 tasks)
- Create new functions: addStackInfoSection(), buildChangesetData(), addChangesetSections()
- Update existing tests to use new functions

**Phase 2: Update Command Function** (4 tasks)
- Remove hardcoded table enforcement
- Create buildAndRenderChangeset() orchestration function
- Update describeChangeset() to use new implementation
- Verify backward compatibility

**Phase 3: Testing** (7 tasks)
- Unit tests for all output formats
- Tests for empty changesets, console URL, ANSI code stripping, JSON structure
- Integration tests
- Golden file verification

**Phase 4: Cleanup** (5 tasks)
- Remove only printChangeset() function (preserve shared functions)
- Clean up imports
- Run formatting and linting
- Update CHANGELOG.md

Each task includes:
- Clear, actionable title
- Detailed implementation steps
- File references for context
- Phase grouping for logical workflow

**Next Steps**: Begin implementation starting with Phase 1, Task 1. Use `rune next specs/changeset-output-format/tasks.md` to get the next task, and `rune progress specs/changeset-output-format/tasks.md <task-id>` to mark tasks as in-progress.
