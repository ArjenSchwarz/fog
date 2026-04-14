# Bugfix Report: report-stdout-leak

**Date:** 2025-06-15
**Status:** Fixed

## Description of the Issue

The `report` command leaks raw stack names (ARN keys) directly to stdout during report rendering. A bare `fmt.Println(stackkey)` call in the stack iteration loop of `generateReport()` injects plain text lines into stdout for every stack being reported.

**Reproduction steps:**
1. Run `fog stack report` with any stack or set of stacks
2. Observe raw ARN strings printed to stdout before the rendered report output
3. When using `--output json`, the JSON output is interspersed with plain text lines

**Impact:** Medium — pollutes machine-readable output (JSON, CSV), causes unexpected lines in CLI/Lambda report output, and breaks downstream parsing of report output.

## Investigation Summary

- **Symptoms examined:** Extra plain-text lines appearing in stdout containing stack ARN keys
- **Code inspected:** `cmd/report.go`, specifically the `generateReport()` function's stack iteration loop
- **Hypotheses tested:** Confirmed the `fmt.Println(stackkey)` call is the sole source of the leak; no other direct stdout writes exist in the rendering path

## Discovered Root Cause

A debug `fmt.Println(stackkey)` call was left in the stack iteration loop inside `generateReport()` at `cmd/report.go:195`.

**Defect type:** Debug statement left in production code

**Why it occurred:** The `fmt.Println` was likely added during development for debugging purposes and was not removed before merging.

**Contributing factors:** The `generateReport()` function was difficult to unit test because it calls AWS directly, so the stdout leak was not caught by existing tests.

## Resolution for the Issue

**Changes made:**
- `cmd/report.go` — Extracted the stack iteration loop into a new `buildStackReports()` helper function and removed the `fmt.Println(stackkey)` call from it
- `cmd/report_stdout_leak_test.go` — Added regression test `TestBuildStackReports_NoRawStackNamesToStdout` that captures stdout and verifies no raw stack keys leak

**Approach rationale:** Extracting the loop into a helper function makes it directly testable without needing AWS mocks, while the fix itself is simply removing the debug print statement.

**Alternatives considered:**
- Gating the print behind a debug/verbose flag — unnecessary complexity for what is clearly a leftover debug statement
- Only removing the line without extracting — would leave the code path untestable

## Regression Test

**Test file:** `cmd/report_stdout_leak_test.go`
**Test name:** `TestBuildStackReports_NoRawStackNamesToStdout`

**What it verifies:** That iterating over multiple stacks and building report sections does not write raw stack keys to stdout. Uses stdout capture to detect any leaked lines.

**Run command:** `go test ./cmd/ -run TestBuildStackReports_NoRawStackNamesToStdout -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/report.go` | Extracted loop into `buildStackReports()`, removed `fmt.Println(stackkey)` |
| `cmd/report_stdout_leak_test.go` | New regression test file |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified the `fmt.Println(stackkey)` is the only direct stdout write in the rendering path

## Prevention

**Recommendations to avoid similar bugs:**
- Avoid using `fmt.Println` for debug output; use structured logging or stderr-based debug output instead
- Extract complex functions into testable helpers so stdout behaviour can be verified in tests
- Consider adding a CI check that flags bare `fmt.Print*` calls in non-test code

## Related

- Transit ticket: T-749
