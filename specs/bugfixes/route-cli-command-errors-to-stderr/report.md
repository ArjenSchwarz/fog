# Bugfix Report: Route CLI command errors to stderr

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

CLI commands that fail through `failWithError` wrote diagnostics to stdout instead of stderr. That polluted structured stdout output such as JSON, CSV, or YAML and broke shell pipelines that expected stdout to contain only command results.

**Reproduction steps:**
1. Run a command that reaches `failWithError`, such as `fog drift --output json` with invalid inputs.
2. Trigger a command error.
3. Observe the error text emitted on stdout instead of stderr.

**Impact:** Structured CLI output became unreliable for failing commands that use `failWithError`, affecting automation and shell pipelines.

## Investigation Summary

The investigation focused on the shared command failure path and the existing stdout/stderr separation tests in the `cmd` package.

- **Symptoms examined:** Structured command output was contaminated by fatal error messages.
- **Code inspected:** `cmd/helpers.go`, `cmd/stream_separation_test.go`, and commands that call `failWithError`.
- **Hypotheses tested:** Whether the incorrect stream routing came from command-specific output builders or the shared fatal helper. Inspection confirmed the shared helper was writing directly with `fmt.Print`, which targets stdout.

## Discovered Root Cause

`failWithError` used `fmt.Print(output.StyleNegative(...))`, so every command path that called the helper wrote user-facing errors to stdout before exiting.

**Defect type:** Incorrect output stream selection

**Why it occurred:** The shared helper printed a formatted error message without directing it to `os.Stderr`, and there was no regression test covering the exiting error path.

**Contributing factors:** Multiple commands reused the helper, which amplified the impact of the incorrect default stream.

## Resolution for the Issue

**Changes made:**
- `cmd/helpers.go:66` - Changed `failWithError` to write formatted error output to `os.Stderr` before exiting.
- `cmd/helpers_test.go:1` - Added a regression test that executes the fatal helper in a subprocess and asserts stderr-only diagnostics.

**Approach rationale:** The bug came from a single shared helper using the wrong stream, so the safest fix was to correct that helper directly and lock the behavior in with a regression test.

**Alternatives considered:**
- Refactor commands to return errors instead of exiting from `failWithError` - not chosen because it would touch many call sites for a bug caused by one incorrect writer.

## Regression Test

**Test file:** `cmd/helpers_test.go`
**Test name:** `TestFailWithError_WritesToStderr`

**What it verifies:** The shared fatal error helper exits with status 1, leaves stdout empty, and writes the error message to stderr.

**Run command:** `go test ./cmd -run TestFailWithError_WritesToStderr`

## Affected Files

| File | Change |
|------|--------|
| `cmd/helpers.go` | Shared fatal error helper for several CLI commands |
| `cmd/helpers_test.go` | Regression test for fatal error stream routing |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the regression test failed before the fix because the error text was emitted on stdout.
- Reviewed the shared helper after the change to verify the error path now targets stderr directly.

## Prevention

**Recommendations to avoid similar bugs:**
- Add regression tests around shared stdout/stderr routing for command exit paths.
- Prefer explicit writers (`os.Stdout` / `os.Stderr`) in shared CLI helpers.

## Related

- Transit ticket `T-1014`
