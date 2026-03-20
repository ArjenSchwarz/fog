# Bugfix Report: Fix RunPrechecks Command Parsing for Quoted Args

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`RunPrechecks` in `lib/files.go` uses `strings.Split(precheck, " ")` to parse precheck command strings into a command and its arguments. This naive split does not handle quoted arguments, causing two problems:

1. When `$TEMPLATEPATH` resolves to a path containing spaces (e.g., `path with spaces/template.yaml`), the path is broken into multiple arguments.
2. When users wrap arguments in quotes (single or double), the quotes are preserved as literal characters in the argument values rather than being stripped.

**Reproduction steps:**
1. Configure a precheck command with a quoted argument, e.g., `cfn-lint -t "$TEMPLATEPATH"`
2. Deploy a template whose path contains spaces
3. The precheck fails because the path is split into multiple arguments and the quotes are kept as literal characters

**Impact:** Precheck commands fail silently or with confusing errors when template paths contain spaces or when users follow standard shell quoting conventions in their configuration.

## Investigation Summary

- **Symptoms examined:** `strings.Split(precheck, " ")` on line 90 of `lib/files.go` splits every space regardless of quoting context
- **Code inspected:** `lib/files.go:RunPrechecks`, `cmd/deploy_helpers.go:runPrechecks`, configuration examples
- **Hypotheses tested:** The only hypothesis was the naive split — confirmed as root cause

## Discovered Root Cause

`strings.Split(precheck, " ")` treats every space character as a delimiter with no awareness of shell quoting conventions.

**Defect type:** Missing feature / insufficient parsing

**Why it occurred:** The original implementation used the simplest possible approach to split a command string, which works for commands without spaces in arguments but fails for the general case.

**Contributing factors:** Configuration examples don't use quoted arguments, so the deficiency wasn't caught during development or testing.

## Resolution for the Issue

**Changes made:**
- `lib/files.go` - Added `splitShellArgs()` function that parses command strings with awareness of single and double quotes, including backslash-escaped quotes inside double-quoted strings
- `lib/files.go` - Changed `RunPrechecks` to use `splitShellArgs()` instead of `strings.Split(precheck, " ")`, and added a guard for empty parse results

**Approach rationale:** A self-contained shell argument parser avoids adding an external dependency while correctly handling the most common quoting patterns that users encounter. The implementation follows standard shell quoting semantics: double quotes and single quotes delimit arguments, spaces inside quotes are preserved, and quotes are stripped from the result.

**Alternatives considered:**
- **Add `google/shlex` or `mattn/go-shellwords` dependency** - Would handle more edge cases but adds a dependency for a straightforward parsing task
- **Document that spaces are unsupported** - Would leave the bug unfixed and limit usability

## Regression Test

**Test file:** `lib/files_test.go`
**Test names:** `TestSplitShellArgs`, `TestRunPrechecksQuotedArgs`

**What it verifies:**
- `TestSplitShellArgs` verifies that the new shell argument parser correctly handles double-quoted args, single-quoted args, mixed quoted/unquoted args, empty quoted args, consecutive spaces, and escaped quotes.
- `TestRunPrechecksQuotedArgs` verifies that `RunPrechecks` correctly executes commands with quoted arguments containing spaces (using `echo` as a safe test command).

**Run command:** `go test ./lib/ -run "TestSplitShellArgs|TestRunPrechecksQuotedArgs" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/files.go` | Added `splitShellArgs()`, updated `RunPrechecks` to use it |
| `lib/files_test.go` | Added regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- N/A (unit-testable fix)

## Prevention

**Recommendations to avoid similar bugs:**
- When parsing user-provided command strings, always use a shell-aware parser rather than naive string splitting
- Add test cases with quoted arguments and paths containing spaces for any command-parsing code

## Related

- Transit ticket: T-378
