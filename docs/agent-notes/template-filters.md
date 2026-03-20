# Template Filter Functions

## Key Abstraction: resourceIdMatchesLogical

`resourceIdMatchesLogical` in `lib/template.go` is a shared helper used by all three filter functions (`FilterNaclEntriesByLogicalId`, `FilterRoutesByLogicalId`, `FilterTGWRoutesByLogicalId`) to match a resource property value against a logical resource ID.

It handles two forms of property values:
- **String** (`"REF: LogicalName"`): strips the `REF: ` prefix and compares directly against the logical ID.
- **Map** (`{"Fn::ImportValue": "ExportName"}`): resolves both the import name and the logical ID through `logicalToPhysical`, then compares their physical IDs.

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
