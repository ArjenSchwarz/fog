# Bugfix Report: frontmatter-random-event-metadata

**Date:** 2025-03-30
**Status:** Fixed

## Description of the Issue

The `generateFrontMatter` function in `cmd/report.go` produced non-deterministic and incorrect YAML frontmatter when multiple stacks or multiple events were present. Each run could output different values for stack name, date, duration, event type, and success status.

**Reproduction steps:**
1. Run `fog report --frontmatter --output markdown` against multiple CloudFormation stacks
2. Run the same command again
3. Observe that frontmatter values (stack, date, eventtype, etc.) may differ between runs

**Impact:** Report frontmatter was unreliable for any automation or publishing pipeline relying on it. Values could describe a non-latest event even when `--latest` was set, creating a mismatch between frontmatter metadata and the report body.

## Investigation Summary

- **Symptoms examined:** Frontmatter fields (`stack`, `date`, `duration`, `eventtype`, `success`, `summary`) changed between runs with identical input
- **Code inspected:** `cmd/report.go` — `generateFrontMatter`, `generateStackReport`, `buildSimpleHTMLTable`
- **Hypotheses tested:** Confirmed Go map iteration non-determinism as root cause

## Discovered Root Cause

Two nested non-deterministic map iterations overwrote the same output keys on every pass:

1. **`generateFrontMatter`** iterated directly over `stacks` (a `map[string]lib.CfnStack`) and over all events per stack, writing to the same `result` map keys each time. Because Go map iteration order is randomised, whichever event was visited last determined the final frontmatter — producing different output on each run.

2. **`buildSimpleHTMLTable`** iterated over a `map[string]any` to build the HTML summary, so the `summary` frontmatter field also had non-deterministic column order.

Additionally, `generateFrontMatter` never checked `reportFlags.LatestOnly`, so frontmatter could describe an event not shown in the report body.

**Defect type:** Non-deterministic iteration over Go maps

**Why it occurred:** The original implementation used a simple nested loop assuming it would be called with a single stack and single event. When multiple stacks/events were present, the overwrite-last-wins pattern produced random results.

**Contributing factors:** The `--frontmatter` flag description says "Only works for single events" but the code didn't enforce or account for this.

## Resolution for the Issue

**Changes made:**
- `cmd/report.go:generateFrontMatter` — Rewrote to sort stacks by key (deterministic, matching report body), then select the single newest event (by `StartDate`) across all stacks. When `LatestOnly` is set, only the latest event per stack is considered. Frontmatter is populated once from this single selected event.
- `cmd/report.go:buildSimpleHTMLTable` — Added key sorting so the HTML summary table has deterministic column order.

**Approach rationale:** Selecting the newest event across all stacks ensures frontmatter describes the most recent activity, aligns with `LatestOnly` semantics, and is deterministic. Sorting stacks by key mirrors the approach used by the report body.

**Alternatives considered:**
- Using the first sorted stack's latest event only — rejected because it wouldn't reflect the most recent activity when multiple stacks are involved
- Returning an error for multi-stack/multi-event scenarios — rejected because this would break existing Lambda integration which sets `LatestOnly=true`

## Regression Test

**Test file:** `cmd/report_frontmatter_test.go`
**Test names:** `TestGenerateFrontMatter_DeterministicWithMultipleStacks`, `TestGenerateFrontMatter_SelectsLatestEventWithinStack`, `TestGenerateFrontMatter_RespectsLatestOnly`, `TestGenerateFrontMatter_MultipleStacksMultipleEvents`, `TestGenerateFrontMatter_EmptyStacks`

**What they verify:**
- Deterministic output across 20 iterations with multiple stacks (catches map iteration randomness)
- Correct selection of the latest event when a single stack has multiple events
- Correct filtering when `LatestOnly` flag is set
- Correct selection across multiple stacks each with multiple events
- Graceful handling of empty input

**Run command:** `go test ./cmd/ -run TestGenerateFrontMatter -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/report.go` | Fixed `generateFrontMatter` to select newest event deterministically; fixed `buildSimpleHTMLTable` key ordering |
| `cmd/report_frontmatter_test.go` | Added 5 regression tests |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When iterating Go maps for output, always sort keys first or document why order doesn't matter
- When frontmatter/metadata is derived from a collection, explicitly select a single item rather than overwriting in a loop
- Add determinism tests (run N iterations, compare results) when output depends on map iteration

## Related

- Transit ticket: T-590
