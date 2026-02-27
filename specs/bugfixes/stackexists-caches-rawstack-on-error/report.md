# Bugfix Report: StackExists Caches RawStack Only on Error

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

`StackExists` in `lib/stacks.go` was setting `deployment.RawStack` when `GetStack` returned an error, instead of when it succeeded. This meant `RawStack` was populated with a zero-value `types.Stack` on failure and left `nil` on success. Downstream code relying on the cached `RawStack` (e.g., `DeployInfo.GetStack`) would never find a cached value after a successful `StackExists` call, causing unnecessary duplicate AWS API calls and potentially exposing zero-value data on error paths.

**Reproduction steps:**
1. Call `StackExists` with a valid, existing stack name
2. Inspect `deployment.RawStack` after the call
3. Observe that `RawStack` is `nil` despite the stack existing

**Impact:** Moderate — `RawStack` cache was never populated on success, causing redundant `DescribeStacks` API calls. On error, a zero-value stack was cached, which could lead to misleading data if `RawStack` was accessed without checking for errors.

## Investigation Summary

- **Symptoms examined:** `RawStack` field set inside an `if err != nil` block, which is the error path
- **Code inspected:** `lib/stacks.go` lines 254–261 (`StackExists`), lines 494–504 (`DeployInfo.GetStack` which reads `RawStack`)
- **Hypotheses tested:** Confirmed the condition was simply inverted — `err != nil` should be `err == nil`

## Discovered Root Cause

The `if err != nil` condition on line 257 was inverted. It should have been `if err == nil`.

**Defect type:** Logic error (inverted conditional)

**Why it occurred:** Likely a typo or copy-paste error — the pattern `if err != nil` is idiomatic Go for error handling, making this easy to write by muscle memory even when the intent was the opposite.

**Contributing factors:** The existing tests only checked the boolean return value, not the `RawStack` side effect, so the bug was not caught.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:257` — Changed `if err != nil` to `if err == nil` so `RawStack` is cached on success

**Approach rationale:** Minimal one-character fix that directly addresses the inverted condition.

**Alternatives considered:**
- Restructuring `StackExists` to return the stack directly — rejected as it would change the function signature and all callers

## Regression Test

**Test file:** `lib/stacks_refactored_test.go`
**Test name:** `TestStackExists_CachesRawStack`

**What it verifies:**
- When a stack exists, `RawStack` is set to the returned stack (not nil)
- When a stack does not exist, `RawStack` remains nil

**Run command:** `go test ./lib/... -run TestStackExists_CachesRawStack -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Fixed inverted condition in `StackExists` |
| `lib/stacks_refactored_test.go` | Added regression test `TestStackExists_CachesRawStack` |
| `specs/bugfixes/stackexists-caches-rawstack-on-error/report.md` | Bugfix report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- When caching side effects in functions, always test the side effect (not just the return value)
- Consider using `if err == nil` more carefully — linting rules or code review checklists should flag assignments inside error-handling blocks that don't relate to error handling

## Related

- Transit ticket: T-142
