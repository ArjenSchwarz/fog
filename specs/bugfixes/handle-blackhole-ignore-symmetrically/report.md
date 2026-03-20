# Bugfix Report: Handle Blackhole Ignore Symmetrically

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`CompareRoutes` only ignores blackhole routes when `route1` is blackhole. If `route2` is blackhole (and its VpcPeeringConnectionId is in the ignore list) while `route1` is active, the comparison returns `false` instead of treating the routes as equivalent.

**Reproduction steps:**
1. Configure `drift.ignore-blackholes` with a VPC peering connection ID
2. Run drift detection on a route table where the CloudFormation template has a blackhole route for that peering connection but the actual AWS route is active (or vice versa depending on argument order)
3. Observe that drift is reported even though the blackhole should be ignored

**Impact:** Drift detection reports false positives for blackhole routes when the blackhole state appears on route2 instead of route1.

## Investigation Summary

The bug is in the state comparison block of `CompareRoutes` in `lib/ec2.go`.

- **Symptoms examined:** Asymmetric blackhole ignore behavior
- **Code inspected:** `lib/ec2.go:CompareRoutes`, `cmd/drift.go` (call site)
- **Hypotheses tested:** Only one hypothesis needed — the blackhole check on line 146 only inspects route1

## Discovered Root Cause

The blackhole ignore logic on line 146 of `lib/ec2.go` only checks `route1.State` and `route1.VpcPeeringConnectionId`. When the states differ and route2 is the blackhole (not route1), the function falls through to return `false`.

**Defect type:** Logic error — asymmetric condition check

**Why it occurred:** The original implementation only considered the scenario where route1 (the AWS actual route) is blackhole. The reverse case was not handled.

**Contributing factors:** The calling code in `cmd/drift.go` passes the AWS route as route1 and the CloudFormation route as route2, so the original author likely only considered the case where the live AWS route becomes blackhole. However, a comparison function should handle both argument orderings.

## Resolution for the Issue

**Changes made:**
- `lib/ec2.go:144-150` - Added symmetric blackhole ignore check: when states differ, check if either route is blackhole with a VpcPeeringConnectionId in the ignore list

**Approach rationale:** A helper function `isIgnoredBlackhole` extracts the repeated check logic and is applied to both routes. This keeps the code DRY and makes the symmetry explicit.

**Alternatives considered:**
- Duplicating the if-condition for route2 inline — rejected because it creates repetition and is harder to read

## Regression Test

**Test file:** `lib/ec2_test.go`
**Test name:** `TestCompareRoutes_BlackholeIgnoreSymmetric`

**What it verifies:** That blackhole ignore works regardless of whether route1 or route2 is the blackhole route. Tests seven scenarios including both orderings, non-ignored peering IDs, both-blackhole, nil peering ID, and empty ignore list.

**Run command:** `go test ./lib/ -run TestCompareRoutes_BlackholeIgnoreSymmetric -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/ec2.go` | Made blackhole ignore check symmetric in `CompareRoutes` |
| `lib/ec2_test.go` | Added `TestCompareRoutes_BlackholeIgnoreSymmetric` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When writing comparison functions, verify that ignore/skip logic works symmetrically for both arguments
- Test comparison functions with arguments in both orderings

## Related

- Transit ticket: T-410
