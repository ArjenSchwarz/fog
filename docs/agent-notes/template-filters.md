# Template Filter Functions

## Key Abstraction: resourceIdMatchesLogical

`resourceIdMatchesLogical` in `lib/template.go` is a shared helper used by `FilterNaclEntriesByLogicalId` and `FilterRoutesByLogicalId` to match a resource property value against a logical resource ID. `FilterTGWRoutesByLogicalId` uses a separate matcher, `tgwRouteMatchesRouteTable` in `lib/tgw_routetables.go`, which covers the same property formats for TGW route tables.

It handles four forms of property values:
- **String** (`"REF: LogicalName"`): strips the `REF: ` prefix and compares directly against the logical ID.
- **String** (plain physical ID such as `"rtb-12345"` or `"acl-12345"`): after stripping any `REF: ` prefix, compares the value against `logicalToPhysical[logicalId]`. This covers CloudFormation templates that hardcode parent resource IDs rather than using intrinsic functions.
- **Map** (`{"Ref": "LogicalName"}`): compares the ref name directly against the logical ID.
- **Map** (`{"Fn::ImportValue": "ExportName"}`): resolves both the import name and the logical ID through `logicalToPhysical`, then compares their physical IDs.

Regression tests for the plain-physical-ID case are in `lib/template_test.go` (`TestResourceIdMatchesLogical_HardcodedPhysicalId`, `TestFilterRoutesByLogicalId_HardcodedPhysicalId`, `TestFilterNaclEntriesByLogicalId_HardcodedPhysicalId`).

## logicalToPhysical Map

Built in `cmd/drift.go` `separateSpecialCases`. Contains both:
1. CloudFormation logical resource IDs mapped to physical IDs (from `DescribeStackResources`)
2. CloudFormation export names mapped to export values (from `ListExports`)

This dual population is what makes `Fn::ImportValue` resolution work -- the export name resolves to the same physical ID as the logical resource ID.

## Filter Function Signatures

All three filter functions now consistently accept `logicalToPhysical map[string]string`:
- `FilterNaclEntriesByLogicalId(logicalId, template, params, logicalToPhysical)` -- in `lib/template.go`
- `FilterRoutesByLogicalId(logicalId, template, params, logicalToPhysical)` -- in `lib/template.go`
- `FilterTGWRoutesByLogicalId(logicalId, template, params, logicalToPhysical)` -- in `lib/tgw_routetables.go`

## Fn::ImportValue Handling

The `customImportValueHandler` in `lib/template.go` preserves `Fn::ImportValue` as `map[string]any{"Fn::ImportValue": input}` during template parsing. This is intentional -- it allows downstream code (like `stringPointer` and `resourceIdMatchesLogical`) to resolve the import value using the `logicalToPhysical` map at filtering time.
