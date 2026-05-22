# Error Handling Patterns

## Library Code (`lib/`)

Library functions must return errors, never call `log.Fatal`, `os.Exit`, or `panic` for recoverable errors. The cmd layer handles error presentation via `failWithError()`.

### Functions that have been fixed to return errors:
- `lib/drift.go`: `StartDriftDetection`, `WaitForDriftDetectionToFinish`, `GetDefaultStackDrift` (T-339)
- `lib/resources.go`: `GetResources` (T-465)
- `lib/drift.go`: `GetUncheckedStackResources` (T-465, propagates from GetResources)

### Pattern for AWS SDK error handling in lib:
```go
func SomeFunction(...) (ResultType, error) {
    result, err := svc.SomeAWSCall(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed to <action>: %w", err)
    }
    return result, nil
}
```

For throttling with retry:
```go
if ae.ErrorCode() == "Throttling" && ae.ErrorMessage() == "Rate exceeded" {
    time.Sleep(5 * time.Second)
    result, err = svc.SomeAWSCall(ctx, input)
    if err != nil {
        return nil, fmt.Errorf("failed after throttling retry: %w", err)
    }
}
```

## Command Layer (`cmd/`)

Commands use `failWithError(err)` which writes the error to **stderr**, panics in debug mode, then calls `osExitFunc(1)`.

### Stream separation (T-1014, T-1238)

User-facing errors and progress must go to stderr so structured stdout (e.g. `--output json`) stays parseable. Never use `fmt.Print*` to stdout for diagnostics. Use one of:
- `failWithError(err)` — for fatal errors that should exit.
- `printMessage(formatError(...))` — writes a styled error to stderr without exiting; pair with `osExitFunc(1)` when you need to bail (see `fetchChangeset` and `reportRetrieveChangesetFailure`).

Always call `osExitFunc` (the package var in `cmd/helpers.go`), never `os.Exit` directly, so tests can intercept the exit.

`cmd/stream_separation_test.go` is the canonical home for stdout-empty/stderr regression tests. It provides `captureBothStreams`, `captureStderr`, and `runWithInterceptedExit` (stubs `osExitFunc` with a panic to unwind). To test a render-failure branch, inject through a package-level render var (e.g. `renderChangesetDocument` in `describe_changeset.go`) — an unknown output format does NOT error (it falls back to table).

## Output option helpers (T-1190)

`config.Config.GetOutputOptions()` and `GetFileOutputOptions()` return
`([]output.OutputOption, error)`. When `output-file` is explicitly set and
`output.NewFileWriter` fails, they return the error instead of logging a warning and
continuing — otherwise commands with `--file` could report success while silently
dropping the requested file. When no `output-file` is set they return early with a nil
error (behaviour unchanged). Callers (`cmd/output_helpers.go` `renderDocument`,
`cmd/report.go` `getReportOutputOptions` and the report render path) must propagate the
error. To force a `NewFileWriter` failure in tests, point the path under a regular file so
the internal `os.MkdirAll` fails with a non-directory error (portable, no root needed).

## Testing Error Paths

Error-path tests should directly assert returned errors, not use subprocess re-execution patterns (`os.Args[0]`, `exec.Command`, `FOG_TEST_HELPER`). The subprocess pattern was used before functions returned errors and should not be used for new tests. Prefer in-process `osExitFunc` interception (see `runWithInterceptedExit`).
