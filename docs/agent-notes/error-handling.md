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

Commands use `failWithError(err)` which formats the error and calls `os.Exit(1)` (or panics in debug mode).

## Testing Error Paths

Error-path tests should directly assert returned errors, not use subprocess re-execution patterns (`os.Args[0]`, `exec.Command`, `FOG_TEST_HELPER`). The subprocess pattern was used before functions returned errors and should not be used for new tests.
