# Bugfix Report: ReadAllLogs Panics on Malformed Log Lines

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

`ReadAllLogs` crashed the CLI when a single malformed line existed in the deployment log file.

**Reproduction steps:**
1. Create a log file with valid deployment log JSON plus one malformed line.
2. Call `ReadAllLogs` (for example via history/report flows).
3. Observe panic from `json.Unmarshal` and CLI termination.

**Impact:** One corrupted or manually edited log line could crash all flows that read deployment history.

## Investigation Summary

Systematic inspection showed parsing was done line-by-line but parse failures were treated as fatal panics.

- **Symptoms examined:** panic `invalid character ... looking for beginning of object key string`
- **Code inspected:** `lib/logging.go`, `lib/logging_test.go`
- **Hypotheses tested:** scanner/file access failures, sort behavior, and malformed JSON handling

## Discovered Root Cause

`ReadAllLogs` assumed every line in the log file was valid JSON and used `panic(err)` on unmarshal failures.

**Defect type:** Error handling defect (non-recoverable panic for recoverable bad input)

**Why it occurred:** The function signature returns only `[]DeploymentLog`, so invalid line handling was implemented as a panic path rather than graceful skip/recovery.

**Contributing factors:** Log files can contain malformed lines due to partial writes or manual edits, which violated the implicit all-valid-lines assumption.

## Resolution for the Issue

**Changes made:**
- `lib/logging.go:170-177` - Replaced panic on JSON unmarshal failure with a warning log and `continue` to skip malformed lines.
- `lib/logging_test.go:350-415` - Added regression test covering mixed valid and malformed log lines.

**Approach rationale:** Skipping bad lines preserves valid history and prevents CLI crashes while still surfacing malformed input via warning logs.

**Alternatives considered:**
- Return `([]DeploymentLog, error)` from `ReadAllLogs` - not chosen due broader API changes.
- Stop processing at first malformed line without panic - not chosen because it still drops valid entries after the bad line.

## Regression Test

**Test file:** `lib/logging_test.go`
**Test name:** `TestReadAllLogsSkipsMalformedLogEntries`

**What it verifies:** `ReadAllLogs` does not panic on malformed lines, skips invalid entries, and keeps valid entries sorted newest-first.

**Run command:** `go test ./lib -run TestReadAllLogsSkipsMalformedLogEntries -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/logging.go` | Graceful handling for malformed JSON log lines |
| `lib/logging_test.go` | Regression test for malformed log line handling |
| `specs/bugfixes/readalllogs-panics-on-malformed-log-lines/report.md` | Bug investigation and resolution report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed malformed line now emits warning and valid lines are still returned.

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer recoverable error handling for file parsing paths that process historical data.
- Add regression tests for malformed/partial input in line-oriented parsers.
- Include line-number context in parser warnings to speed troubleshooting.

## Related

- Transit ticket: `T-340`
- PR: pending creation
