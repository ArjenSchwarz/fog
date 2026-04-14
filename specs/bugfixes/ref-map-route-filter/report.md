# Bugfix Report: ref-map-route-filter

**Date:** 2026-04-14
**Status:** Fixed
**Transit:** T-741

## Description of the Issue

`FilterRoutesByLogicalId` (and `FilterNaclEntriesByLogicalId`) failed to match resources whose `RouteTableId` or `NetworkAclId` was expressed as a `{"Ref": "LogicalId"}` map. This meant that when CloudFormation templates contained raw-map Ref values instead of the stringified `"REF: LogicalId"` format, routes and NACL entries were silently skipped during drift comparisons.

**Reproduction steps:**
1. Provide a CloudFormation template where a Route resource has `"RouteTableId": {"Ref": "MyRouteTable"}`
2. Call `FilterRoutesByLogicalId("MyRouteTable", ...)` with that template
3. Observe that the route is not included in the results despite the matching logical ID

**Impact:** Drift comparisons could report incomplete results, missing valid routes/NACLs attached to a route table when templates use the Ref-map format.

## Investigation Summary

- **Symptoms examined:** `FilterRoutesByLogicalId` returns fewer routes than expected when `RouteTableId` uses `{"Ref": "..."}` format
- **Code inspected:** `lib/template.go:resourceIdMatchesLogical`, `lib/tgw_routetables.go:tgwRouteMatchesRouteTable`
- **Hypotheses tested:** The `tgwRouteMatchesRouteTable` function already handles `{"Ref": "..."}` correctly (lines 182-187), confirming this was a known format that `resourceIdMatchesLogical` was missing

## Discovered Root Cause

`resourceIdMatchesLogical` only handled two property formats in its type switch:
1. `string` — matching `"REF: LogicalId"` after prefix stripping
2. `map[string]any` — matching only `{"Fn::ImportValue": "..."}` maps

The `{"Ref": "LogicalId"}` map format was never checked in the `map[string]any` branch.

**Defect type:** Missing case in type switch — incomplete format handling

**Why it occurred:** The function was originally written for the stringified `"REF: "` format. When `Fn::ImportValue` support was added, only that specific map key was handled. The `{"Ref": "..."}` map format (used when templates bypass stringification) was overlooked.

**Contributing factors:** The TGW equivalent (`tgwRouteMatchesRouteTable`) was written later and does handle all formats, but the fix was not back-ported to the shared `resourceIdMatchesLogical` helper.

## Resolution for the Issue

**Changes made:**
- `lib/template.go:334-337` — Added `{"Ref": "LogicalId"}` handling in the `map[string]any` branch of `resourceIdMatchesLogical`, checked before the existing `Fn::ImportValue` case
- `lib/template.go:324-328` — Updated function doc comment to document the three supported formats

**Approach rationale:** The fix mirrors the existing pattern in `tgwRouteMatchesRouteTable` (lines 182-187) and is the minimal change needed. By fixing `resourceIdMatchesLogical` rather than the callers, both `FilterRoutesByLogicalId` and `FilterNaclEntriesByLogicalId` are fixed simultaneously.

**Alternatives considered:**
- Refactoring `tgwRouteMatchesRouteTable` to use `resourceIdMatchesLogical` — not chosen because `tgwRouteMatchesRouteTable` also handles a plain physical ID string comparison that doesn't apply to the general case

## Regression Test

**Test file:** `lib/template_test.go`
**Test names:**
- `TestFilterRoutesByLogicalId_RefMap` — end-to-end test with Ref-map routes
- `TestResourceIdMatchesLogical_RefMap` — unit test covering all format variants
- `TestFilterNaclEntriesByLogicalId_RefMap` — ensures NACL filtering also works with Ref-maps

**What they verify:** Routes and NACL entries with `{"Ref": "LogicalId"}` format are correctly matched by their respective filter functions, while non-matching Ref-map entries are excluded.

**Run command:** `go test ./lib/ -run "TestFilterRoutesByLogicalId_RefMap|TestResourceIdMatchesLogical_RefMap|TestFilterNaclEntriesByLogicalId_RefMap" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/template.go` | Added `{"Ref": "..."}` handling in `resourceIdMatchesLogical` |
| `lib/template_test.go` | Added 3 regression tests for Ref-map format |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters/validators pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When adding a new property format to one matching function, audit other functions that perform similar matching (e.g., `resourceIdMatchesLogical` vs `tgwRouteMatchesRouteTable`) to ensure consistency
- Consider consolidating format-matching logic into a single shared helper to avoid divergence

## Related

- T-741: FilterRoutesByLogicalId misses RouteTableId values in Ref-map format
- `lib/tgw_routetables.go:tgwRouteMatchesRouteTable` — reference implementation that already handled all formats
