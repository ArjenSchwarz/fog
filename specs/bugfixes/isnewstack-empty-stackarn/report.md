# Bugfix Report: IsNewStack Empty StackArn

**Date:** 2025-07-14
**Status:** Fixed
**Transit:** T-761

## Description of the Issue

`IsNewStack` could misclassify `REVIEW_IN_PROGRESS` stacks as not-new when `StackArn` was empty. This happens because `GetFreshStack` unconditionally used `StackArn` to look up the stack. Before a changeset is created, `StackArn` is typically empty, causing an unscoped `DescribeStacks` API call that either errors (multiple stacks in the account) or returns the wrong stack.

**Reproduction steps:**
1. Set `deployment.StackName` to an existing stack in `REVIEW_IN_PROGRESS` status
2. Leave `deployment.StackArn` empty (default path before changeset creation)
3. Call `deployment.IsNewStack(...)` in an account with multiple stacks
4. Observe `GetFreshStack` fails or returns incorrect stack, causing `IsNewStack` to return `false`

**Impact:** `ChangesetType` could be set to `UPDATE` instead of `CREATE` for new stacks in review. Behaviour was non-deterministic depending on the number of stacks in the AWS account.

## Investigation Summary

- **Symptoms examined:** `IsNewStack` returning `false` for `REVIEW_IN_PROGRESS` stacks when `StackArn` is empty
- **Code inspected:** `lib/stacks.go` — `IsNewStack`, `GetFreshStack`, `GetStack`, `StackExists`
- **Hypotheses tested:** The `GetFreshStack` method was identified as the source — it passes `deployment.StackArn` to `GetStack` without checking for an empty value

## Discovered Root Cause

`GetFreshStack` always passed `deployment.StackArn` to the package-level `GetStack` function. When `StackArn` was empty, `GetStack` omitted the `StackName` filter from the `DescribeStacks` API call, causing it to return all stacks in the account. With multiple stacks, this triggered an error (`expected exactly one stack`); with a single stack, it could silently return the wrong one.

**Defect type:** Missing fallback / empty-value guard

**Why it occurred:** `StackArn` is only populated after changeset creation (`deployment.StackArn = changeset.StackID`), but `IsNewStack` is called before changeset creation during `prepareDeployment`.

**Contributing factors:** `StackExists` (called first in `IsNewStack`) correctly uses `StackName`, masking the inconsistency — the problem only surfaced in the subsequent `GetFreshStack` call.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:506-511` — `GetFreshStack` now checks whether `StackArn` is empty and falls back to `StackName`

**Approach rationale:** This is the minimal, localised fix. The fallback is safe because `StackName` is always set when a deployment is constructed, and using it as identifier is exactly what `StackExists` already does in the same flow.

**Alternatives considered:**
- Populating `StackArn` inside `StackExists` from `RawStack.StackId` — rejected because it mutates state as a side effect of an existence check, and would couple `StackExists` to `DeployInfo` field management

## Regression Test

**Test file:** `lib/stacks_refactored_test.go`
**Test names:**
- `TestDeployInfo_IsNewStack/REVIEW_IN_PROGRESS_with_empty_StackArn_-_new`
- `TestDeployInfo_IsNewStack/CREATE_COMPLETE_with_empty_StackArn_-_not_new`
- `TestDeployInfo_GetFreshStack_FallsBackToStackName`

**What it verifies:** With multiple stacks in the mock account and an empty `StackArn`, `IsNewStack` correctly identifies `REVIEW_IN_PROGRESS` as new and `CREATE_COMPLETE` as not-new. `GetFreshStack` successfully resolves by `StackName` when `StackArn` is empty.

**Run command:** `go test ./lib/... -run "TestDeployInfo_IsNewStack|TestDeployInfo_GetFreshStack_FallsBackToStackName" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | `GetFreshStack` falls back to `StackName` when `StackArn` is empty |
| `lib/stacks_refactored_test.go` | Added 3 regression tests covering the empty-StackArn path |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Functions that accept an identifier should validate it is non-empty before use, or document the precondition
- When a struct has both a name and an ARN field, prefer a helper that resolves the best available identifier rather than hard-coding one field

## Related

- Transit ticket: T-761
