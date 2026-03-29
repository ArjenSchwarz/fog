# Bugfix Report: Guard Drift Tag Diff Parsing

**Date:** 2026-03-29
**Status:** Fixed

## Description of the Issue

The `tagDifferences` function in `cmd/drift.go` panics when AWS returns drift property paths that are shorter than expected or when pointer fields (`PropertyPath`, `ExpectedValue`, `ActualValue`) are nil.

**Reproduction steps:**
1. Run drift detection on a stack where AWS returns a tag property path like `/Tags/0` (3 segments) instead of `/Tags/0/Key` (4 segments) in a NOT_EQUAL diff
2. Or encounter a property with a nil `PropertyPath`, `ExpectedValue`, or `ActualValue` pointer
3. Observe a panic crash instead of structured output

**Impact:** Drift reporting crashes with an unrecoverable panic, preventing users from seeing any drift results for the affected stack.

## Investigation Summary

Systematic inspection of the `tagDifferences` function revealed multiple unsafe operations:

- **Symptoms examined:** Panic on nil pointer dereference and index-out-of-bounds
- **Code inspected:** `cmd/drift.go` lines 619-684, caller at line 134-145, AWS SDK type definitions
- **Hypotheses tested:** Confirmed that `types.PropertyDifference` fields are all `*string` pointers (can be nil despite being marked "required" by AWS), and that `strings.Split` on short paths produces fewer segments than the code assumes

## Discovered Root Cause

The function made two categories of unsafe assumptions:

1. **No nil guards on pointer fields:** `*property.PropertyPath` (line 624) and `*property.ExpectedValue`/`*property.ActualValue` (line 667) are dereferenced without nil checks.
2. **No bounds checks on path segments:** The default case accesses `pathsplit[3]` and `pathsplit[2]` without verifying the slice has at least 4 elements. A path like `/Tags/0` produces only 3 segments.

Additionally, `json.Indent` was called on the raw `aws.ToString()` output without checking if it was valid JSON first — empty strings from nil pointers would cause errors.

**Defect type:** Missing input validation

**Why it occurred:** The original code assumed AWS CloudFormation would always return well-formed, complete property paths and non-nil pointer fields. The commented-out `verifyTagOrder` function in the same file actually showed the correct pattern with `len(pathsplit) == 4` bounds checking, but `tagDifferences` was not written with the same defensive style.

**Contributing factors:** AWS SDK Go v2 uses pointer types for all struct fields, making nil a possibility even for "required" fields.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go:619-625` — Added early return when `PropertyPath` is nil or empty, and bounds check for `len(pathsplit) < 2`
- `cmd/drift.go:632-636` — Added `json.Valid` check before `json.Indent` to handle empty/invalid JSON from nil pointer values
- `cmd/drift.go:649-651` — Added `expected.Len() > 0` guard before indexing into the expected buffer string
- `cmd/drift.go:667-670` — Added nil-safe empty buffer check in Add case
- `cmd/drift.go:680` — Replaced `*property.ExpectedValue`/`*property.ActualValue` with `aws.ToString()` for nil-safe comparison
- `cmd/drift.go:689-695` — Added `len(pathsplit) >= 4` guard before accessing `pathsplit[3]` and `pathsplit[2]`

**Approach rationale:** Defensive guard-and-return approach — when inputs are malformed, the function returns empty strings (no output) rather than crashing. This matches the existing pattern where unhandled tags return `"", ""`.

**Alternatives considered:**
- Logging a warning for malformed paths — rejected because the drift output system uses structured formatters, not log messages
- Returning an error value — rejected because the function signature returns `(string, string)` and callers don't expect errors; changing the signature would be a larger refactor

## Regression Test

**Test file:** `cmd/drift_test.go`
**Test name:** `TestTagDifferences_MalformedPaths`

**What it verifies:** Seven sub-tests covering nil `PropertyPath`, short paths (2 and 3 segments) in the default case, nil `ExpectedValue`, nil `ActualValue`, empty string path, and nil values in the Add case. Each confirms the function returns gracefully without panicking.

**Run command:** `go test ./cmd -run TestTagDifferences_MalformedPaths -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Added nil/bounds guards to `tagDifferences` function |
| `cmd/drift_test.go` | Added 7 regression test cases for malformed inputs |

## Verification

**Automated:**
- [x] Regression test passes (7/7 sub-tests)
- [x] Full test suite passes (all packages)
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- Always validate AWS SDK pointer fields before dereferencing — use `aws.ToString()` for safe access
- Always check slice bounds before indexing, especially after `strings.Split` where the number of segments depends on input
- Use the pattern from the commented-out `verifyTagOrder` function (`len(pathsplit) == 4`) as a reference for defensive path parsing

## Related

- Transit ticket: T-589
