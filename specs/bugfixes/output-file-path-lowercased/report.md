# Bugfix Report: Output File Path Lowercased

**Date:** 2025-07-14
**Status:** Fixed

## Description of the Issue

When the `output-file` configuration option was set (via `--file` flag or config file), the entire file path was lowercased before being passed to the file writer. This meant a path like `/Path/To/MyProject/DeployReport.json` would be written to `/path/to/myproject/deployreport.json` instead.

**Reproduction steps:**
1. Run any fog command with `--file /Path/To/MixedCase/Report.json --output json`
2. Observe that the file is created at the lowercased path `/path/to/mixedcase/report.json`

**Impact:** On case-sensitive filesystems (Linux, some macOS configurations), the file would be created at the wrong path or fail to create if the lowercased directory path didn't exist. On case-insensitive filesystems (default macOS), the bug was masked because both paths resolve to the same file.

## Investigation Summary

Traced the output-file configuration value through all code paths from CLI flag binding to file writer creation.

- **Symptoms examined:** File path being lowercased when output-file is set
- **Code inspected:** `config/config.go` (GetOutputOptions, GetFileOutputOptions, GetString, GetLCString), `cmd/root.go` (flag binding), `cmd/output_helpers.go` (renderDocument), go-output/v2 FileWriter
- **Hypotheses tested:** Viper lowercasing values (ruled out — Viper only lowercases keys), go-output library lowercasing paths (ruled out — preserves paths), YAML parser lowercasing (ruled out)

## Discovered Root Cause

The `Config.GetLCString()` method was used to read the `output-file` configuration value. `GetLCString` calls `strings.ToLower()` on the value, which is correct for format names (e.g., "json", "table") but incorrect for file paths.

**Defect type:** Incorrect API usage — `GetLCString` used where `GetString` was needed

**Why it occurred:** The `GetLCString` method was originally designed for output format names which should be case-insensitive. When the `output-file` setting was added, it was read with the same `GetLCString` method by pattern, without considering that file paths are case-sensitive.

**Contributing factors:**
- No regression test existed for file path case preservation
- macOS case-insensitive filesystem masked the bug during development
- The initial implementation (commit `87fa2e8`) used `GetLCString("output-file")` in `NewOutputSettings()`
- The v2 migration (commit `88849cd`) carried the same bug into `GetOutputOptions()`

## Resolution for the Issue

**Changes made:**
- `config/config.go:188` — Changed `GetLCString("output-file")` to `GetString("output-file")` in `GetOutputOptions()` (fixed in prior commit `63eeb79`)
- `config/config.go:225` — Used `GetString("output-file")` in `GetFileOutputOptions()` (added with correct call in commit `63eeb79`)
- `config/config_test.go` — Added `TestConfig_OutputFilePathCasePreserved` regression test verifying config reading preserves path case
- `cmd/output_helpers_test.go` — Added `TestRenderDocument_FilePathCasePreserved` and `TestRenderDocument_FilePathCasePreserved_DifferentFormats` end-to-end regression tests

**Approach rationale:** The fix uses `GetString` (case-preserving) instead of `GetLCString` (lowercasing) for the file path, while keeping `GetLCString` for format names where case-insensitivity is desired. Regression tests verify both the config reading layer and the end-to-end file creation.

**Alternatives considered:**
- Adding a dedicated `GetFilePath` method — Not chosen because `GetString` already provides the correct behavior; a separate method would add unnecessary API surface

## Regression Test

**Test file:** `config/config_test.go`
**Test name:** `TestConfig_OutputFilePathCasePreserved`

**What it verifies:** That `Config.GetString("output-file")` returns the exact file path as set, preserving all case.

**Test file:** `cmd/output_helpers_test.go`
**Test name:** `TestRenderDocument_FilePathCasePreserved`

**What it verifies:** End-to-end verification that when `output-file` is set to a mixed-case path, the file is created with the correct case-preserved name. Works on both case-sensitive and case-insensitive filesystems.

**Run command:** `go test ./config/ ./cmd/ -v -run "FilePathCasePreserved"`

## Affected Files

| File | Change |
|------|--------|
| `config/config.go:188` | Use `GetString` instead of `GetLCString` for `output-file` (fixed in prior commit) |
| `config/config.go:225` | Use `GetString` for `output-file` in `GetFileOutputOptions` (fixed in prior commit) |
| `config/config_test.go` | Added `TestConfig_OutputFilePathCasePreserved` regression test |
| `cmd/output_helpers_test.go` | Added two end-to-end regression tests for file path case preservation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

**Manual verification:**
- Traced the `output-file` config value through all code paths: flag binding (`cmd/root.go`), config reading (`config/config.go`), file writer creation (go-output/v2), and file writing
- Confirmed no remaining calls to `GetLCString("output-file")` in any Go source files
- Verified the go-output/v2 `FileWriter` preserves directory and filename case

## Prevention

**Recommendations to avoid similar bugs:**
- Always use `GetString` (not `GetLCString`) for configuration values that represent file paths, URLs, or other case-sensitive identifiers
- The `GetLCString` method should only be used for values where case-insensitive comparison is needed (e.g., format names like "json", "table")
- Add regression tests for case preservation when handling user-provided paths
- Test on case-sensitive filesystems (Linux CI) to catch case-related bugs that are masked on macOS

## Related

- Commit `87fa2e8`: Original introduction of `output-file` support using `GetLCString`
- Commit `88849cd`: v2 migration that carried the bug forward
- Commit `63eeb79`: Fix that changed to `GetString` (PR #80)
- Transit ticket: T-112
