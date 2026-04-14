# Bugfix Report: prechecks-stop-flag-regression

**Date:** 2026-04-14
**Status:** Fixed
**Transit Ticket:** T-711

## Description of the Issue

`runPrechecks` in `cmd/deploy_helpers.go` always aborts the deployment when `lib.RunPrechecks` returns an execution or configuration error (e.g. missing command, unsafe command), regardless of the `templates.stop-on-failed-prechecks` setting.

**Reproduction steps:**
1. Configure a precheck command that fails to execute (e.g., a non-existent binary).
2. Set `templates.stop-on-failed-prechecks=false`.
3. Run `fog deploy`.
4. Deployment exits early instead of continuing after recording the precheck failure.

**Impact:** Any deployment with a misconfigured precheck command is blocked even when the user has explicitly opted to continue on precheck failures. This is a regression from T-577.

## Investigation Summary

- **Symptoms examined:** `runPrechecks` returns `abort=true` for execution errors even when stop flag is disabled.
- **Code inspected:** `cmd/deploy_helpers.go` — the `runPrechecks` function, specifically the `if err != nil` branch.
- **Hypotheses tested:** Compared the error branch with the non-error failure branch (`if info.PrechecksFailed`), which correctly returns `stopOnFailed`.

## Discovered Root Cause

**Defect type:** Logic error — hardcoded return value

In `cmd/deploy_helpers.go`, the `if err != nil` branch of `runPrechecks` returns `(builder.String(), true)` unconditionally. The comment says "the stop flag is honored" but the code doesn't actually check the flag.

**Why it occurred:** When T-577 added execution-error handling, the error path was written to always abort (defensive coding) but should have mirrored the non-error failure path which checks `viper.GetBool("templates.stop-on-failed-prechecks")`.

**Contributing factors:** The existing test case `"execution error missing command marks failed"` encoded the buggy behavior by asserting `wantAbort: true` with `stopOnFailedPrecheck: false`, masking the regression.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy_helpers.go:~109` — Changed `return builder.String(), true` to `return builder.String(), viper.GetBool("templates.stop-on-failed-prechecks")`
- `cmd/deploy_helpers_test.go` — Fixed existing test case to expect `wantAbort: false` when stop flag is disabled; added dedicated regression test `TestRunPrechecks_T711_ExecutionErrorHonorsStopFlag`

**Approach rationale:** The fix mirrors the pattern already used in the non-error failure path, ensuring consistent behavior regardless of whether the precheck failed during execution or produced a non-zero exit code.

**Alternatives considered:**
- Returning a separate error type to let `deployTemplate` decide — over-engineering for a boolean flag check.

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test name:** `TestRunPrechecks_T711_ExecutionErrorHonorsStopFlag`

**What it verifies:** That execution errors (missing command, unsafe command) in prechecks respect the stop-on-failed-prechecks flag — returning `abort=false` when the flag is disabled and `abort=true` when enabled.

**Run command:** `go test ./cmd/ -run TestRunPrechecks_T711 -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy_helpers.go` | Fix hardcoded `true` return to respect stop flag |
| `cmd/deploy_helpers_test.go` | Fix existing test + add regression test for T-711 |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When adding error-handling branches, ensure they follow the same control-flow patterns as existing branches for the same function.
- Test cases for error paths should include both stop-flag states (enabled and disabled).

## Related

- T-577: Original precheck execution error handling
- `specs/bugfixes/precheck-execution-errors/report.md`: Related previous bugfix
