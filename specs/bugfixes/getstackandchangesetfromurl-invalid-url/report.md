# Bugfix Report: GetStackAndChangesetFromURL Invalid URL

**Date:** 2026-02-27
**Status:** Fixed
**Transit:** T-253

## Description of the Issue

`GetStackAndChangesetFromURL` in `lib/changesets.go` used `log.Fatal` and `panic` to handle URL parsing errors. When a user supplied an invalid changeset URL (e.g., with bad percent-encoding), the entire process would exit immediately instead of returning an error that the CLI could report gracefully.

**Reproduction steps:**
1. Run `fog describe changeset --url "https://example.com?stackId=%zz"`
2. Observe the process exits with a fatal log or panic instead of a user-friendly error message

**Impact:** Any invalid URL input to the `--url` flag caused an unrecoverable crash. The CLI should handle bad input gracefully and report the error to the user.

## Investigation Summary

- **Symptoms examined:** `log.Fatal(err)` on line 90 and `panic(err)` on lines 98 and 102 of `lib/changesets.go`
- **Code inspected:** `lib/changesets.go` (the function), `cmd/describe_changeset.go` (the caller), and both test files
- **Hypotheses tested:** The function signature `(string, string)` made it impossible to propagate errors — the only option was to terminate the process

## Discovered Root Cause

The function used `log.Fatal` (which calls `os.Exit(1)`) and `panic` for error handling because the original return signature `(string, string)` had no way to communicate errors to the caller.

**Defect type:** Missing error propagation

**Why it occurred:** The function was written with a two-return-value signature, so errors had nowhere to go except fatal exits.

**Contributing factors:** No error-path tests existed to catch this behaviour.

## Resolution for the Issue

**Changes made:**
- `lib/changesets.go:87` — Changed function signature from `(string, string)` to `(string, string, error)` and replaced `log.Fatal`/`panic` with `fmt.Errorf` wrapped returns
- `lib/changesets.go:3` — Removed unused `log` import
- `cmd/describe_changeset.go:69` — Updated caller to handle the returned error via `failWithError`
- `lib/changesets_test.go:463` — Updated test to use 3-return-value signature
- `lib/changesets_refactored_test.go:638` — Updated test to use 3-return-value signature and added parallel execution

**Approach rationale:** Changing the signature to return an error follows Go conventions and lets callers decide how to handle failures. The `fmt.Errorf` wrapping with `%w` preserves the original error for inspection.

**Alternatives considered:**
- Recovering from panic in callers — Rejected because it's non-idiomatic Go and hides the real issue

## Regression Test

**Test file:** `lib/changesets_refactored_test.go`
**Test name:** `TestGetStackAndChangesetFromURL_InvalidInput`

**What it verifies:** Invalid URL inputs (bad percent-encoding, empty strings, non-URL strings) return errors instead of causing process exit, panic, or silently producing empty IDs.

**Run command:** `go test ./lib -run TestGetStackAndChangesetFromURL_InvalidInput -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/changesets.go` | Changed signature to return error; replaced `log.Fatal`/`panic` with error returns |
| `cmd/describe_changeset.go` | Updated caller to handle returned error |
| `lib/changesets_test.go` | Updated test for new 3-return signature |
| `lib/changesets_refactored_test.go` | Updated test for new signature; added regression tests for invalid inputs |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass
- [x] Build succeeds

## Prevention

**Recommendations to avoid similar bugs:**
- Library functions should always return errors instead of calling `log.Fatal` or `panic`
- Reserve `log.Fatal` and `panic` for truly unrecoverable situations in `main()` or init code
- Add error-path test cases when writing URL/input parsing functions

## Related

- Transit ticket: T-253
