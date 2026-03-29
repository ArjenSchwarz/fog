# Bugfix Report: Guard GetCleanedStackName Against Malformed ARN Input

**Date:** 2025-07-14
**Status:** Fixed
**Ticket:** T-659

## Description of the Issue

`GetCleanedStackName` in `lib/stacks.go` panics with an index-out-of-range error when called with a malformed ARN that does not contain a `/` separator. For example, passing `"arn:aws:cloudformation:ap-southeast-2:123456789012"` causes a runtime crash because `strings.Split` returns a single-element slice and the code unconditionally accesses index `[1]`.

**Reproduction steps:**
1. Create a `DeployInfo` with `StackName` set to an ARN-like string without a `/` (e.g. `"arn:aws:cloudformation:ap-southeast-2:123456789012"`)
2. Call `GetCleanedStackName()`
3. Observe panic: `runtime error: index out of range [1] with length 1`

**Impact:** Any command that calls `GetCleanedStackName` (e.g. `cmd/describe_changeset.go`) crashes if the stack name is a malformed ARN. This is a user-facing crash with no recovery.

## Investigation Summary

- **Symptoms examined:** The function splits the stack name by `/` and accesses `filtered[1]` without bounds checking.
- **Code inspected:** `lib/stacks.go:544-552`, `cmd/describe_changeset.go` (caller), existing tests in `lib/stacks_test.go`.
- **Hypotheses tested:** Confirmed that any ARN-prefixed string lacking a `/` triggers the panic.

## Discovered Root Cause

**Defect type:** Missing input validation (bounds check)

**Why it occurred:** The function assumed that any string starting with `"arn:"` would always contain at least one `/` character, which is true for well-formed CloudFormation stack ARNs but not for truncated or malformed inputs.

**Contributing factors:** The existing test suite only covered the happy path (valid ARN and plain name), so the missing guard was never caught.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:549-551` — Added a bounds check after `strings.Split`. If the split produces fewer than 2 elements (no `/` found), the original `StackName` is returned instead of panicking.

**Approach rationale:** The simplest and safest fix — a single bounds check that falls back to returning the original value. This matches the existing fallback behaviour for non-ARN inputs.

**Alternatives considered:**
- Using `strings.Contains(s, "/")` before splitting — adds an extra scan of the string; the length check after split is sufficient and idiomatic.
- Full ARN parsing with a library — overkill for this single extraction; the function's scope is narrow.

## Regression Test

**Test file:** `lib/stacks_test.go`
**Test name:** `TestDeployInfo_GetCleanedStackName` (expanded with 4 new sub-tests)

**What it verifies:**
- `malformed ARN without slash returns original` — ARN with no `/` returns the input unchanged
- `ARN prefix only returns original` — bare `"arn:"` returns the input unchanged
- `ARN with trailing slash returns empty stack name segment` — trailing `/` returns `""`
- `ARN with single slash and name returns name` — single `/` correctly extracts the name

**Run command:** `go test ./lib/... -run TestDeployInfo_GetCleanedStackName -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added bounds check in `GetCleanedStackName` |
| `lib/stacks_test.go` | Added 4 regression test cases for malformed ARN inputs |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run` — 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- Always validate slice indices after `strings.Split` before accessing specific elements.
- Include edge-case and malformed-input test cases when writing tests for parsing functions.

## Related

- Transit ticket: T-659
