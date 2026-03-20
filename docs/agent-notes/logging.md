# Logging Module

## Overview

`lib/logging.go` handles deployment log persistence. Deployment logs are JSON-lines files written to a configurable path (`logging.filename` in viper config).

## Key Functions

- `writeLogToFile(contents, outputFile)` — opens the file and delegates to `writeToFile`
- `writeToFile(contents, io.WriteCloser) (err error)` — writes contents + newline via a buffered writer, then closes the file. Uses a named return so the deferred `Close()` error propagates correctly.
- `readAllLogs(logf)` — reads all JSON lines from the log file. Accepts a logger function for testability. Skips malformed lines with a warning rather than panicking.
- `DeploymentLog.Write()` — marshals the log to JSON and calls `writeLogToFile`. Panics on marshal failure, calls `log.Fatal` on write failure.

## Gotchas

- The `Write()` method uses `panic` for JSON marshal errors and `log.Fatal` for write errors. These are not recoverable in the normal error-handling sense.
- The defer-close-error pattern (`if cerr := file.Close(); cerr != nil && err == nil { err = cerr }`) requires a **named return** to work. Without it, the assignment to the local `err` variable has no effect on the return value. This was the root cause of T-393.
- `readAllLogs` has a 10MB scanner buffer limit for large log lines.

## Testing

- Tests use `viper.Set` to configure logging paths, so tests that run in parallel could interfere with each other. Current tests don't use `t.Parallel()`.
- `writeToFile` accepts `io.WriteCloser` to enable injecting mock closers that return errors on `Close()`.
