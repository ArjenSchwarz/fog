# Exports and Imports

## Key Files
- `lib/outputs.go` - Core logic for `GetExports` and `CfnOutput.FillImports`
- `lib/interfaces.go` - `CFNListImportsAPI` and `CFNExportsAPI` interfaces
- `lib/outputs_test.go` - Tests including pagination mocks

## How It Works
- `GetExports` retrieves all CloudFormation stacks (paginated via `DescribeStacksPaginator`), filters for exports, then checks each export's importers concurrently using goroutines
- `FillImports` does the same import lookup for a single `CfnOutput`
- Both use `cloudformation.NewListImportsPaginator` to handle multi-page `ListImports` responses (fixed in T-499)

## Interfaces
- `CFNListImportsAPI` matches the SDK's `ListImportsAPIClient` interface exactly, so our interface can be passed directly to `NewListImportsPaginator`
- `CFNExportsAPI` combines `CFNDescribeStacksAPI` + `CFNListImportsAPI`

## Testing Pattern
- `MockCFNClient` in `outputs_test.go` returns all imports in a single response (simple mock)
- `paginatingListImportsMock` simulates multi-page `ListImports` by encoding page index in `NextToken` as `"page:<index>"`
- `paginatingMockCFNClient` simulates multi-page `DescribeStacks` using a `map[string]DescribeStacksOutput` keyed by token

## Gotchas
- Error handling for `ListImports` treats all errors as "not imported" (there's a TODO to limit this to only "not found" errors)
- `GetExports` uses goroutines for concurrent import lookups per export — each goroutine creates its own paginator instance so there are no race conditions
