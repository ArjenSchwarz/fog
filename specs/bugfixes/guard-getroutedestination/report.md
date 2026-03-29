# Bugfix Report: Guard GetRouteDestination Against Nil Destination Fields

**Date:** 2026-03-29
**Status:** Fixed
**Transit:** T-639

## Description of the Issue

`lib.GetRouteDestination` panics with a nil pointer dereference when a `types.Route` has none of its destination fields (`DestinationCidrBlock`, `DestinationPrefixListId`, `DestinationIpv6CidrBlock`) set. The `default` branch of the switch statement unconditionally dereferences `route.DestinationIpv6CidrBlock` without checking for nil.

**Reproduction steps:**
1. Construct or receive a `types.Route` with all destination pointer fields left nil (e.g., a partially-resolved CloudFormation template route).
2. Call `GetRouteDestination(route)`.
3. Observe: panic — `runtime error: invalid memory address or nil pointer dereference`.

**Impact:** Any drift detection or report flow that encounters a route missing destination properties crashes instead of returning a recoverable value. This aborts the entire command.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panic in `GetRouteDestination` at `ec2.go:178`.
- **Code inspected:** `lib/ec2.go` (GetRouteDestination, GetRouteTarget), `lib/template.go` (FilterRoutesByLogicalId), `cmd/drift.go` (two callers).
- **Hypotheses tested:** Confirmed that all three destination fields are `*string` pointers; confirmed that `GetRouteTarget` (same file) already handles the all-nil case safely by returning an empty string via its default zero-value; only `GetRouteDestination` has the unsafe dereference.

## Discovered Root Cause

**Defect type:** Missing nil guard

The `switch` statement in `GetRouteDestination` checks `DestinationCidrBlock` and `DestinationPrefixListId` for nil before dereferencing but falls through to a `default` case that unconditionally dereferences `DestinationIpv6CidrBlock`. When all three fields are nil, the default case triggers a nil pointer dereference.

**Why it occurred:** The function was written assuming every route has at least one destination field set. This is true for well-formed AWS API responses but not guaranteed for routes constructed from CloudFormation templates with missing or unresolved properties.

**Contributing factors:** The sibling function `GetRouteTarget` handles the all-nil case safely (it returns the zero-value empty string), so the pattern inconsistency was easy to overlook.

## Resolution for the Issue

_Filled in after fix is implemented._

## Regression Test

**Test file:** `lib/ec2_test.go`
**Test names:** `TestGetRouteDestinationNilFields`, `TestGetRouteDestinationOnlyIPv6Nil`

**What they verify:** Calling `GetRouteDestination` with a route where all destination fields are nil returns an empty string and does not panic.

**Run command:** `go test ./lib -run "TestGetRouteDestinationNilFields|TestGetRouteDestinationOnlyIPv6Nil" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/ec2.go` | Add nil guard in `GetRouteDestination` default branch |
| `lib/ec2_test.go` | Add regression tests for nil destination fields |
| `lib/template.go` | Skip routes with empty destination key in `FilterRoutesByLogicalId` |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When switching on pointer fields, always add a nil check before dereferencing in the default/fallback branch.
- Use the `GetRouteTarget` pattern (which naturally returns a zero-value) as the reference for similar functions.

## Related

- Transit ticket: T-639
