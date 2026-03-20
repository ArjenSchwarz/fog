# Bugfix Report: writeLogToFile Drops Close Errors

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `writeLogToFile` function in `lib/logging.go` silently discarded `file.Close()` errors. The deferred closure attempted to assign the close error to a local variable `err`, but because the function used an unnamed return (`error` instead of `(err error)`), the assignment had no effect on the returned value.

**Reproduction steps:**
1. Call `writeLogToFile` with valid content and a valid file path
2. Have the underlying `file.Close()` return an error (e.g., filesystem error during flush-on-close)
3. Observe that `writeLogToFile` returns `nil` despite the close failure

**Impact:** Low severity. Close errors on regular files are rare, but when they occur (e.g., disk full detected on close, NFS errors), the caller would not know the data may not have been persisted. This could lead to silently corrupted or incomplete deployment logs.

## Investigation Summary

The bug was identified from code inspection based on the ticket name.

- **Symptoms examined:** The defer block in `writeLogToFile` assigns `err = cerr` but `err` is a local variable from `os.OpenFile`, not a named return
- **Code inspected:** `lib/logging.go:114-134` — the `writeLogToFile` function
- **Hypotheses tested:** Confirmed that Go unnamed returns ignore deferred assignments to local variables

## Discovered Root Cause

The function signature `func writeLogToFile(contents []byte, outputFile string) error` uses an unnamed return. The deferred closure captures the local variable `err` (from line 115, the `os.OpenFile` call) and assigns the close error to it. However, when the function executes `return nil` on line 133, Go returns the literal `nil` — it does not check the local `err` variable. Named returns are required for deferred functions to modify the return value.

**Defect type:** Logic error — incorrect use of defer pattern for error propagation

**Why it occurred:** The defer-close-error pattern requires a named return value to work. The function was written with an unnamed return, making the pattern ineffective.

**Contributing factors:** This is a common Go pitfall. The code reads as though it should work (it captures `err` in a closure), but the semantics of unnamed returns mean the assignment is a no-op with respect to the caller.

## Resolution for the Issue

**Changes made:**
- `lib/logging.go` — Extracted file-writing logic into a new `writeToFile(contents []byte, file io.WriteCloser) (err error)` function with a named return. `writeLogToFile` now opens the file and delegates to `writeToFile`.

**Approach rationale:** Extracting `writeToFile` with an `io.WriteCloser` parameter serves two purposes: (1) the named return `(err error)` fixes the bug, and (2) accepting an interface makes the close-error behavior directly testable with a mock closer.

**Alternatives considered:**
- Simply adding a named return to `writeLogToFile` — This would fix the bug but leaves the function untestable for close errors since it manages the file internally. The extraction costs almost nothing and enables proper testing.

## Regression Test

**Test file:** `lib/logging_test.go`
**Test names:**
- `TestWriteLogToFile_PropagatesCloseError` — verifies the happy path still works
- `TestWriteToFile_PropagatesCloseError` — verifies close errors are returned when no write error occurs
- `TestWriteToFile_WriteErrorTakesPrecedenceOverCloseError` — verifies write errors take precedence over close errors

**What it verifies:** That `writeToFile` returns the close error when the write and flush succeed but `Close()` fails, and that write errors take precedence over close errors.

**Run command:** `go test ./lib/ -run "TestWriteLogToFile_PropagatesCloseError|TestWriteToFile_PropagatesCloseError|TestWriteToFile_WriteErrorTakesPrecedenceOverCloseError" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/logging.go` | Extracted `writeToFile` with named return; `writeLogToFile` delegates to it |
| `lib/logging_test.go` | Added three regression tests for close error propagation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When using the defer-close-error pattern in Go, always use named returns. The pattern `defer func() { if cerr := f.Close(); cerr != nil && err == nil { err = cerr } }()` only works with `(err error)` named returns.
- Consider a linter rule (e.g., `errcheck` or a custom rule) that flags deferred close calls where the error is captured but the function uses unnamed returns.

## Related

- Transit ticket: T-393
