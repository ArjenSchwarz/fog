# Decision Log - Deploy Output Feature

This document tracks key decisions made during the requirements and design phases of the deploy-output feature.

## Decisions

### D1: stdout/stderr Separation Strategy
**Date:** 2025-11-06
**Status:** Decided
**Context:** The deploy command produces both streaming progress information and final formatted output. We needed to determine how to separate these outputs to follow CLI best practices.

**Decision:**
- All progress and diagnostic information goes to **stderr**:
  - Streaming table showing deployment progress
  - Stack information displayed before deployment
  - Changeset overview displayed before deployment
  - Interactive prompts (e.g., "Do you want to continue?")
  - Deployment progress updates

- Final formatted output goes to **stdout**:
  - JSON/CSV/YAML/Markdown/table output at the end of deployment
  - Only produced when a format is explicitly specified (or table with file target)

**Rationale:**
- Follows Unix/POSIX standard practice used by tools like `rm`, `mv`, `find`, `bash`
- Enables users to redirect data output while still seeing progress: `fog deploy --output json > result.json`
- Keeps stdout clean for the actual data product
- Allows automation tools to suppress progress with `2>/dev/null` while capturing data
- Interactive prompts on stderr is the standard pattern, making it easy to detect non-interactive environments

**Alternatives Considered:**
1. Everything to stdout - rejected because it mixes progress with data output
2. Streaming to stdout, final output to stderr - rejected because it inverts the purpose of these streams
3. Suppress streaming when formatted output requested - rejected because users want to see real-time progress

**Consequences:**
- Implementation must ensure all progress uses stderr writers
- Documentation must explain the stream separation pattern
- Users benefit from standard Unix redirection patterns
- CI/CD tools can easily capture just the data output

---

### D2: Format Handling for Streaming vs Final Output
**Date:** 2025-11-06
**Status:** Decided
**Context:** The current code forces table format via `viper.Set("output", "table")` at cmd/deploy.go:77. With the addition of multi-format output, we needed to determine how to handle format selection for streaming progress vs final output.

**Decision:**
- **stderr streaming output** always uses table format (appropriate for real-time progress display)
- **stdout final output** uses the user's requested format (JSON/CSV/YAML/Markdown/table)
- These are independent rendering paths, so no format override/restore logic is needed

**Rationale:**
- The stderr/stdout separation naturally separates the two rendering concerns
- Table format is ideal for streaming progress updates (readable, real-time)
- User's format preference applies only to the final data output on stdout
- Eliminates complexity of saving/restoring format settings

**Alternatives Considered:**
1. Save user format, force table, then restore - rejected as unnecessarily complex
2. Separate "streaming format" and "output format" settings - rejected as the stderr/stdout split handles this naturally

**Consequences:**
- Streaming progress rendering always uses table format (stderr)
- Final output rendering uses user's requested format (stdout)
- No need to override or restore Viper settings
- Cleaner implementation without format juggling logic

---

### D3: "Applied Changes" Definition
**Date:** 2025-11-06
**Status:** Decided
**Context:** Requirement 6.3 states the deployment summary should include "applied changes collected from the changeset" but this was ambiguous - it could mean planned changes or actual executed changes.

**Decision:**
Applied changes shall mean the **planned changes from the changeset** (the resources that were intended to be created/updated/deleted).

**Rationale:**
- Changeset data is readily available and reliable
- Parsing actual executed changes from stack events is complex and brittle
- The changeset represents the complete plan that was executed
- For successful deployments, planned changes match actual changes
- Simpler implementation with consistent data structure

**Alternatives Considered:**
1. Actual executed changes parsed from stack events - rejected as too complex and fragile
2. Combination of both planned and actual - rejected as unnecessary and confusing

**Consequences:**
- Implementation captures changeset data immediately after creation
- Output shows what was planned/executed, not parsed from events
- Consistent data structure across all deployment types
- Design phase will need to specify changeset data structure

---

### D4: Dry-Run Output Reuses Describe Changeset Builder
**Date:** 2025-11-06
**Status:** Decided
**Context:** Requirement 5.2 states dry-run output should be "identical to describe changeset" command. We needed to clarify whether this means reusing the same code or just producing similar output.

**Decision:**
Dry-run output shall reuse the exact same output builder function that the `describe changeset` command uses.

**Rationale:**
- Ensures true identical output (not just similar)
- Avoids code duplication and maintenance burden
- Single source of truth for changeset formatting
- Any improvements to describe changeset automatically benefit deploy dry-run

**Alternatives Considered:**
1. Duplicate the changeset formatting logic - rejected due to maintenance issues
2. Similar output but different implementation - rejected as it violates "identical" requirement

**Consequences:**
- Design phase must ensure changeset output builder is properly extracted/accessible
- Both commands will share the same formatting code
- Changes to changeset output affect both commands equally
- Implementation must handle builder's dependencies (if any)

---

### D6: Quiet Mode for CI/CD Automation
**Date:** 2025-11-06
**Status:** Decided
**Context:** The requirements specified that streaming progress output to stderr is always shown. This is problematic for CI/CD automation where users may only want the final result without progress information.

**Decision:**
Add a `--quiet` flag that suppresses all stderr output, showing only the final stdout output.

**Rationale:**
- CI/CD environments often don't need real-time progress
- Reduces log verbosity in automated pipelines
- Users can still get structured data output via stdout
- Standard pattern in CLI tools (curl --silent, wget --quiet, etc.)
- Makes automation scripts cleaner and logs more focused

**Alternatives Considered:**
1. No quiet mode, users redirect stderr themselves - rejected as less user-friendly
2. `--no-stream` flag name - rejected in favor of more common `--quiet` convention
3. Separate flags for different types of output - rejected as overengineered

**Consequences:**
- When `--quiet` is enabled, no stderr output is produced
- Interactive prompts must be disabled in quiet mode (fail if input needed)
- Progress indicators, tables, and prompts are all suppressed
- Only final formatted output appears on stdout
- Documentation must explain quiet mode behavior and CI/CD use cases

---

### D5: Handling Missing Deployment Scenarios
**Date:** 2025-11-06
**Status:** Decided
**Context:** The initial requirements didn't cover several deployment scenarios: user cancellation, no-change deployments, --create-changeset mode, and table format with file output.

**Decisions:**

1. **User Cancellation (Ctrl+C):**
   - No special handling required
   - Whatever stderr output has already been produced remains visible
   - No final formatted output is produced
   - Process exits naturally

2. **No-Change Deployments:**
   - Treated as successful outcome (exit code 0)
   - Final output includes message stating no changes were found
   - Includes current stack information
   - Formatted according to specified output format

3. **--create-changeset mode:**
   - Behaves like --dry-run
   - Outputs changeset details to stdout
   - Uses same builder as describe changeset command
   - DOES NOT delete changeset after output (key difference from dry-run)

4. **--output table --file behavior:**
   - Uses new mode (consistent with all other formats)
   - Writes final formatted output to file
   - NOT legacy mode (no longer special-cased)

**NOTE:** The --deploy-changeset flag was originally considered in this decision but has been removed from scope. It is currently broken (separate bug) and will be fixed independently of this feature.

**Rationale:**
- User cancellation needs no special handling - it's an interrupt
- No-change is a valid successful outcome that users need to process
- --create-changeset mode should align with dry-run semantics (preview without execution)
- Table format should be consistent with other formats (no special legacy behavior)

**Alternatives Considered:**
1. User cancellation produces partial output - rejected as unnecessary and could be incomplete/invalid
2. No-change produces no output - rejected as users need confirmation
3. Special modes remain unchanged - rejected as it creates inconsistency
4. Table with file uses legacy mode - rejected as it creates confusing exceptions

**Consequences:**
- Ctrl+C handling is simple (no special code)
- No-change scenario requires new output structure
- Create-changeset requires minimal changes (don't delete changeset)
- Table format behavior is now consistent across all use cases
- All output modes follow the same patterns (no special cases)

---

### D7: Clarifications from Review Feedback
**Date:** 2025-11-06
**Status:** Decided
**Context:** After agent review, several critical questions needed clarification regarding backwards compatibility, data structures, changeset lifecycle, and technical implementation.

**Clarifications:**

1. **Backwards Compatibility and Default Format:** (SUPERSEDED by D8 - see below)
   - Fog always has a format defined - the default is table
   - ~~Backwards compatibility means "table format without file target" not "no format specified"~~
   - ~~When table format is used without a file target, behavior remains identical to current implementation (streaming only, no final stdout output)~~
   - **NOTE:** This interpretation was superseded by D8, which clarifies that final results always go to stdout for all deployment outcomes

2. **Data Structure Consistency:**
   - All output formats (JSON, CSV, YAML, Markdown, table) will use the same data structure as the current table-based output
   - No new fields or schema changes - just different serialization formats
   - Existing table structure is well-proven and comprehensive

3. **Changeset Deletion Policy:**
   - Manual deletion only occurs in two scenarios:
     - Dry-run deployments (changeset created for preview, then deleted)
     - User cancels deployment after reviewing changeset
   - For all other scenarios, CloudFormation handles changeset deletion automatically after execution
   - For `--create-changeset`, changeset is explicitly NOT deleted (preserved for later `--deploy-changeset`)
   - No conflict between different execution paths

4. **Technical Implementation - Stream Separation:**
   - go-output v2.6.0 (releasing 2025-11-07) adds `StderrWriter` support
   - This resolves the technical concern about routing output to stderr
   - Existing `fmt.Println()` statements for progress will be updated to use stderr writer
   - Two rendering paths: stderr for streaming progress (using StderrWriter), stdout for final formatted output (using StdoutWriter)

**Rationale:**
- Clarifying that table is the default resolves backwards compatibility ambiguity
- Reusing existing data structure avoids breaking changes and reduces implementation complexity
- Explicit changeset lifecycle policy prevents confusion about when deletion occurs
- go-output v2.6.0 StderrWriter provides the technical foundation for stream separation

**Consequences:**
- Requirements now specify table as default format explicitly
- No need to define new data structures during design phase
- Changeset handling is clear: manual delete only for dry-run and cancellation
- Implementation can leverage new StderrWriter for proper stream separation
- All fmt.Println() calls for progress output will need updating

---

### D8: Final Review Clarifications and Scope Decisions
**Date:** 2025-11-06
**Status:** Decided
**Context:** Final review by design-critic and peer-review-validator agents identified critical issues. User provided clarifications on scope and approach.

**Clarifications:**

1. **stdout Always Receives Final Results:**
   - Changed from "table without file = no stdout" to "final results always go to stdout"
   - Resolves quiet mode issue (quiet + table would produce zero output)
   - Resolves failed deployment contradiction (errors always go to stdout)
   - Simpler model: stderr = progress, stdout = results

2. **Quiet Mode Auto-Approval:**
   - `--quiet` flag automatically enables non-interactive mode (auto-approve all prompts)
   - No need for user to specify both `--quiet` and `--yes`
   - Simplifies CI/CD usage - single flag for automation

3. **Stream Ordering:**
   - Flush stderr before writing to stdout to ensure proper ordering
   - No goroutines expected in output rendering, so interleaving should not be an issue
   - Header written to stdout before formatted data provides clear separation

4. **File Target Specification:**
   - Built-in fog functionality (existing `--file` flag)
   - No need to define in requirements
   - Out of scope for this feature

5. **go-output v2.6.0 StderrWriter Confirmed:**
   - User (go-output maintainer) confirms v2.6.0 is available
   - StderrWriter support is available as documented
   - API documentation updated at docs/research/go-output-v2/API.md

**Scope Exclusions:**

6. **--deploy-changeset Flag:**
   - Functionality used to work but is currently broken (separate bug)
   - Removed from requirements - will be fixed separately
   - Does not impact this feature

7. **Pre-Deployment Error Handling:**
   - Out of scope - no changes to current behavior
   - Changeset creation failures, validation errors, etc. remain unchanged

8. **User Cancellation During Changeset Creation:**
   - Out of scope - no changes to current cleanup behavior

9. **No-Changes Output Structure Details:**
   - Out of scope for requirements phase
   - Will be defined during design phase using same structure as current output

**Rationale:**
- Always outputting to stdout provides consistent model and resolves contradictions
- Quiet mode auto-approval makes automation simpler
- Scope exclusions prevent feature creep and focus on core functionality
- Separating --deploy-changeset bug fix prevents blocking this feature

**Consequences:**
- Simpler backwards compatibility model
- No special cases for table format
- Quiet mode is single-flag automation solution
- --deploy-changeset removed from requirements (will be separate fix)
- Pre-deployment and cancellation behavior unchanged
- Clear separation between progress (stderr) and results (stdout)

---
