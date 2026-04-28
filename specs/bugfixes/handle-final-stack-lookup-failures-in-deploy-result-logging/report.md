# Bugfix Report: Handle final stack lookup failures in deploy result logging

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

`printDeploymentResults` fetched the final stack state after a deployment and treated lookup errors as fatal process exits. When that happened, the command stopped before writing a final `SUCCESS` or `FAILED` deployment log entry.

**Reproduction steps:**
1. Execute a deployment that reaches `printDeploymentResults`.
2. Make `getFreshStackFunc` return an error while fetching the post-deploy stack state.
3. Observe the helper print an error and terminate before `DeploymentLog.Failed()` runs.

**Impact:** Deployment history could miss the final status for a failed deployment-result path, making logs incomplete and violating the helper's documented behaviour.

## Investigation Summary

The inspection focused on the post-deploy result handling path and how deployment logs are finalized.

- **Symptoms examined:** abrupt termination on final stack lookup failure, missing deployment log status, missing regression coverage for the lookup-error path
- **Code inspected:** `cmd/deploy_helpers.go`, `cmd/deploy_helpers_test.go`, `cmd/deploy_integration_test.go`, `lib/logging.go`, `cmd/deploy_output.go`
- **Hypotheses tested:** whether the helper wrote a failure log before exiting, whether failure output could be rendered without a final stack state, and whether existing tests exercised the error branch

## Discovered Root Cause

The `getFreshStackFunc` error branch in `printDeploymentResults` called `log.Fatalln`, which terminated the process immediately after printing an error. Because that branch never called `DeploymentLog.Failed()`, the deployment log was left without a final status.

**Defect type:** Error handling / control-flow error

**Why it occurred:** The helper had explicit success and rollback handling, but the final stack lookup failure path bypassed the same logging lifecycle and used process termination instead of handled failure reporting.

**Contributing factors:** Existing tests only covered successful final stack retrieval, so the fatal-exit path was not protected by regression coverage.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy_helpers.go` - replace the fatal exit on final stack lookup failure with handled failure reporting: set `DeploymentError`, record a failed deployment log entry, emit failure output, and return.
- `cmd/deploy_helpers_test.go` - add a subprocess regression test that proves the lookup-error path no longer exits abruptly and now records a failed deployment log entry.

**Approach rationale:** The fix stays local to deployment result handling and aligns the lookup-error branch with the existing pattern used for other deployment failures.

**Alternatives considered:**
- Return an error from `printDeploymentResults` and handle it in the caller - not chosen because it would require broader signature and call-site changes for a localized bug.
- Keep an exit after writing the log - not chosen because this path is now handled like other deployment-result failures and no longer needs abrupt termination.

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test name:** `TestPrintDeploymentResults_HandlesFinalStackLookupFailure`

**What it verifies:** A final stack lookup error is handled without a fatal exit, records a failed deployment log entry, and preserves the deployment error on the result object.

**Run command:** `go test ./cmd -run TestPrintDeploymentResults_HandlesFinalStackLookupFailure -count=1`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy_helpers.go` | Handle final stack lookup errors as deployment-result failures instead of fatal exits |
| `cmd/deploy_helpers_test.go` | Add regression coverage for the `getFreshStackFunc` error path |
| `specs/bugfixes/handle-final-stack-lookup-failures-in-deploy-result-logging/report.md` | Document investigation, root cause, fix, and verification |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified the regression via a subprocess test that reproduces the old fatal-exit behaviour safely and confirms the helper now returns with a failed deployment log status.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat deploy-result edge cases the same way as normal deployment failures so the deployment log is always finalized.
- Add focused regression tests for all non-success branches in lifecycle helpers that are documented to always emit a final status.
- Prefer injectable exit hooks or returned errors over direct fatal logging in code paths that still need cleanup or auditing.

## Related

- Transit ticket `T-1003`
