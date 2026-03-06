# Transit Gateway Drift Detection

## Template Property Formats

CloudFormation template properties (like `TransitGatewayRouteTableId`, `RouteTableId`, etc.) can appear in multiple formats depending on template preprocessing:

1. **Preprocessed string**: `"REF: LogicalId"` - the template parser has already resolved `!Ref` to this string format
2. **Ref map**: `{"Ref": "LogicalId"}` - raw CloudFormation intrinsic function
3. **ImportValue map**: `{"Fn::ImportValue": "ExportName"}` - cross-stack reference
4. **Plain string**: `"tgw-rtb-12345"` - hardcoded physical ID

Code that reads these properties must use a type switch or `extractStringProperty` helper. Direct type assertions like `prop.(string)` will panic on map values.

## Key Functions

- `FilterTGWRoutesByLogicalId` (lib/tgw_routetables.go): Filters TGW routes by route table logical ID. Uses `tgwRouteMatchesRouteTable` helper for type-safe matching.
- `FilterRoutesByLogicalId` (lib/template.go): Analogous function for regular route tables. **Note**: As of T-365, this function still uses the unsafe `.(string)` type assertion pattern and may need the same fix.
- `extractStringProperty` (lib/tgw_routetables.go): General-purpose helper that resolves property values from all formats. Used by `TGWRouteResourceToTGWRoute` for other properties.
- `tgwRouteMatchesRouteTable` (lib/tgw_routetables.go): Type-safe route table ID matching supporting all property formats.

## logicalToPhysical Map

The `logicalToPhysical` map (built in `cmd/drift.go:separateSpecialCases`) contains both:
- Stack resource logical ID -> physical ID mappings
- CloudFormation export name -> value mappings (for `Fn::ImportValue` resolution)

This dual purpose allows `extractStringProperty` and `tgwRouteMatchesRouteTable` to resolve both `Ref` and `ImportValue` references.
