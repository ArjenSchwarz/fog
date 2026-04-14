# Bugfix Report: Guard CompareNaclEntries Against Nil Nested EC2 Pointers

**Date:** 2026-04-14
**Status:** Fixed
**Ticket:** T-731

## Description of the Issue

`CompareNaclEntries` in `lib/ec2.go` dereferenced nested pointer fields (`IcmpTypeCode.Code`, `IcmpTypeCode.Type`, `PortRange.From`, `PortRange.To`) without nil checks. While the parent structs (`IcmpTypeCode`, `PortRange`) were guarded, their inner `*int32` fields were not.

**Reproduction steps:**
1. Call `CompareNaclEntries` with two entries where both have a non-nil `IcmpTypeCode` but one has a nil `Code` or `Type` field
2. Observe runtime panic due to nil pointer dereference

**Impact:** A malformed or partial AWS response (or mocked fixture) with nil inner fields would crash drift detection with a runtime panic instead of returning a safe comparison result.

## Investigation Summary

- **Symptoms examined:** Potential nil pointer dereference in nested EC2 struct fields
- **Code inspected:** `lib/ec2.go` lines 73-80 (IcmpTypeCode) and lines 88-95 (PortRange)
- **Hypotheses tested:** The existing `stringPointerValueMatch` helper safely handles nil `*string` pointers; the same pattern was missing for `*int32` fields

## Discovered Root Cause

The function guarded `IcmpTypeCode` and `PortRange` containers for nil but assumed their inner pointer fields (`Code`, `Type`, `From`, `To`) were always non-nil when the container was present.

**Defect type:** Missing nil validation on nested pointer fields

**Why it occurred:** The AWS SDK EC2 types use `*int32` for these fields, meaning they can independently be nil even when the parent struct is present. The original code only checked the parent.

**Contributing factors:** AWS SDK v2 uses pointer types extensively; partial API responses or test fixtures may omit inner fields.

## Resolution for the Issue

**Changes made:**
- `lib/ec2.go` - Added `int32PointerValueMatch` helper (mirrors `stringPointerValueMatch`) for safe `*int32` comparison
- `lib/ec2.go` - Replaced raw `*` dereferences in `IcmpTypeCode.Code/Type` and `PortRange.From/To` comparisons with `int32PointerValueMatch` calls

**Approach rationale:** Follows the existing `stringPointerValueMatch` pattern already used throughout the file, keeping the code consistent and easy to understand.

**Alternatives considered:**
- Inline nil checks before each dereference — rejected because it duplicates logic and is less readable than the helper pattern already established

## Regression Test

**Test file:** `lib/ec2_test.go`
**Test names:** `TestCompareNaclEntries/*nil*` and `Test_int32PointerValueMatch`

**What it verifies:**
- Nil `IcmpTypeCode.Code` vs non-nil returns false (no panic)
- Nil `IcmpTypeCode.Type` vs non-nil returns false (no panic)
- Both nil inner ICMP fields match each other
- Nil `PortRange.From` vs non-nil returns false (no panic)
- Nil `PortRange.To` vs non-nil returns false (no panic)
- Both nil inner PortRange fields match each other
- `int32PointerValueMatch` handles all nil/non-nil combinations

**Run command:** `go test ./lib/ -run 'TestCompareNaclEntries|Test_int32PointerValueMatch' -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/ec2.go` | Added `int32PointerValueMatch` helper; used it in `CompareNaclEntries` for safe nested pointer comparison |
| `lib/ec2_test.go` | Added 8 regression test cases for nil nested pointers and 5 unit tests for `int32PointerValueMatch` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Code formatted (`go fmt`)

## Prevention

**Recommendations to avoid similar bugs:**
- When comparing AWS SDK struct fields that use pointer types, always use nil-safe helper functions rather than direct dereference
- Consider using generics for pointer comparison helpers to avoid creating type-specific variants (future improvement)

## Related

- Transit ticket: T-731
