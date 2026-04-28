# Bugfix Report: Drift Structured Error Handling

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

The `stack drift` command still used `log.Fatal` in two AWS failure paths:
unmanaged-resource discovery and stack resource loading for special-case drift
checks. Those exits bypassed the command's normal `failWithError` formatter.

**Reproduction steps:**
1. Run the drift command with an AWS failure while loading stack resources.
2. Run the drift command with an AWS failure while listing unmanaged resources.
3. Observe raw `log.Fatal` output and immediate exit instead of formatted command
   errors and debug panic handling.

**Impact:** Drift failures were inconsistent with other commands, bypassed the
normal formatted error flow, and were harder to verify in tests.

## Investigation Summary

The investigation focused on the drift command's error paths and existing test
coverage around special-case resource handling and unmanaged-resource reporting.

- **Symptoms examined:** raw fatal log output, skipped `failWithError`, and
  missing debug panic behaviour.
- **Code inspected:** `cmd/drift.go`, `cmd/helpers.go`,
  `cmd/drift_specialcases_test.go`, and `cmd/drift_unmanaged_test.go`.
- **Hypotheses tested:** whether the remaining `log.Fatal` calls were confined
  to helper paths and whether those helpers could return errors cleanly for the
  command to handle.

## Discovered Root Cause

The drift command delegated two AWS lookups to code paths that still hard-exited
with `log.Fatal` instead of returning errors to the main command flow.

**Defect type:** Inconsistent error handling

**Why it occurred:** Earlier drift logic adopted `failWithError`, but the
special-case stack-resource lookup and unmanaged-resource listing paths were not
updated to follow the same pattern.

**Contributing factors:** The helper code returned data directly instead of
returning errors, which made the `log.Fatal` exits harder to test.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go` - Returned errors from `separateSpecialCases`, added
  `detectUnmanagedResources`, and routed both AWS failure paths back through
  `failWithError`.
- `cmd/helpers.go` - Switched `failWithError` to use the shared `osExitFunc`
  hook so command-exit paths remain testable.
- `cmd/drift_specialcases_test.go` - Added a regression test for
  `DescribeStackResources` failures.
- `cmd/drift_unmanaged_test.go` - Added a regression test for unmanaged
  resource lookup failures.

**Approach rationale:** Returning errors from the helper paths keeps the drift
command aligned with the rest of the CLI and lets the existing command-level
error handler control formatting, exit behaviour, and debug panics.

**Alternatives considered:**
- Keep `log.Fatal` and use subprocess tests - rejected because it preserves the
  inconsistent command behaviour instead of fixing it.

## Regression Test

**Test file:** `cmd/drift_specialcases_test.go`, `cmd/drift_unmanaged_test.go`
**Test name:** `TestSeparateSpecialCasesReturnsDescribeStackResourcesError`,
`TestDetectUnmanagedResourcesReturnsListAllResourcesError`

**What it verifies:** AWS lookup failures return errors for the command-level
handler instead of hard-exiting via `log.Fatal`.

**Run command:** `go test ./cmd -run 'TestSeparateSpecialCasesReturnsDescribeStackResourcesError|TestDetectUnmanagedResourcesReturnsListAllResourcesError' -count=1`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Returned helper errors and centralized unmanaged-resource error handling |
| `cmd/helpers.go` | Reused the package exit hook from `failWithError` |
| `cmd/drift_specialcases_test.go` | Added regression coverage for stack resource load failure |
| `cmd/drift_unmanaged_test.go` | Added regression coverage for unmanaged-resource lookup failure |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Not run

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer returning errors from helper functions and let command entrypoints call
  `failWithError`.
- Add focused regression tests whenever command helpers previously terminated
  the process directly.
- Reuse testable exit hooks for command-level failures instead of direct fatal
  logging.

## Related

- Transit ticket `T-1015`
