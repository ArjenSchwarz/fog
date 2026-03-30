# Bugfix Report: Stop Deployment on Failed Prechecks

**Date:** 2026-03-30
**Status:** Fixed
**Ticket:** T-567

## Description of the Issue

When `templates.stop-on-failed-prechecks` was set to `true` and prechecks failed, fog displayed the correct "stopping deployment" message but proceeded to create and deploy the changeset anyway. The configuration flag only controlled the wording of the output message, not the actual deployment flow.

**Reproduction steps:**
1. Configure `templates.prechecks` with a command that fails (e.g., `sh -c 'exit 1'`)
2. Set `templates.stop-on-failed-prechecks: true` in fog.yaml
3. Run `fog deploy` — observe that the "stopping deployment" message appears but deployment continues

**Impact:** Users relying on prechecks as a safety gate (e.g., running `cfn-lint`) would still deploy invalid templates, potentially causing CloudFormation failures or deploying misconfigured resources.

## Investigation Summary

- **Symptoms examined:** The `runPrechecks` function displayed different messages based on `stop-on-failed-prechecks` but returned only a display string. The caller (`deployTemplate`) had no mechanism to act on the precheck outcome.
- **Code inspected:** `cmd/deploy.go` (lines 89-94), `cmd/deploy_helpers.go` (`runPrechecks` function), `lib/files.go` (`RunPrechecks`)
- **Hypotheses tested:** Confirmed that `deployment.PrechecksFailed` was set correctly by `lib.RunPrechecks`, and that `viper.GetBool("templates.stop-on-failed-prechecks")` was read correctly — but neither value influenced the deployment flow.

## Discovered Root Cause

**Defect type:** Missing control flow — the function `runPrechecks` returned only a display string with no signal for the caller to abort deployment.

**Why it occurred:** The `runPrechecks` function was designed to collect output for display. The `stop-on-failed-prechecks` config was read inside it purely to choose between two message variants. No abort signal was propagated back to `deployTemplate`, which unconditionally proceeded to `createAndShowChangeset`.

**Contributing factors:** The `deployTemplate` function treats `runPrechecks` as a side-effect-only call (display output), with no early-return path for precheck failures.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy_helpers.go:93-125` — Changed `runPrechecks` return type from `string` to `(string, bool)`. The bool indicates whether the deployment should abort (true when prechecks fail AND `stop-on-failed-prechecks` is enabled).
- `cmd/deploy.go:89-96` — Updated caller to receive the abort signal and exit with code 1 when abort is true, preventing changeset creation.

**Approach rationale:** Returning an abort signal from `runPrechecks` is the minimal, testable change. The function already inspects both `PrechecksFailed` and `stop-on-failed-prechecks` to choose messages, so it is the natural place to determine whether to abort.

**Alternatives considered:**
- Adding a check in `deployTemplate` using `deployment.PrechecksFailed && viper.GetBool(...)` directly — rejected because it would duplicate the config-reading logic already in `runPrechecks` and make the abort decision less testable.
- Having `runPrechecks` return an error — rejected because a failed precheck is not an error in the Go sense; it's a deliberate abort based on configuration.

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test name:** `TestRunPrechecks/failed_precheck_stop`

**What it verifies:** When prechecks fail and `stop-on-failed-prechecks` is `true`, `runPrechecks` returns `abort=true`. All other scenarios (no prechecks, successful prechecks, failed but continue) return `abort=false`.

**Integration test file:** `cmd/deploy_integration_test.go`
**Test name:** `TestDeploymentWorkflow_WithPrechecks/failed_prechecks_stop_deployment`

**What it verifies:** End-to-end flow: when the abort signal is true, changeset creation is skipped entirely.

**Run command:** `go test ./cmd/... -run TestRunPrechecks -v` and `INTEGRATION=1 go test -tags integration ./cmd/... -run TestDeploymentWorkflow_WithPrechecks -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy_helpers.go` | Changed `runPrechecks` to return `(string, bool)` with abort signal |
| `cmd/deploy.go` | Added abort check after prechecks, preventing changeset creation |
| `cmd/deploy_helpers_test.go` | Added `wantAbort` field to existing test cases |
| `cmd/deploy_integration_test.go` | Enhanced to verify changeset creation is skipped on abort; fixed pre-existing missing argument |
| `cmd/drift_integration_test.go` | Removed pre-existing unused import blocking integration test compilation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Integration tests pass (`INTEGRATION=1 go test -tags integration ./cmd/... -run TestDeploymentWorkflow_WithPrechecks`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When a function reads configuration to decide behaviour, ensure the decision is propagated to callers — not just used for display.
- Integration tests for deployment flow should verify that downstream steps (changeset creation, deployment) are actually skipped, not just that status flags are set.

## Related

- T-567: Stop deployment when prechecks fail with stop-on-failed-prechecks enabled
