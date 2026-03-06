# Bugfix Report: Handle Ref/import map for TGW route table id in FilterTGWRoutesByLogicalId

**Date:** 2026-03-06
**Status:** Fixed

## Description of the Issue

`FilterTGWRoutesByLogicalId` panicked when a Transit Gateway route resource's `TransitGatewayRouteTableId` property was specified as a `Ref` map (`{"Ref": "LogicalId"}`) or `Fn::ImportValue` map (`{"Fn::ImportValue": "ExportName"}`), instead of the preprocessed `"REF: LogicalId"` string format.

**Reproduction steps:**
1. Define a CloudFormation template with a TGW route whose `TransitGatewayRouteTableId` uses `!Ref` or `!ImportValue`.
2. Run drift detection on that stack.
3. Observe panic: `interface conversion: interface {} is map[string]interface {}, not string`.

**Impact:** Drift detection crashes when templates contain TGW routes that reference route table IDs via intrinsic functions rather than the preprocessed string format.

## Investigation Summary

- **Symptoms examined:** Panic at line 139 of `tgw_routetables.go` due to type assertion `.(string)` on a `map[string]any` value.
- **Code inspected:** `lib/tgw_routetables.go` (`FilterTGWRoutesByLogicalId`), `lib/tgw_routetables.go` (`extractStringProperty`), `cmd/drift.go` (call site).
- **Hypotheses tested:** The `TransitGatewayRouteTableId` property can arrive in multiple formats depending on template preprocessing, and the function only handled one format.

## Discovered Root Cause

`FilterTGWRoutesByLogicalId` unconditionally type-asserted `TransitGatewayRouteTableId` to `string`, but the property can also be a `map[string]any` when it contains a `Ref` or `Fn::ImportValue` intrinsic function.

**Defect type:** Missing type handling / unsafe type assertion

**Why it occurred:** The function was written to only handle the preprocessed `"REF: "` string format. Other property formats (raw Ref maps, ImportValue maps) were not considered.

**Contributing factors:** The related `extractStringProperty` function already handles all formats correctly for other properties in the same resource, but was not reused for the route table ID matching logic.

## Resolution for the Issue

**Changes made:**
- `lib/tgw_routetables.go` - Extracted route table matching into a new `tgwRouteMatchesRouteTable` helper that handles all property formats: `"REF: "` strings, `{"Ref": ...}` maps, `{"Fn::ImportValue": ...}` maps, and plain physical ID strings.
- `lib/template_test.go` - Added regression test `TestFilterTGWRoutesByLogicalId_RefAndImportMap` covering Ref map, ImportValue map, and REF: string formats.

**Approach rationale:** A dedicated matching function handles all the type variations in one place, with a type switch that mirrors how `extractStringProperty` works. The physical ID lookup enables matching ImportValue references that resolve to the same route table.

**Alternatives considered:**
- Reuse `extractStringProperty` directly and compare resolved values - Rejected because the semantics differ: the route table ID needs to match the logical ID (not resolve to a physical ID for use elsewhere), and ImportValue requires physical-to-physical comparison.
- Only handle the Ref map case - Rejected because ImportValue is also a valid format that would still cause panics.

## Regression Test

**Test file:** `lib/template_test.go`
**Test name:** `TestFilterTGWRoutesByLogicalId_RefAndImportMap`

**What it verifies:** Routes with `TransitGatewayRouteTableId` specified as `{"Ref": "LogicalId"}`, `{"Fn::ImportValue": "ExportName"}`, and `"REF: LogicalId"` are all correctly matched. Routes referencing different route tables are excluded.

**Run command:** `go test ./lib -run TestFilterTGWRoutesByLogicalId_RefAndImportMap -count=1`

## Affected Files

| File | Change |
|------|--------|
| `lib/tgw_routetables.go` | Replaced unsafe type assertion with `tgwRouteMatchesRouteTable` helper handling all property formats. |
| `lib/template_test.go` | Added regression test for Ref map and ImportValue map route table ID formats. |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

**Manual verification:**
- Confirmed the new test panics on pre-fix code and passes after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- When accessing CloudFormation template properties, always use a type switch or the existing `extractStringProperty` helper instead of direct type assertions.
- The analogous `FilterRoutesByLogicalId` in `template.go` has the same pattern and may need a similar fix.

## Related

- Transit ticket: T-365
