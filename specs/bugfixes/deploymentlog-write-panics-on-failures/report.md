# Bugfix Report: DeploymentLog.Write Panics on Failures

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`DeploymentLog.Write()` uses `panic()` on JSON marshal failure and `log.Fatal()` on file write errors. Both of these terminate the CLI process instead of allowing callers to handle the error gracefully. Since `Write()` is called by `Success()` and `Failed()`, which are invoked during deployment result processing, a log write issue crashes the CLI mid-deployment.

**Reproduction steps:**
1. Enable deployment logging in fog configuration
2. Configure an invalid or unwritable log file path
3. Run a deployment — observe the CLI crashes with `log.Fatal` instead of reporting the write error

**Impact:** Medium severity. Any file system issue (permissions, disk full, invalid path) during log writing crashes the entire CLI, preventing proper deployment result reporting.

## Investigation Summary

Systematic inspection of `lib/logging.go` revealed two defects in `DeploymentLog.Write()`:

- **Symptoms examined:** `panic()` on line 103 and `log.Fatal()` on line 108
- **Code inspected:** `lib/logging.go` (Write, Success, Failed methods), `cmd/deploy_helpers.go` (callers)
- **Hypotheses tested:** Confirmed both `Success()` and `Failed()` blindly call `Write()` with no error handling

## Discovered Root Cause

`DeploymentLog.Write()` uses fatal error handling patterns (panic and log.Fatal) instead of returning errors.

**Defect type:** Incorrect error handling — using process-terminating calls instead of error propagation

**Why it occurred:** The original implementation treated log write failures as unrecoverable. In a CLI tool, log writing is ancillary to the main deployment workflow and should not crash the process.

**Contributing factors:** `Write()` has a `void` return signature, making it impossible for callers to handle errors. `Success()` and `Failed()` also return nothing, propagating the same problem up the call chain.

## Resolution for the Issue

**Changes made:**
- `lib/logging.go:98-111` - Changed `Write()` to return `error`; replaced `panic()` with `fmt.Errorf()` and `log.Fatal()` with error return
- `lib/logging.go:142-145` - Changed `Success()` to return `error`, propagating from `Write()`
- `lib/logging.go:148-152` - Changed `Failed()` to return `error`, propagating from `Write()`
- `cmd/deploy_helpers.go:181,199` - Updated callers to handle returned errors with stderr warnings
- `lib/logging_test.go` - Updated existing tests and added regression tests for error return behavior

**Approach rationale:** Converting panic/fatal to error returns is the standard Go pattern. Callers log warnings to stderr so deployment results are still reported even if log writing fails.

**Alternatives considered:**
- Silent failure (swallow errors in Write) — rejected because callers should know logging failed
- Retry logic — rejected as overly complex for a logging side-effect

## Regression Test

**Test file:** `lib/logging_test.go`
**Test names:** `TestDeploymentLog_WriteReturnsErrorOnFileWriteFailure`, `TestDeploymentLog_WriteReturnsErrorOnLoggingDisabled`, `TestDeploymentLog_SuccessReturnsErrorOnWriteFailure`, `TestDeploymentLog_FailedReturnsErrorOnWriteFailure`

**What it verifies:** That `Write()`, `Success()`, and `Failed()` return errors instead of panicking or exiting when log writing fails, and that status/failure fields are still set even when the write fails.

**Run command:** `go test ./lib/ -run "TestDeploymentLog_WriteReturnsError|TestDeploymentLog_SuccessReturnsError|TestDeploymentLog_FailedReturnsError" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/logging.go` | Changed `Write()`, `Success()`, `Failed()` to return `error` |
| `lib/logging_test.go` | Added regression tests, updated existing tests |
| `cmd/deploy_helpers.go` | Updated callers to handle errors with stderr warnings |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Avoid `panic()` and `log.Fatal()` in library code; reserve for truly unrecoverable situations in `main()`
- Functions that perform I/O should always return errors
- Consider adding a linter rule to flag `log.Fatal` usage in non-main packages

## Related

- Transit ticket: T-466
