# Bugfix Report: Describe Changeset Errors Write Diagnostics to Stdout

**Date:** 2026-05-22
**Status:** Fixed
**Ticket:** T-1238

## Description of the Issue

`cmd/describe_changeset.go` had several error paths that wrote user-facing
diagnostics to stdout via `fmt.Print*` and then called `os.Exit(1)` directly,
bypassing the shared `failWithError` handler established by T-1014.

**Reproduction steps:**
1. Run `fog describe changeset --output json` with both `--changeset` and `--url` set, or against a stack whose changeset cannot be retrieved.
2. Capture stdout separately from stderr.
3. Observe the error text on stdout, contaminating the structured (JSON) output and breaking downstream parsing.

**Impact:** Medium. Stdout contamination breaks machine-readable output for the
`describe changeset` command. The direct `os.Exit` calls also bypassed
`failWithError`'s debug-mode panic behaviour and could not be intercepted by
tests.

## Investigation Summary

- **Symptoms examined:** Error branches in `describe_changeset.go` using `fmt.Println`/`fmt.Print`/`fmt.Printf` (stdout) plus `os.Exit(1)`.
- **Code inspected:** `cmd/describe_changeset.go`, the shared helpers `failWithError`, `printMessage`, `formatError`, `createStderrOutput`, `osExitFunc` (`cmd/helpers.go`, `cmd/output_helpers.go`), and the `fetchChangeset` analogue in `cmd/deploy.go` that already routes the same retrieval failures to stderr.
- **Hypotheses tested:** Confirmed the fallback for an unknown output format does not produce a render error, so the render-failure branch needed an injectable seam to be testable in-process.

## Discovered Root Cause

The three (and the equivalent fourth `AddChangeset`) error branches predated the
T-1014 stderr-routing convention and were never migrated. They wrote to stdout
and exited via the bare `os.Exit`.

**Defect type:** Output stream misrouting / missed convention migration.

**Why it occurred:** The describe changeset command was written before the
shared `failWithError`/`printMessage(formatError(...))` stderr helpers and the
`osExitFunc` test seam existed, and was not updated when those were introduced.

**Contributing factors:** No regression test asserted stdout stays empty on
these error paths.

## Resolution for the Issue

**Changes made:**
- `cmd/describe_changeset.go` — mutually exclusive `--changeset`/`--url` check now routes through `failWithError` using a named error, and runs before AWS config loading so invalid input fails fast.
- `cmd/describe_changeset.go` — `GetChangeset` and `AddChangeset` failures route through a new `reportRetrieveChangesetFailure` helper that emits to stderr via `printMessage(formatError(...))` and exits through `osExitFunc`, mirroring `fetchChangeset` in `deploy.go`.
- `cmd/describe_changeset.go` — render failure in `buildAndRenderChangeset` routes through `failWithError`; the render call is now made through an injectable `renderChangesetDocument` variable so tests can exercise the failure branch.

**Approach rationale:** Reuse the existing stderr helpers and `osExitFunc` seam
exactly as T-1014 did. The `renderChangesetDocument` variable follows the
established injection pattern (`loadAWSConfig`, `getCfnClient`).

**Alternatives considered:**
- Triggering a real render error via output configuration — rejected because unknown formats fall back to table and do not error.
- Converting `renderDocument` itself into a package-level var — rejected as broader than needed and risks cross-file test interference.

## Regression Test

**Test file:** `cmd/stream_separation_test.go`
**Test names:** `TestDescribeChangeset_MutuallyExclusiveFlags_StderrOnly`, `TestReportRetrieveChangesetFailure_StderrOnly`, `TestBuildAndRenderChangeset_RenderFailure_StderrOnly`

**What it verifies:** On each fixed error path, stdout stays empty, the error is
reported on stderr, and the process exits with code 1 through `osExitFunc`.

**Run command:** `go test ./cmd/ -run 'StderrOnly'`

## Affected Files

| File | Change |
|------|--------|
| `cmd/describe_changeset.go` | Route the four error branches through `failWithError`/`reportRetrieveChangesetFailure`; add `renderChangesetDocument` injection seam. |
| `cmd/stream_separation_test.go` | Add three regression tests asserting stdout stays empty on the fixed error paths. |

## Verification

**Automated:**
- [x] Regression tests pass (and fail against the pre-fix stdout behaviour)
- [x] Full test suite passes (`go test ./...` and `INTEGRATION=1 go test ./...`)
- [x] `golangci-lint run ./...` reports 0 issues

**Manual verification:**
- Confirmed via a temporary revert that the new tests fail when the error paths write to stdout.

## Prevention

**Recommendations to avoid similar bugs:**
- Always route user-facing errors through `failWithError` or `printMessage(formatError(...))`; never `fmt.Print*` to stdout for diagnostics.
- Use `osExitFunc` rather than `os.Exit` in command handlers so exits are testable.
- Add stdout-empty stream-separation assertions for new error paths.

## Related

- T-1014 ("Route CLI Command Errors to Stderr") established the pattern reused here.
- `fetchChangeset` in `cmd/deploy.go` is the analogue for the retrieval-failure handling.
