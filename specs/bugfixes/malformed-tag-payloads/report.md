# Bugfix Report: malformed-tag-payloads

**Date:** 2025-07-14
**Status:** Fixed
**Transit:** T-695

## Description of the Issue

`getExpectedAndActualTags` in `cmd/drift.go` used unchecked type assertions on parsed drift properties. If AWS returned a malformed or non-standard tag structure (e.g. Tags as a map instead of a slice, non-map entries in the Tags slice, or non-string Key/Value fields), the function would panic, crashing the entire `fog stack drift` command.

**Reproduction steps:**
1. Call `getExpectedAndActualTags` with `map[string]any{"Tags": map[string]any{"Key":"k"}}` (Tags is a map, not a slice)
2. Observe panic from type assertion: `interface {} is map[string]interface {}, not []interface {}`

**Impact:** Drift command crashes on unexpected API payloads; users lose all drift results for the affected stack.

## Investigation Summary

- **Symptoms examined:** Panic on type assertion when Tags payload is non-standard
- **Code inspected:** `cmd/drift.go` lines 575-596, `getExpectedAndActualTags` function
- **Hypotheses tested:** All six type assertions in the function are unchecked and each can panic independently

## Discovered Root Cause

All type assertions in `getExpectedAndActualTags` use the single-value form (`x.(T)`) which panics on type mismatch, instead of the two-value comma-ok form (`v, ok := x.(T)`) which returns a zero value and `false`.

**Defect type:** Missing validation / unchecked type assertions

**Why it occurred:** The function was written assuming AWS always returns well-formed tag structures. This is true under normal operation but not guaranteed.

**Contributing factors:** Go's type assertion syntax makes it easy to forget the safe two-value form when the developer expects a known structure.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go:575-606` — Replaced unchecked type assertions with comma-ok patterns; extracted a helper `extractTagKeyValue` that safely returns (key, value, ok). Malformed entries are silently skipped.
- `cmd/drift_test.go` — Added `TestGetExpectedAndActualTags_MalformedPayloads` with 14 sub-tests covering all malformed variants.

**Approach rationale:** Using comma-ok assertions is the idiomatic Go approach for safely handling interface values. Extracting a helper keeps the main function readable and avoids repeating the same validation logic for expected and actual tags.

**Alternatives considered:**
- `recover()` wrapper — would mask the problem and make debugging harder; not chosen.
- Pre-validation with `reflect` — over-engineered for this use case; not chosen.

## Regression Test

**Test file:** `cmd/drift_test.go`
**Test name:** `TestGetExpectedAndActualTags_MalformedPayloads`

**What it verifies:** The function does not panic on any of 14 malformed tag payload variants, including non-slice Tags, non-map entries, non-string Key/Value, and missing fields. Also verifies that valid tags mixed with malformed ones are still correctly parsed.

**Run command:** `go test ./cmd -run TestGetExpectedAndActualTags_MalformedPayloads -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Replaced unchecked type assertions with safe comma-ok forms; added `extractTagKeyValue` helper |
| `cmd/drift_test.go` | Added 14 regression tests for malformed tag payloads |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- Always use the comma-ok form for type assertions on `any`/`interface{}` values from external data (API responses, parsed JSON)
- Consider a linter rule (e.g. `forcetypeassert`) to flag single-value type assertions

## Related

- Transit ticket: T-695
