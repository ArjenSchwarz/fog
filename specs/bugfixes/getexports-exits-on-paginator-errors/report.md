# Bugfix Report: GetExports Exits Process on Paginator Errors

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `GetExports` function in `lib/outputs.go` called `log.Fatalln()` when the DescribeStacks paginator encountered an error. `log.Fatalln` calls `os.Exit(1)`, which terminates the process immediately from library code. This bypassed caller error handling and made the error path untestable.

**Reproduction steps:**
1. Call `GetExports` with a service client that returns an error from `DescribeStacks`
2. The process exits immediately via `log.Fatalln` instead of returning an error
3. The cmd layer's `failWithError` (which provides formatted error output) is never reached

**Impact:** Medium. Any AWS API error during pagination (auth failures, throttling, network issues) would terminate the CLI abruptly without the standard error formatting, and the error path could not be tested.

## Investigation Summary

- **Symptoms examined:** `log.Fatalln` calls in `GetExports` paginator error handling at lines 43 and 45
- **Code inspected:** `lib/outputs.go` (GetExports), `lib/resources.go` (GetResources for comparison, same pattern), `cmd/exports.go` (caller), `lib/interfaces.go` (API interfaces)
- **Hypotheses tested:** Confirmed that the function signature returns `[]CfnOutput` with no error, making it impossible for callers to handle paginator failures

## Discovered Root Cause

`GetExports` used `log.Fatalln` to handle paginator errors instead of returning them to the caller. This is a process-termination call from library code, which violates Go conventions where library code should return errors and let the caller decide how to handle them.

**Defect type:** Incorrect error handling pattern (library code terminates process)

**Why it occurred:** The original implementation used `log.Fatalln` as a quick way to handle errors, likely carried over from early development when the separation between lib and cmd layers was less defined.

**Contributing factors:** The same pattern exists in `lib/resources.go` (GetResources), suggesting this was a common early convention that was never refactored.

## Resolution for the Issue

**Changes made:**
- `lib/outputs.go` - Changed `GetExports` return type from `[]CfnOutput` to `([]CfnOutput, error)`. Replaced `log.Fatalln` calls with `return nil, fmt.Errorf(...)` wrapping the original error. Replaced `log` import with `fmt`.
- `cmd/exports.go` - Updated caller to handle the new error return with `failWithError(err)`.
- `lib/outputs_test.go` - Updated existing tests to handle the new return signature. Added two regression tests for the error paths.

**Approach rationale:** Returning errors from library functions is the standard Go convention. The cmd layer already has `failWithError` for formatting and displaying errors to users.

**Alternatives considered:**
- Wrapping the error in a custom `FogError` type - Not necessary for this change; the existing `fmt.Errorf` wrapping with `%w` preserves the error chain and is sufficient.
- Also fixing `GetResources` which has the same pattern - Out of scope for this ticket; should be tracked separately.

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test names:** `TestGetExports_PaginatorError`, `TestGetExports_OperationError`

**What it verifies:** That `GetExports` returns an error (instead of terminating the process) when the paginator encounters a general error or a `smithy.OperationError`.

**Run command:** `go test ./lib/ -run "TestGetExports_PaginatorError|TestGetExports_OperationError" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Changed return type to `([]CfnOutput, error)`, replaced `log.Fatalln` with error returns |
| `cmd/exports.go` | Updated caller to handle error return |
| `lib/outputs_test.go` | Updated existing tests, added `TestGetExports_PaginatorError` and `TestGetExports_OperationError` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Audit remaining `log.Fatalln` / `log.Fatalf` calls in the `lib/` package (notably `lib/resources.go`) and refactor them to return errors
- Consider adding a linter rule to flag `log.Fatal*` usage in library code
- Library functions should always return errors; only `main()` or top-level command handlers should decide to terminate the process

## Related

- T-464: GetExports exits process on paginator errors
- `lib/resources.go` has the same `log.Fatalln` pattern in `GetResources` (separate fix needed)
