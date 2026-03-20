# Bugfix Report: Unmanaged Drift Detection Using Map Values

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `checkIfResourcesAreManaged` function in `cmd/drift.go` incorrectly determines whether AWS resources are managed by a CloudFormation stack. It uses `stringValueInMap(resource, logicalToPhysical)`, which checks if the resource identifier matches any **value** (physical ID) in the `logicalToPhysical` map. It should instead check if the resource identifier exists as a **key** (logical ID) in the map.

**Reproduction steps:**
1. Have a CloudFormation stack with resources where logical IDs differ from physical IDs (which is nearly always the case)
2. Run drift detection with `detect-unmanaged-resources` configured
3. Observe that managed resources are falsely reported as UNMANAGED

**Impact:** Managed resources are incorrectly flagged as UNMANAGED in drift detection output, producing false positives that undermine the reliability of drift reports.

## Investigation Summary

- **Symptoms examined:** Resources managed by CloudFormation are reported as UNMANAGED in drift output
- **Code inspected:** `cmd/drift.go` (`checkIfResourcesAreManaged`), `cmd/helpers.go` (`stringValueInMap`), `lib/identitycenter.go` (resource listing)
- **Hypotheses tested:** Confirmed that `stringValueInMap` iterates over map values (physical IDs) and compares them against the resource identifier, which should be compared against map keys (logical IDs)

## Discovered Root Cause

`stringValueInMap` iterates over map **values** with `for _, b := range list` and checks `b == a`. When called as `stringValueInMap(resource, logicalToPhysical)`, it checks if `resource` equals any physical resource ID. Since `resource` is a logical identifier from `allresources`, the comparison only succeeds when a physical ID coincidentally equals a logical ID.

**Defect type:** Logic error — wrong map dimension (values vs keys) used for lookup

**Why it occurred:** The helper function `stringValueInMap` was designed to check values, but the call site needed a key existence check. The names are similar enough that the distinction was easy to miss.

**Contributing factors:** No unit tests existed for `checkIfResourcesAreManaged` to catch this mismatch.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go:294` - Replaced `stringValueInMap(resource, logicalToPhysical)` with direct map key lookup `_, exists := logicalToPhysical[resource]`
- `cmd/helpers.go:61-68` - Removed the now-unused `stringValueInMap` function

**Approach rationale:** A direct map key lookup is both correct (checks keys instead of values) and more efficient (O(1) vs O(n)). Since `stringValueInMap` has no other callers, removing it eliminates dead code.

**Alternatives considered:**
- Creating a new `stringKeyInMap` helper function — unnecessary; Go's native map lookup is cleaner and more idiomatic

## Regression Test

**Test file:** `cmd/drift_managed_test.go`
**Test name:** `TestCheckIfResourcesAreManaged_KeyLookup`

**What it verifies:** That `checkIfResourcesAreManaged` uses map key lookup (logical IDs) to determine if resources are managed, not map value lookup (physical IDs). Covers managed resources, unmanaged resources, physical-ID-only matches, mixed scenarios, ignore lists, and empty inputs.

**Run command:** `go test ./cmd/ -run TestCheckIfResourcesAreManaged_KeyLookup -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Replace `stringValueInMap` call with direct map key lookup |
| `cmd/helpers.go` | Remove unused `stringValueInMap` function |
| `cmd/drift_managed_test.go` | Add regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer Go's native map key lookup (`_, ok := m[key]`) over custom helper functions for key existence checks
- Add unit tests for functions that perform map lookups to verify the correct dimension (keys vs values) is checked
- Consider naming helper functions to explicitly indicate what they check (e.g., `valueExistsInMap` vs `keyExistsInMap`)

## Related

- Transit ticket: T-435
