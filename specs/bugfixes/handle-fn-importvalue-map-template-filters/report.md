# Bugfix Report: Handle Fn::ImportValue map for NetworkAclId/RouteTableId in template filters

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`FilterNaclEntriesByLogicalId`, `FilterRoutesByLogicalId`, and `FilterTGWRoutesByLogicalId` panic when a resource property (`NetworkAclId`, `RouteTableId`, or `TransitGatewayRouteTableId`) uses `Fn::ImportValue` instead of a `Ref` or literal string.

**Reproduction steps:**
1. Have a CloudFormation template where `NetworkAclId`, `RouteTableId`, or `TransitGatewayRouteTableId` uses `!ImportValue` (e.g., `NetworkAclId: !ImportValue SharedNaclExport`).
2. Run `fog drift` on a stack using such a template.
3. Observe panic from `.(string)` type assertion on a `map[string]any` value.

**Impact:** Drift detection crashes for any stack that uses cross-stack references via `Fn::ImportValue` for NACLs, route tables, or TGW route tables.

## Investigation Summary

- **Symptoms examined:** Panic on type assertion `.(string)` when `Fn::ImportValue` produces a `map[string]any`.
- **Code inspected:** `lib/template.go` (filter functions and `stringPointer`), `lib/tgw_routetables.go`, `cmd/drift.go`.
- **Hypotheses tested:** Whether the `customImportValueHandler` returns a map instead of a string, and whether the filter functions handle that map type.

## Discovered Root Cause

The `customImportValueHandler` in `lib/template.go` returns `map[string]any{"Fn::ImportValue": input}` for `Fn::ImportValue` intrinsics. Three filter functions then perform unsafe `.(string)` type assertions on the affected property:

1. `FilterNaclEntriesByLogicalId` in `lib/template.go`: `resource.Properties["NetworkAclId"].(string)`
2. `FilterRoutesByLogicalId` in `lib/template.go`: `resource.Properties["RouteTableId"].(string)`
3. `FilterTGWRoutesByLogicalId` in `lib/tgw_routetables.go`: `resource.Properties["TransitGatewayRouteTableId"].(string)`

The `stringPointer` function already handles the `map[string]any` case for both `Ref` and `Fn::ImportValue`, but these filter functions don't use it.

**Defect type:** Missing type handling / unsafe type assertion

**Why it occurred:** The filter functions were written assuming these properties would always resolve to strings (via `Ref` or literals). `Fn::ImportValue` was added later with its own handler that preserves the map structure, but the filter functions were not updated.

**Contributing factors:** No tests covered `Fn::ImportValue` in the filter functions; only `stringPointer` had `Fn::ImportValue` handling.

Additionally, `FilterNaclEntriesByLogicalId` does not receive the `logicalToPhysical` map, so it cannot resolve import values even if the type assertion were safe. The other two functions already receive this map but don't use it for the ID extraction.

## Resolution for the Issue

**Changes made:**
- `lib/template.go` — Added `resourceIdMatchesLogical` helper function that handles both string (`"REF: LogicalName"`) and map (`{"Fn::ImportValue": "ExportName"}`) property values. For strings, it strips the `REF: ` prefix and compares directly. For `Fn::ImportValue` maps, it resolves both the import name and the logical ID through `logicalToPhysical` and compares their physical IDs. Updated `FilterNaclEntriesByLogicalId` to accept `logicalToPhysical` and use the new helper. Updated `FilterRoutesByLogicalId` to use the new helper.
- `lib/tgw_routetables.go` — Updated `FilterTGWRoutesByLogicalId` to use the new `resourceIdMatchesLogical` helper.
- `cmd/drift.go` — Updated `checkNaclEntries` function signature and call site to pass `logicalToPhysical`.
- `lib/template_test.go` — Added three regression tests and updated existing `TestFilterNaclEntriesByLogicalId` call site.

**Approach rationale:** A single shared helper function centralises the type-switching logic, preventing future drift between the three filter functions. Resolving `Fn::ImportValue` through `logicalToPhysical` (which already contains CloudFormation exports from `separateSpecialCases`) correctly maps imported physical IDs to the logical IDs being filtered.

**Alternatives considered:**
- Inline type switches in each filter function — rejected because it would duplicate logic across three locations.
- Converting `Fn::ImportValue` maps to strings during template parsing — rejected because it would lose information needed by `stringPointer` for other properties.

## Regression Test

**Test file:** `lib/template_test.go`
**Test names:** `TestFilterNaclEntriesByLogicalId_FnImportValue`, `TestFilterRoutesByLogicalId_FnImportValue`, `TestFilterTGWRoutesByLogicalId_FnImportValue`

**What they verify:** Filter functions do not panic when `NetworkAclId`, `RouteTableId`, or `TransitGatewayRouteTableId` is an `Fn::ImportValue` map, and correctly match resources by resolving the import value through the `logicalToPhysical` map.

**Run command:** `go test ./lib -run "TestFilter.*FnImportValue" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/template.go` | Add helper to extract resource ID from string or map; update filter function signatures and logic |
| `lib/tgw_routetables.go` | Update `FilterTGWRoutesByLogicalId` to handle `Fn::ImportValue` map |
| `cmd/drift.go` | Update `FilterNaclEntriesByLogicalId` call to pass `logicalToPhysical` |
| `lib/template_test.go` | Add regression tests for `Fn::ImportValue` in all three filter functions |

## Verification

**Automated:**
- [x] Regression tests pass (`go test ./lib -run "TestFilter.*FnImportValue" -v`)
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Related

- Transit ticket: T-539
