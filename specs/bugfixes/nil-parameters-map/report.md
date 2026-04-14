# Bugfix Report: nil-parameters-map

**Date:** 2026-04-14
**Status:** Fixed

## Description of the Issue

`lib.GetParametersMap` panics with a nil pointer dereference when AWS CloudFormation returns a `types.Parameter` whose `ParameterKey` or `ParameterValue` field is nil. Both fields are `*string` pointers and AWS may leave them nil for redacted or unresolved values.

**Reproduction steps:**
1. Call `GetParametersMap` with a `[]types.Parameter` containing an entry where `ParameterKey` is nil.
2. Observe a `SIGSEGV` panic at `lib/stacks.go:868`.

**Impact:** Any drift or report path that passes stack parameters through `GetParametersMap` can crash the entire process when AWS returns nil fields.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panic in `GetParametersMap`.
- **Code inspected:** `lib/stacks.go` (lines 864-871), `cmd/drift.go:192` (sole caller).
- **Hypotheses tested:** Only one hypothesis needed — the pointer dereferences are unguarded.

## Discovered Root Cause

**Defect type:** Missing nil-pointer validation.

**Why it occurred:** The function assumed both `ParameterKey` and `ParameterValue` would always be non-nil, but the AWS SDK v2 `types.Parameter` struct uses `*string` pointers which can legitimately be nil.

**Contributing factors:** AWS documentation does not guarantee these fields are always populated. Responses with redacted SSM parameters or partially resolved values can contain nil fields.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:865-878` — Added nil guard: skip entries with nil key; map nil value to empty string.

**Approach rationale:** Skipping nil keys is the only safe option (no meaningful map key exists). Mapping nil values to empty string preserves the key in the result, which is the least surprising behaviour for callers that iterate over the map.

**Alternatives considered:**
- Return an error — rejected because every caller would need error handling for a rare edge case that has a safe default.
- Use a sentinel value like `"<nil>"` — rejected because empty string is more conventional and less likely to leak into user-facing output.

## Regression Test

**Test file:** `lib/stacks_test.go`
**Test names:** `TestGetParametersMap/nil_key_is_skipped`, `TestGetParametersMap/nil_value_maps_to_empty_string`, `TestGetParametersMap/nil_key_and_nil_value_is_skipped`

**What it verifies:** That nil `ParameterKey` entries are silently skipped, nil `ParameterValue` entries are stored as `""`, and fully-nil entries are skipped.

**Run command:** `go test ./lib/ -run TestGetParametersMap -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added nil guards to `GetParametersMap` |
| `lib/stacks_test.go` | Added three regression test cases for nil key/value handling |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When working with AWS SDK v2 types, always treat `*string` fields as potentially nil.
- Consider a project-wide linter rule or convention for dereferencing pointer fields from external SDK types.

## Related

- Transit ticket: T-717
