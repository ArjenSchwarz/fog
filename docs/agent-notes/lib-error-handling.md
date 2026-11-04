# Lib Layer Error Handling

## Known Issue: log.Fatalln in Library Code

Several functions in the `lib/` package use `log.Fatalln` or `log.Fatalf` to handle errors. This terminates the process from library code, bypassing the cmd layer's error handling (`failWithError` in `cmd/helpers.go`).

### Fixed

- `lib/outputs.go` `GetExports` - T-464: Changed to return `([]CfnOutput, error)` instead of calling `log.Fatalln`.
- `lib/resources.go` `GetResources` - Now returns `([]CfnResource, error)` (see `specs/bugfixes/getresources-exits-on-api-errors/`).

## Convention

Library functions (`lib/`) should return errors. The cmd layer decides how to present errors to users via `failWithError()` in `cmd/helpers.go`, which formats the error and optionally panics in debug mode.
