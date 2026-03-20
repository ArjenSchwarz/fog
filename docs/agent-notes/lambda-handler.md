# Lambda Handler

## Architecture

The Lambda handler is in `main.go`. When `AWS_LAMBDA_FUNCTION_NAME` is set, `main()` calls `lambda.Start(HandleRequest)` instead of `cmd.Execute()`.

`HandleRequest` reads configuration from environment variables:
- `ReportS3Bucket` (required) — S3 bucket for report output
- `ReportOutputFormat` (required) — output format (markdown, html, etc.)
- `ReportNamePattern` (optional) — filename pattern with placeholders
- `ReportTimezone` (optional) — timezone for timestamps

It delegates to `cmd.GenerateReportFromLambda`, which sets report flags and calls `generateReport()`.

## Error Handling

`HandleRequest` returns `error` to the Lambda runtime. Required env vars are validated upfront. `generateReport()` returns `error` (not `os.Exit`), which flows back through `GenerateReportFromLambda` to `HandleRequest`.

The CLI path (`stackReport`) wraps the error with `failWithError` which prints and calls `os.Exit(1)`.

## Key Files

- `main.go` — Handler, EventBridgeMessage struct, env var validation
- `cmd/report.go` — `GenerateReportFromLambda`, `generateReport`, report flags
- `cmd/helpers.go` — `failWithError` (CLI-only error handling)

## Gotchas

- `generateReport` must return errors, not call `failWithError`/`os.Exit`. The Lambda runtime needs the error returned from the handler to report failures.
- `ReportTimezone` is handled by `setTimezoneIfPresent` which only overrides the viper default when non-empty (see T-352 bugfix for empty timezone panic).
