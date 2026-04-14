# Bugfix Report: deploy-changeset-execute-failure

**Date:** 2026-04-14
**Status:** Fixed
**Ticket:** T-675

## Description of the Issue

When `ExecuteChangeSet` fails (e.g., due to permissions errors or API failures), the `deployChangeset` function prints an error message but continues executing into the event polling loop and final status handling. This produces misleading output and can result in a false-success command exit.

**Reproduction steps:**
1. Configure a deployment where `ExecuteChangeSet` will fail (e.g., insufficient permissions)
2. Run `fog deploy` with the configuration
3. Observe that fog prints "Could not execute changeset!" but continues polling for events

**Impact:** Medium — users see misleading output after a deployment failure, and the command may exit with status 0 despite the changeset execution failing.

## Investigation Summary

- **Symptoms examined:** `deployChangeset` prints error but continues into event polling loop
- **Code inspected:** `cmd/deploy.go` (deployChangeset), `cmd/deploy_helpers.go` (confirmAndDeployChangeset), caller in deploy command
- **Hypotheses tested:** Confirmed that the error from `DeployChangeset()` is logged but not propagated; control flow continues unconditionally

## Discovered Root Cause

In `deployChangeset()`, the error returned by `deployment.Changeset.DeployChangeset()` is only printed to stderr — it is never returned or used to short-circuit execution. The function has no return value, so callers cannot detect the failure.

**Defect type:** Missing error propagation

**Why it occurred:** The original implementation treated the execute-changeset call as fire-and-forget, relying on downstream stack-status polling to detect failure. However, when the API call itself fails (before any stack operation begins), polling never observes a failure state.

**Contributing factors:** The function signature `func deployChangeset(...) ` (no error return) made it impossible for callers to handle the failure.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy.go:446` — Changed `deployChangeset` to return `error`; return immediately on `ExecuteChangeSet` failure instead of continuing into the event loop
- `cmd/deploy_helpers.go:162` — Changed `confirmAndDeployChangeset` to return `(bool, error)` and propagate deployment errors
- `cmd/deploy.go:120` — Updated caller to handle error: prints error, cleans up new stacks, and exits non-zero

**Approach rationale:** Propagating the error through the call chain follows Go conventions and gives the caller full control over error handling. The command now exits non-zero on execute failure, ensuring CI/CD pipelines detect the problem.

**Alternatives considered:**
- Calling `os.Exit(1)` directly inside `deployChangeset` — rejected because it bypasses cleanup and makes testing difficult
- Logging and continuing with a flag — rejected because downstream polling cannot detect pre-execution API failures

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test name:** `TestConfirmAndDeployChangeset/deploy_fails_with_error`

**What it verifies:** When `deployChangesetFunc` returns an error, `confirmAndDeployChangeset` returns `false` and propagates the error. The changeset is not deleted (since it was never executed).

**Run command:** `go test ./cmd/ -run TestConfirmAndDeployChangeset -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy.go` | `deployChangeset` returns `error`, early return on failure; caller handles error with non-zero exit |
| `cmd/deploy_helpers.go` | `confirmAndDeployChangeset` returns `(bool, error)`, propagates deploy error |
| `cmd/deploy_helpers_test.go` | Updated existing tests for new signatures, added error propagation test case |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Build succeeds (`go build -o fog`)
- `golangci-lint run` reports 0 issues

## Prevention

**Recommendations to avoid similar bugs:**
- Functions that call AWS APIs should always return errors rather than printing and continuing
- Use `error` return values consistently for operations that can fail
- Consider adding a linter rule or code review checklist item for "error printed but not returned"

## Related

- Transit ticket T-675
