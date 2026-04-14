# Bugfix Report: precheck-abort-skips-failure-log

**Date:** 2026-04-14
**Status:** Fixed
**Transit Ticket:** T-684

## Description of the Issue

When `runPrechecks` returns `abort=true` (either due to a precheck execution error or a failed precheck with `stop-on-failed-prechecks` enabled), the `deployTemplate` function calls `os.Exit(1)` immediately without first calling `deploymentLog.Failed(nil)`. This means the deployment failure status is never written to the deployment log file.

**Reproduction steps:**
1. Configure a precheck command that fails (e.g., `sh -c 'exit 1'`) with `templates.stop-on-failed-prechecks: true` and logging enabled
2. Run `fog stack deploy`
3. Observe that the prechecks fail and the deployment aborts
4. Check the deployment log file — no failure entry is recorded

**Impact:** Failed deploy runs terminate without writing the expected deployment failure status/log metadata, making it impossible to audit or track failed deployments caused by precheck failures.

## Investigation Summary

- **Symptoms examined:** The abort path at `cmd/deploy.go:93-95` exits before the failure log write at `cmd/deploy.go:97-101`
- **Code inspected:** `cmd/deploy.go` (deployTemplate), `cmd/deploy_helpers.go` (runPrechecks), `lib/logging.go` (DeploymentLog.Failed)
- **Hypotheses tested:** Confirmed the dead code block at lines 97-101 was unreachable in all abort scenarios

## Discovered Root Cause

**Defect type:** Logic error — unreachable code after early return

**Why it occurred:** Commit b151395 (T-567) added an `abort` return value to `runPrechecks`. The abort check at line 93 now captures both scenarios that the old code at lines 97-101 handled (failed prechecks with stop flag, and execution errors). However, the abort path calls `os.Exit(1)` without calling `deploymentLog.Failed(nil)` first, while the old (now dead) code block at lines 97-101 did call it.

**Contributing factors:** The old failure-log code block was left in place after the abort signal was introduced, giving the appearance that it was still handling the failure log write. In reality, it was dead code.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy.go:93-98` — Added `deploymentLog.Failed(nil)` call before `osExitFunc(1)` in the abort path, with proper error handling
- `cmd/deploy.go:97-101` — Removed dead code block that was unreachable after the abort check
- `cmd/deploy_helpers.go:30` — Added `osExitFunc` stub variable (defaults to `os.Exit`) to enable testing of exit paths
- `cmd/deploy.go:94,97` — Changed `os.Exit(1)` to `osExitFunc(1)` in the abort path for testability

**Approach rationale:** The fix adds the missing `deploymentLog.Failed(nil)` call to the abort path and removes the dead code. The error from `Failed` is handled with a warning to stderr (consistent with the pattern used in `printDeploymentResults`). The `osExitFunc` stub follows the existing pattern of function variables used for testing throughout the deploy module.

**Alternatives considered:**
- Restructuring `deployTemplate` to use error returns instead of `os.Exit` — too invasive for a bugfix, better suited for a separate refactor
- Moving the `Failed` call into `runPrechecks` — incorrect separation of concerns; `runPrechecks` shouldn't be responsible for deployment log finalization

## Regression Test

**Test file:** `cmd/deploy_precheck_abort_test.go`
**Test names:**
- `TestDeployTemplate_PrecheckAbortWritesFailureLog/failed_precheck_with_stop_flag_writes_failure_log`
- `TestDeployTemplate_PrecheckAbortWritesFailureLog/execution_error_writes_failure_log`
- `TestDeployTemplate_PrecheckAbortWritesFailureLog/execution_error_with_stop_flag_writes_failure_log`
- `TestDeployTemplate_SuccessfulPrecheckNoAbort` (verifies successful prechecks are not affected)

**What it verifies:** When the precheck abort path is taken, the deployment log file contains a FAILED status entry with precheck status set to FAILED.

**Run command:** `go test ./cmd/... -v -run "TestDeployTemplate_PrecheckAbort"`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy.go` | Added `deploymentLog.Failed(nil)` before exit in abort path; removed dead code; use `osExitFunc` |
| `cmd/deploy_helpers.go` | Added `osExitFunc` stub variable for testability |
| `cmd/deploy_precheck_abort_test.go` | New regression tests for the abort path |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding early-exit paths, audit all finalization code that should run before exit
- Consider refactoring `deployTemplate` to return errors instead of calling `os.Exit` directly, which would make all exit paths more testable
- Dead code blocks after control-flow changes should be identified and removed during review

## Related

- T-567: Original commit (b151395) that introduced the abort signal from `runPrechecks`
- T-577: Related fix for execution errors in prechecks
