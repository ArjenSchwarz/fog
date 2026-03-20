# Bugfix Report: Unmanaged Resource Detection in Drift Check

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `checkIfResourcesAreManaged` function in the drift command used `stringValueInMap` to determine whether a resource was managed by CloudFormation. This function performed an O(n) linear scan over map values for each resource, resulting in O(n*m) overall complexity and a fragile matching approach.

**Reproduction steps:**
1. Run `fog stack drift` with `drift.detect-unmanaged-resources` configured for resource types.
2. Observe that unmanaged resource detection relies on `stringValueInMap`, which scans map values linearly.

**Impact:** Inefficient O(n*m) matching for unmanaged resource detection, with potential for false positives/negatives if physical ID formats differ between AWS API responses and CloudFormation's physical resource IDs.

## Investigation Summary

- **Symptoms examined:** `checkIfResourcesAreManaged` uses `stringValueInMap` to check if a physical resource ID exists as a value in the `logicalToPhysical` map.
- **Code inspected:** `cmd/drift.go`, `cmd/helpers.go`, `lib/identitycenter.go`, `lib/drift.go`.
- **Data flow traced:** `logicalToPhysical` maps logical IDs (keys) to physical IDs (values). `allresources` from `ListAllResources` maps physical identifiers (keys) to resource types (values). The check needs to match physical IDs from `allresources` against physical IDs stored as values in `logicalToPhysical`.

## Discovered Root Cause

`stringValueInMap` iterates over all map values with a linear scan (`for _, b := range list`) to find a match. While functionally checking the correct dimension (physical IDs against physical IDs), this approach has two problems:

1. O(n) per lookup instead of O(1) map key lookup, making the overall check O(n*m)
2. No proper reverse index — the code relies on a helper that scans values instead of building a set for direct membership testing

**Defect type:** Performance and correctness defect (inefficient value-scan instead of set-based lookup).

**Why it occurred:**
- Why was `stringValueInMap` used? Because the resource identifier needed to be checked against map values (physical IDs), not keys (logical IDs).
- Why wasn't a reverse index built? The helper was a quick convenience function that traded correctness guarantees for simplicity.
- Why was this not caught? No unit tests existed for `checkIfResourcesAreManaged`.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go` - Replaced `stringValueInMap` call with a pre-built reverse lookup set (`managedPhysicalIDs`) that provides O(1) key-based lookups. The set is constructed by iterating over `logicalToPhysical` values once before the resource loop.
- `cmd/helpers.go` - Removed the now-unused `stringValueInMap` function.
- `cmd/drift_unmanaged_test.go` - Added regression tests covering managed resources, unmanaged resources, mixed scenarios, ignore lists, and empty inputs.

**Approach rationale:** Building a set of physical IDs from `logicalToPhysical` values converts O(n) value scans to O(1) map key lookups. The one-time cost of building the set is O(n), making the overall check O(n+m) instead of O(n*m).

**Alternatives considered:**
- Keep `stringValueInMap` but add tests — rejected because the linear scan is inherently inefficient and the approach is semantically unclear.
- Build a full `physicalToLogical` reverse map — rejected because only existence is needed, not the reverse mapping; a `map[string]struct{}` set is simpler and more memory-efficient.

## Regression Test

**Test file:** `cmd/drift_unmanaged_test.go`
**Test names:**
- `TestCheckIfResourcesAreManaged_CorrectlyIdentifiesManagedResources`
- `TestCheckIfResourcesAreManaged_AllManaged`
- `TestCheckIfResourcesAreManaged_NoneManaged`
- `TestCheckIfResourcesAreManaged_IgnoreList`
- `TestCheckIfResourcesAreManaged_EmptyInputs`

**What they verify:** Correct identification of managed vs unmanaged resources using set-based physical ID matching, including edge cases for ignore lists and empty inputs.

**Run command:** `go test ./cmd -run 'TestCheckIfResourcesAreManaged'`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Replaced `stringValueInMap` with pre-built `managedPhysicalIDs` set lookup |
| `cmd/helpers.go` | Removed unused `stringValueInMap` function |
| `cmd/drift_unmanaged_test.go` | Added regression tests for `checkIfResourcesAreManaged` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint, go fmt)

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer set-based lookups over linear scans when checking membership in map values.
- Add unit tests for utility functions that perform data matching, especially when they bridge two different data representations.
- When a function checks map values rather than keys, consider whether a reverse index should be built upfront.

## Related

- Transit ticket: `T-455`
