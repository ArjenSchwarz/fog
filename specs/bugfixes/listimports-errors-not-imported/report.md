# Bugfix Report: ListImports Errors Treated as "Not Imported"

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

When `GetExports` or `FillImports` calls the AWS `ListImports` API and receives any error, the error is silently swallowed and the export is marked as `Imported=false`. This means real errors like throttling, permission denied, or service outages produce incorrect "not imported" results instead of failing visibly.

**Reproduction steps:**
1. Have an AWS account with CloudFormation exports
2. Run `fog exports` with insufficient IAM permissions for `ListImports` (or during API throttling)
3. All exports show as "not imported" regardless of whether they actually are

**Impact:** Medium severity. Users get silently incorrect output, believing exports are not imported when in reality the import status could not be determined due to API errors.

## Investigation Summary

The code has TODO comments at both bug locations acknowledging the issue was known but unresolved.

- **Symptoms examined:** Both `GetExports` (line 66-69) and `FillImports` (line 122-125) in `lib/outputs.go` catch all `ListImports` errors and set `Imported=false`
- **Code inspected:** `lib/outputs.go` (GetExports, FillImports), `lib/stacks.go` (FillImports caller), `cmd/exports.go` (GetExports caller), `lib/interfaces.go` (API interfaces)
- **Hypotheses tested:** Confirmed that the only expected error is "Export 'X' is not imported by any stack." — all other errors are real failures that should be propagated

## Discovered Root Cause

Both `GetExports` and `FillImports` treat all `ListImports` errors as meaning "export is not imported." Only the specific error message "is not imported by any stack" indicates that the export genuinely has no importers. Other errors (throttling, access denied, service errors) are real failures.

**Defect type:** Missing error discrimination / overly broad error handling

**Why it occurred:** The initial implementation took a shortcut by treating all errors as "not imported." The TODO comments show the developer was aware this was incorrect but deferred the fix.

**Contributing factors:** The `ListImports` API returns a regular error (not a typed error) for the "not imported" case, making it easy to fall into the trap of catching all errors uniformly.

## Resolution for the Issue

**Changes made:**
- `lib/outputs.go` - Added `isNotImportedError()` helper that checks for the specific "is not imported by any stack" message
- `lib/outputs.go` - Changed `FillImports` to return `error`; only returns nil for the "not imported" case
- `lib/outputs.go` - Changed `GetExports` to return `([]CfnOutput, error)`; collects and returns real ListImports errors
- `lib/stacks.go` - Updated `FillImports` caller to handle the returned error
- `cmd/exports.go` - Updated `GetExports` caller to handle the returned error

**Approach rationale:** Minimal change that distinguishes the expected "not imported" error from real API failures, propagating real errors to callers.

**Alternatives considered:**
- Add an `ImportError` field to `CfnOutput` for per-export error tracking - Rejected as more complex than needed; callers can fail on any error
- Use `smithy.APIError` type assertion instead of string matching - The "not imported" error from CloudFormation does not use a distinct error code; the message string is the only reliable discriminator

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test names:** `TestFillImports_NotImportedError`, `TestFillImports_PropagatesRealErrors`, `TestGetExports_PropagatesListImportsError`, `TestGetExports_NotImportedErrorSetsImportedFalse`

**What it verifies:** The "not imported" error is handled correctly (Imported=false, no error returned), while other errors (throttling, access denied, generic) are propagated to callers.

**Run command:** `go test ./lib/ -run "TestFillImports_NotImportedError|TestFillImports_PropagatesRealErrors|TestGetExports_PropagatesListImportsError|TestGetExports_NotImportedErrorSetsImportedFalse" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Added `isNotImportedError()`, changed `FillImports` and `GetExports` signatures to return errors, added error discrimination logic |
| `lib/outputs_test.go` | Added 4 regression tests and `perExportMockCFNClient` |
| `lib/stacks.go` | Updated `FillImports` caller to handle returned error |
| `cmd/exports.go` | Updated `GetExports` caller to handle returned error |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When catching errors from AWS APIs, always check for the specific expected error condition rather than treating all errors as a particular state
- Remove TODO comments by addressing them promptly; they indicate known defects

## Related

- Transit ticket: T-514
