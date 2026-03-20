# Deployment Logging

## Location

- Core: `lib/logging.go`
- Tests: `lib/logging_test.go`
- Callers: `cmd/deploy_helpers.go`

## Architecture

`DeploymentLog` is a struct representing a deployment log entry. It records account, region, stack name, changeset changes, deployment status, and timestamps. Logs are written as newline-delimited JSON to a file configured via `viper` (`logging.filename`, `logging.enabled`).

Key methods:
- `Write() error` — marshals to JSON and appends to the log file
- `Success() error` — sets status to SUCCESS and calls Write
- `Failed(failures) error` — sets status to FAILED, records failures, and calls Write
- `ReadAllLogs()` — reads all log entries from the file, skipping malformed lines

`writeLogToFile()` is the low-level function that handles file I/O with buffered writing.

## Callers

`cmd/deploy_helpers.go:printDeploymentResults()` calls `logObj.Success()` and `logObj.Failed()` after deployment completes. Errors from these calls are logged as warnings to stderr — they do not block the deployment result reporting.

## Error Handling Pattern

All three methods (`Write`, `Success`, `Failed`) return errors. This was changed from the original panic/log.Fatal pattern in T-466. The callers handle errors non-fatally since log writing is ancillary to the deployment workflow.

## Testing

Tests use `viper.Set` to configure temporary log files via `t.TempDir()`. Be careful to save and restore viper settings in deferred functions since tests share a global viper instance.

The `readAllLogs()` function has an internal variant that accepts a logger function for testing (allows capturing warnings without using `log.Printf` directly).
