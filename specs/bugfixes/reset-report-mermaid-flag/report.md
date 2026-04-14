# Bugfix Report: Reset Report Mermaid Flag

**Date:** 2025-07-14
**Status:** Fixed

## Description of the Issue

The `reportFlags.HasMermaid` global flag in `cmd/report.go` was only set to `true` when the output format was markdown or HTML, but was never reset to `false` for other formats (JSON, CSV, text). This caused Mermaid Gantt chart data to leak into non-Mermaid output formats in long-lived processes.

**Reproduction steps:**
1. Run `generateReport()` with `output=markdown` — sets `HasMermaid = true`
2. Run `generateReport()` again with `output=json` — `HasMermaid` remains `true`
3. The JSON report unexpectedly includes Gantt chart data

**Impact:** In long-lived processes such as Lambda, one markdown/HTML invocation leaves `HasMermaid` stuck as `true`, causing all subsequent non-Mermaid reports to include unwanted Gantt chart content.

## Investigation Summary

- **Symptoms examined:** Global flag `reportFlags.HasMermaid` sticks as `true` across invocations
- **Code inspected:** `cmd/report.go` — `generateReport()` and `generateStackReport()`
- **Hypotheses tested:** The conditional assignment `if hasMermaid { reportFlags.HasMermaid = true }` never has a corresponding `else` branch to reset the flag

## Discovered Root Cause

The `generateReport` function computed `hasMermaid` correctly but only assigned the flag inside a one-sided `if`:

```go
if hasMermaid {
    reportFlags.HasMermaid = true
}
```

This never resets the flag to `false` when `hasMermaid` is `false`.

**Defect type:** Logic error — missing else branch on global state mutation

**Why it occurred:** The flag was treated as a "set once" value rather than a per-invocation state.

**Contributing factors:** Global mutable state (`reportFlags`) shared across invocations in Lambda.

## Resolution for the Issue

**Changes made:**
- `cmd/report.go:126` — Changed conditional `if hasMermaid { reportFlags.HasMermaid = true }` to unconditional `reportFlags.HasMermaid = outputFormat == outputFormatMarkdown || outputFormat == outputFormatHTML`

**Approach rationale:** Directly assigning the computed boolean ensures the flag always reflects the current invocation's output format, regardless of prior state.

**Alternatives considered:**
- Adding an explicit `else { reportFlags.HasMermaid = false }` — functionally equivalent but less idiomatic Go

## Regression Test

**Test file:** `cmd/report_mermaid_flag_test.go`
**Test names:** `TestHasMermaid_ResetForNonMermaidOutput`, `TestGenerateStackReport_NoGanttForNonMermaidOutput`

**What it verifies:**
1. The `HasMermaid` flag is correctly reset to `false` when the output format changes from markdown to JSON
2. `generateStackReport` respects the `HasMermaid` flag — Gantt charts appear only when `true`

**Run command:** `go test ./cmd -run 'TestHasMermaid_ResetForNonMermaidOutput|TestGenerateStackReport_NoGanttForNonMermaidOutput' -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/report.go` | Changed conditional flag set to unconditional assignment |
| `cmd/report_mermaid_flag_test.go` | Added regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Code formatted with gofmt

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer unconditional assignment (`flag = expr`) over conditional set (`if expr { flag = true }`) for boolean flags derived from a computed value
- Consider resetting all per-invocation state at the start of `generateReport` to avoid stale global state in long-lived processes

## Related

- Transit ticket: T-796
