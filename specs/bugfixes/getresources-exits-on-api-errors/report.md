# Bugfix Report: GetResources Exits Process on AWS API Errors

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`GetResources` in `lib/resources.go` uses `log.Fatalln` and `log.Fatalf` to handle AWS API errors. These functions call `os.Exit(1)`, terminating the process from library code. Library functions should return errors so the cmd layer can format and present them consistently.

**Reproduction steps:**
1. Call `GetResources` with an AWS client that returns an API error from `DescribeStacks` or `DescribeStackResources`.
2. Observe that the process exits immediately via `log.Fatal` instead of returning an error.

**Impact:** The process exits from library code, bypassing the cmd layer's error formatting (`failWithError`). This prevents consistent error presentation and makes the library unsafe for reuse in contexts where process exit is unacceptable.

## Investigation Summary

Systematic inspection of `lib/resources.go` identified four `log.Fatal` call sites that terminate the process.

- **Symptoms examined:** `log.Fatalln` / `log.Fatalf` calls in library code that exit the process.
- **Code inspected:** `lib/resources.go` (GetResources), `lib/drift.go` (GetUncheckedStackResources), `cmd/resources.go` (listResources).
- **Hypotheses tested:** Confirmed that the same anti-pattern fixed in T-339 (drift detection) exists in GetResources.

## Discovered Root Cause

`GetResources` uses `log.Fatalln` and `log.Fatalf` at four error paths instead of returning errors to callers:

1. Line 43: `log.Fatalln("error:", bne.Err)` -- DescribeStacks pagination OperationError
2. Line 45: `log.Fatalln(err)` -- DescribeStacks pagination generic error
3. Line 75: `log.Fatalln(err)` -- throttling retry still fails
4. Line 79: `log.Fatalf(...)` -- non-throttling API error on DescribeStackResources

**Defect type:** Error handling defect (process exit from library code).

**Why it occurred:**
- Why does the process exit? Because `log.Fatal` calls `os.Exit(1)`.
- Why is `log.Fatal` used? The function was written before the project adopted error-returning conventions.
- Why wasn't this caught? The existing test (`TestGetResourcesNonThrottlingError`) explicitly expected process exit via subprocess re-execution.

**Contributing factors:** The same pattern was already fixed in `lib/drift.go` (T-339), but `GetResources` was not updated at the same time.

## Resolution for the Issue

**Changes made:**
- `lib/resources.go` - Changed `GetResources` signature from `[]CfnResource` to `([]CfnResource, error)`. Replaced all `log.Fatal` calls with error returns. Removed `log` import.
- `lib/drift.go` - Updated `GetUncheckedStackResources` signature from `[]CfnResource` to `([]CfnResource, error)`. Propagated error from `GetResources`.
- `cmd/resources.go` - Added error handling for `GetResources` return value via `failWithError`.
- `lib/resources_test.go` - Updated all tests to use the new `([]CfnResource, error)` return signature. Replaced process-exit test with direct error assertion tests.
- `lib/drift_test.go` - Updated `GetUncheckedStackResources` tests to use the new `([]CfnResource, error)` return signature.

**Approach rationale:** Follows the exact same pattern applied in T-339 for drift detection. Returns errors from library code and lets the cmd layer decide how to present them.

**Alternatives considered:**
- Add panic recovery in command layer - rejected because it hides the root problem and keeps library code unsafe for reuse.

## Regression Test

**Test file:** `lib/resources_test.go`
**Test names:**
- `TestGetResourcesPaginationError` - DescribeStacks pagination errors returned
- `TestGetResourcesNonThrottlingAPIError` - non-throttling DescribeStackResources errors returned
- `TestGetResourcesThrottlingRetryExhausted` - throttling retry failure returned
- `TestGetResourcesGenericDescribeStackResourcesError` - generic errors returned

**What it verifies:** AWS API errors in GetResources are returned as errors instead of calling log.Fatal/os.Exit.

**Run command:** `go test ./lib -run 'TestGetResources(PaginationError|NonThrottlingAPIError|ThrottlingRetryExhausted|GenericDescribeStackResourcesError)' -count=1`

## Affected Files

| File | Change |
|------|--------|
| `lib/resources.go` | Changed return type to `([]CfnResource, error)`, replaced `log.Fatal` with error returns |
| `lib/drift.go` | Updated `GetUncheckedStackResources` to return `([]CfnResource, error)` |
| `cmd/resources.go` | Added error handling for `GetResources` return |
| `lib/resources_test.go` | Updated tests for new signature, replaced exit test with error assertion tests |
| `lib/drift_test.go` | Updated `GetUncheckedStackResources` tests for new signature |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified all callers of GetResources and GetUncheckedStackResources handle the new error return.

## Prevention

**Recommendations to avoid similar bugs:**
- Never use `log.Fatal` in library code; reserve it for `main()` or top-level initialization only.
- Return errors from library functions and let cmd-layer handlers format them.
- When fixing an error-handling pattern in one function, audit related functions for the same issue.

## Related

- Transit ticket: `T-465`
- Previous similar fix: T-339 (drift detection crashes on AWS API errors)
