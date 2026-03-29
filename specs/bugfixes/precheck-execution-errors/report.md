# Bugfix Report: precheck-execution-errors

**Date:** 2026-03-29
**Status:** Fixed
**Ticket:** T-577

## Description of the Issue

When `lib.RunPrechecks` returned an error (e.g. command not found, unsafe command, unbalanced quotes), the `runPrechecks` helper in `cmd/deploy_helpers.go` only appended the error message to the output string and returned early. It did not set `info.PrechecksFailed` or `logObj.PreChecks`, so the deployment log never recorded the failure and the `stop-on-failed-prechecks` flag was never evaluated. The deployment then always proceeded to changeset creation.

**Reproduction steps:**
1. Configure `templates.prechecks` with a nonexistent command (e.g. `nonexistent-cmd $TEMPLATEPATH`)
2. Set `templates.stop-on-failed-prechecks: true`
3. Run `fog stack deploy`
4. Observe that deployment proceeds to changeset creation despite the precheck error

**Impact:** Medium — deployments could proceed past precheck validation failures, undermining the safety net that prechecks are designed to provide.

## Investigation Summary

- **Symptoms examined:** `runPrechecks` error path at `cmd/deploy_helpers.go:102-106` returns without marking failure
- **Code inspected:** `cmd/deploy_helpers.go` (runPrechecks), `cmd/deploy.go` (deployTemplate flow), `lib/files.go` (RunPrechecks)
- **Hypotheses tested:** The error path was written to handle the error display but the failure bookkeeping (`PrechecksFailed`, `logObj.PreChecks`) was missing from that branch

## Discovered Root Cause

The `runPrechecks` function had two distinct failure paths: (1) when `lib.RunPrechecks` returns a non-nil error (configuration/execution errors like missing commands), and (2) when individual prechecks fail at runtime (non-zero exit code). Only path (2) correctly set `info.PrechecksFailed = true` and `logObj.PreChecks = FAILED`. Path (1) was missing this bookkeeping.

Additionally, `deployTemplate` in `deploy.go` had no gate between `runPrechecks` and `createAndShowChangeset` — it never checked whether prechecks failed with `stop-on-failed-prechecks` enabled.

**Defect type:** Missing error handling — incomplete failure bookkeeping on an error branch

**Why it occurred:** The error path was likely added later (for safety checks like unsafe commands) without updating it to match the failure bookkeeping of the runtime-failure path.

**Contributing factors:** The two failure paths (config error vs runtime failure) were structurally separate, making it easy to miss that both needed the same side effects.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy_helpers.go:102-115` — In the `err != nil` branch, set `info.PrechecksFailed = true` and `logObj.PreChecks = DeploymentLogPreChecksFailed`, and show the appropriate stop/continue message based on the `stop-on-failed-prechecks` flag
- `cmd/deploy.go:93-96` — Added a check after `runPrechecks`: if `deployment.PrechecksFailed` and `stop-on-failed-prechecks` is true, exit before changeset creation

**Approach rationale:** This is the minimal, most localized fix — it brings the error path in line with the existing runtime-failure path and adds the missing gate in the deploy flow.

**Alternatives considered:**
- Refactoring `RunPrechecks` to never return errors and always use `PrechecksFailed` — rejected because the distinction between config errors and runtime failures is useful at the library level

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test names:** `TestRunPrechecks/execution_error_missing_command_marks_failed`, `TestRunPrechecks/execution_error_missing_command_honors_stop_flag`, `TestRunPrechecks/execution_error_unsafe_command_marks_failed`

**What they verify:** That when `lib.RunPrechecks` returns an error (missing command, unsafe command), `PrechecksFailed` is set to true and `logObj.PreChecks` is set to `FAILED`.

**Run command:** `go test ./cmd/ -run TestRunPrechecks -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy_helpers.go` | Set `PrechecksFailed` and log status on execution errors |
| `cmd/deploy.go` | Stop deployment when prechecks failed and stop flag is set |
| `cmd/deploy_helpers_test.go` | Added 3 regression test cases for execution errors |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding new error paths, ensure all side effects (flags, log status) match existing failure paths
- Consider extracting the failure bookkeeping into a helper to ensure consistency across branches

## Related

- T-577: Treat precheck execution errors as failed prechecks and honor stop flag
