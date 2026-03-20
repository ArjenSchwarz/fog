# Lib Layer Error Handling

## Known Issue: log.Fatalln in Library Code

Several functions in the `lib/` package use `log.Fatalln` or `log.Fatalf` to handle errors. This terminates the process from library code, bypassing the cmd layer's error handling (`failWithError` in `cmd/helpers.go`).

### Fixed

- `lib/outputs.go` `GetExports` - T-464: Changed to return `([]CfnOutput, error)` instead of calling `log.Fatalln`.

### Still Present (as of 2026-03-20)

- `lib/resources.go` `GetResources` - Same pattern: `log.Fatalln` on paginator errors and `DescribeStackResources` errors. The function returns `[]CfnResource` with no error return. Needs the same refactor as `GetExports`.

## Convention

Library functions (`lib/`) should return errors. The cmd layer decides how to present errors to users via `failWithError()` in `cmd/helpers.go`, which formats the error and optionally panics in debug mode.
