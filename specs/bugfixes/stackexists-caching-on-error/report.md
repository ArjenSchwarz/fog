# Bugfix Report: StackExists Caching on Error

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

The `StackExists` function in `lib/stacks.go` cached a zero-value `types.Stack` into `deployment.RawStack` when `GetStack` returned an error. This meant that subsequent calls to `deployment.GetStack()` would find a non-nil `RawStack` and return the empty cached data instead of retrying the API call.

**Reproduction steps:**
1. Call `StackExists` with a stack name that doesn't exist (or when the API call transiently fails)
2. The function correctly returns `false`, but also sets `deployment.RawStack` to an empty `types.Stack{}`
3. Later calls to `deployment.GetStack()` find `RawStack != nil` and return the empty stack without making a fresh API call

**Impact:** Incorrect deployment output and logic when a stack doesn't exist or when a transient API failure occurs. The cached zero-value stack could cause downstream code to operate on empty data silently.

## Investigation Summary

- **Symptoms examined:** `StackExists` sets `deployment.RawStack` when `GetStack` returns an error, caching a zero-value stack
- **Code inspected:** `lib/stacks.go` — `StackExists` function (line 255-261) and `DeployInfo.GetStack` method (line 495-504)
- **Hypotheses tested:** The condition guarding the `RawStack` assignment was inverted (`err != nil` instead of `err == nil`)

## Discovered Root Cause

The condition in `StackExists` was inverted. The code read `if err != nil` where it should have read `if err == nil`.

**Defect type:** Logic error — inverted condition

**Why it occurred:** The `err != nil` condition is the common Go error-checking idiom, making it easy to write by reflex. However, in this case the intent was to cache the stack only on success, requiring the opposite condition.

**Contributing factors:** The existing tests for `StackExists` only checked the boolean return value, not the side effect on `deployment.RawStack`, so the inverted condition was not caught.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:258` — Changed `if err != nil` to `if err == nil` so that `deployment.RawStack` is only populated when the stack is successfully retrieved

**Approach rationale:** This is the minimal one-character fix that directly addresses the root cause. The caching logic in `DeployInfo.GetStack()` already correctly handles the nil check, so no other changes are needed.

**Alternatives considered:**
- Removing the caching entirely from `StackExists` — not chosen because caching the stack on success is a useful optimisation that avoids redundant API calls

## Regression Test

**Test file:** `lib/stacks_refactored_test.go`
**Test names:** `TestStackExists_DoesNotCacheOnError`, `TestStackExists_CachesOnSuccess`

**What it verifies:**
- `TestStackExists_DoesNotCacheOnError`: When `GetStack` returns an error, `deployment.RawStack` must remain `nil`
- `TestStackExists_CachesOnSuccess`: When `GetStack` succeeds, `deployment.RawStack` must be populated with the returned stack

**Run command:** `go test ./lib/ -run "TestStackExists_DoesNotCacheOnError|TestStackExists_CachesOnSuccess" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Fixed inverted condition in `StackExists` (`err != nil` → `err == nil`) |
| `lib/stacks_refactored_test.go` | Added two regression tests for `RawStack` caching behaviour |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- When a function has side effects (like setting a field), always test the side effects explicitly, not just the return value
- Consider using a linter rule or code review checklist item for verifying cache-on-error patterns
- The `DeployInfo.GetStack()` method already had the correct pattern — the standalone `StackExists` function should have mirrored it

## Related

- Transit ticket: T-166
