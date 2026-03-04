# Bugfix Report: Drift Detection Crashes on AWS API Errors

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

The drift command crashed with a panic stack trace whenever CloudFormation returned an AWS SDK error during drift detection operations.

**Reproduction steps:**
1. Run `fog stack drift` against a stack where CloudFormation drift APIs return an error.
2. Trigger an error from `DetectStackDrift`, `DescribeStackDriftDetectionStatus`, or `DescribeStackResourceDrifts`.
3. Observe an unhandled panic and stack trace output.

**Impact:** The command failed noisily and bypassed normal CLI error formatting, resulting in poor UX and harder debugging in automation.

## Investigation Summary

A systematic inspection of drift flow found direct panic usage in `lib/drift.go` and unchecked call sites in `cmd/drift.go`.

- **Symptoms examined:** panic stack traces from drift command on AWS API failures.
- **Code inspected:** `lib/drift.go`, `cmd/drift.go`, `lib/drift_test.go`.
- **Hypotheses tested:** whether panics originated in command handling (rejected) versus lower-level drift helpers (confirmed).

## Discovered Root Cause

Drift helper functions used `panic(err)` when AWS SDK calls failed instead of propagating errors to callers.

**Defect type:** Error handling defect (unhandled exceptions/panics).

**Why it occurred:**
- Why did the command crash? Because helper functions panicked.
- Why did panics escape? Because `detectDrift` did not receive errors for these calls.
- Why were errors not returned? Helper signatures returned only values, not `(value, error)`.
- Why was this not caught? Existing tests explicitly expected panic behavior.

**Contributing factors:** Recursive polling function reused panic pattern, reinforcing crash behavior across multiple API paths.

## Resolution for the Issue

**Changes made:**
- `lib/drift.go` - Updated `StartDriftDetection`, `WaitForDriftDetectionToFinish`, and `GetDefaultStackDrift` to return errors instead of panicking; wrapped AWS SDK errors with context.
- `cmd/drift.go` - Handled returned errors from all three drift helper calls and routed them through `failWithError`.
- `lib/drift_test.go` - Updated error-path tests to assert returned errors and nil/zero results instead of panic assertions.

**Approach rationale:** Returning errors preserves normal CLI control flow and enables consistent user-facing error output through existing command-level handlers.

**Alternatives considered:**
- Add panic recovery in command layer - rejected because it hides root behavior and keeps helpers unsafe for reuse.

## Regression Test

**Test file:** `lib/drift_test.go`
**Test name:**
- `TestStartDriftDetection` (case: `API error returns error`)
- `TestWaitForDriftDetectionToFinish` (case: `API error returns error`)
- `TestGetDefaultStackDrift` (case: `API error returns error`)

**What it verifies:** AWS API failures in drift helpers are returned as errors instead of causing panics.

**Run command:** `go test ./lib -run 'Test(StartDriftDetection|WaitForDriftDetectionToFinish|GetDefaultStackDrift)'`

## Affected Files

| File | Change |
|------|--------|
| `lib/drift.go` | Replaced panic paths with error returns and contextual wrapping |
| `cmd/drift.go` | Added error handling for drift helper calls via `failWithError` |
| `lib/drift_test.go` | Converted panic expectations to error assertions |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified command code path now checks and forwards helper errors to `failWithError`.

## Prevention

**Recommendations to avoid similar bugs:**
- Return errors from library code and reserve panics for unrecoverable programmer errors.
- Add/maintain explicit error-path tests for AWS SDK interactions.
- Prefer command-layer error presentation (`failWithError`) for user-facing failures.

## Related

- Transit ticket: `T-339`
