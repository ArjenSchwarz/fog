# Exports and Imports

## Key Files

- `lib/outputs.go` — `GetExports`, `FillImports`, `getOutputsForStack`, `isNotImportedError`
- `lib/outputs_test.go` — Tests with mock clients (`MockCFNClient`, `paginatingMockCFNClient`, `perExportMockCFNClient`)
- `cmd/exports.go` — CLI command that calls `GetExports`
- `lib/stacks.go` — `GetCfnStacks` calls `FillImports` for each output

## How It Works

`GetExports` retrieves all CloudFormation stack exports for an account/region, then concurrently checks each export's import status via `ListImports`. Results are collected through a channel using `importResult` (output + optional error).

`FillImports` is a method on `CfnOutput` that populates import info for a single export. Used by `GetCfnStacks` in `stacks.go`.

## Error Handling

The AWS `ListImports` API returns an error with the message "Export 'X' is not imported by any stack." when an export has no importers. This is an expected condition, not a real error. The `isNotImportedError()` helper checks for this specific message using `strings.Contains`.

All other `ListImports` errors (throttling, permissions, service errors) are propagated to callers. This distinction was added in T-514 — previously all errors were silently treated as "not imported".

## Concurrency

`GetExports` launches one goroutine per export for `ListImports` calls. Errors from individual goroutines are collected and joined via `errors.Join`. The function returns partial results alongside the error.

## Mock Clients

Test mocks return the "is not imported by any stack" message (not a generic "not found") for exports not in `ImportsByExport`. This matches real AWS behavior.
