# Bugfix Report: Guard GetStack Against Empty or Multi-Stack Results

**Date:** 2025-02-27
**Status:** Fixed

## Description of the Issue

`lib/stacks.go:GetStack` blindly accesses `resp.Stacks[0]` after calling `DescribeStacks`, without checking the length of the returned slice. When `StackName` is empty or contains `*`, the `StackName` parameter is omitted from the API call, causing `DescribeStacks` to return zero or multiple stacks.

**Reproduction steps:**
1. Call `GetStack` with an empty string or a wildcard-containing stack name
2. AWS returns zero stacks → index-out-of-range panic
3. AWS returns multiple stacks → silently returns the wrong stack

**Impact:** Runtime panic (index out of range) or incorrect stack returned, affecting all callers including `DeployInfo.GetStack`, `DeployInfo.GetFreshStack`, and `ChangesetInfo.GetStack`.

## Investigation Summary

- **Symptoms examined:** `GetStack` accesses `resp.Stacks[0]` unconditionally on line 203
- **Code inspected:** `lib/stacks.go` (standalone `GetStack`, `DeployInfo.GetStack`, `DeployInfo.GetFreshStack`), `lib/changesets.go` (`ChangesetInfo.GetStack`)
- **Hypotheses tested:** Confirmed that when `stackname` is empty or contains `*`, the input filter is skipped and `DescribeStacks` can return any number of stacks

## Discovered Root Cause

**Defect type:** Missing validation

The function assumed `DescribeStacks` would always return exactly one stack. However, when the `StackName` filter is omitted (empty or wildcard input), AWS returns all stacks in the account — which could be zero or many.

**Why it occurred:** The guard for empty/wildcard names at line 196 correctly skips setting the filter, but no corresponding guard was added for the response.

**Contributing factors:** The AWS API design where omitting `StackName` returns all stacks rather than an error.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:194-210` — Added length checks: return error when 0 stacks found, return error when >1 stack found, only return `resp.Stacks[0]` when exactly 1 stack is present.

**Approach rationale:** A function named `GetStack` (singular) should return exactly one stack. Returning an explicit error for 0 or multiple results is the safest behavior and matches caller expectations.

**Alternatives considered:**
- Return first stack when multiple found — rejected because it silently returns potentially wrong data
- Filter by name client-side when multiple returned — rejected as it would change the function's contract and callers already pass specific names

## Regression Test

**Test file:** `lib/stacks_test.go`, `lib/stacks_refactored_test.go`
**Test names:** `TestGetStack/empty_stacks_response`, `TestGetStack/multiple_stacks_response`, `TestGetStack_WithDependencyInjection/empty_stack_name_-_no_stacks_returns_error`, `TestGetStack_WithDependencyInjection/empty_stack_name_-_multiple_stacks_returns_error`

**What it verifies:** GetStack returns an error (not a panic) when the API response contains zero stacks or more than one stack.

**Run command:** `go test ./lib/... -run TestGetStack -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added length guards before accessing `resp.Stacks[0]` |
| `lib/stacks_test.go` | Added regression tests for empty and multi-stack responses |
| `lib/stacks_refactored_test.go` | Updated existing test and added regression test for empty response |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Always check slice length before indexing into API response slices
- Functions retrieving a single resource should validate exactly-one semantics
- Consider using a linter rule or code review checklist for unchecked slice access

## Related

- Transit ticket: T-275
