# Bugfix Report: No-Changes Deployment Output Loses Cached Stack Details

**Date:** 2026-04-20
**Status:** Fixed
**Transit:** T-832

## Description of the Issue

When CloudFormation reports that a changeset contains no changes, `fog deploy` falls back to `outputNoChangesResult(deployment)` which renders a "Stack Information" table with stack status and last-updated time sourced from `deployment.RawStack`. For an existing stack, `RawStack` was `nil` at this point, so the output showed a blank `Status` column and `Last Updated: N/A`, even though the deployment pipeline had already fetched the stack during readiness checks.

**Reproduction steps:**
1. Deploy an existing stack (e.g. `CREATE_COMPLETE`) with a template that yields no changes.
2. Observe the final "Stack Information" table on stdout.
3. The `Status` cell is empty and `Last Updated` shows `N/A`.

**Impact:** Cosmetic but misleading. Users running no-op deployments saw incomplete output that made it look like stack metadata had been lost or the stack had never been fetched. No data loss, no behaviour change for actual deployments.

## Investigation Summary

- **Symptoms examined:** `outputNoChangesResult` rendering blank `Status` and `Last Updated: N/A` for existing stacks.
- **Code inspected:** `cmd/deploy_output.go` (`outputNoChangesResult`), `cmd/deploy.go` (`createChangeset` no-changes branch), `cmd/deploy_helpers.go` (`prepareDeployment`), `lib/stacks.go` (`IsNewStack`, `StackExists`, `GetFreshStack`).
- **Hypotheses tested:**
  - `StackExists` not caching on success — ruled out: `StackExists` correctly writes to `deployment.RawStack` via a pointer argument (verified by `TestStackExists_CachesRawStack`).
  - `outputNoChangesResult` dereferencing the wrong field — ruled out: the function reads the same `RawStack` field `StackExists` writes.

## Discovered Root Cause

`IsNewStack` was declared with a **value receiver**:

```go
func (deployment DeployInfo) IsNewStack(ctx context.Context, svc CloudFormationDescribeStacksAPI) bool {
    stackExists := StackExists(ctx, &deployment, svc)
    ...
}
```

`&deployment` inside the method points to the method's local copy of the receiver, not the caller's `DeployInfo`. `StackExists` then correctly caches the stack onto that copy via `deployment.RawStack = &stack`, but the copy is discarded when `IsNewStack` returns. Callers such as `prepareDeployment` retained a `DeployInfo` with `RawStack == nil`.

`prepareDeployment` then passes this `DeployInfo` all the way through to `createChangeset`, which on a no-change changeset calls `outputNoChangesResult(deployment)` — where `RawStack == nil` drives the blank status and `N/A` fallback.

**Defect type:** Value receiver mutating a copy (lost side-effect).

**Why it occurred:** When `StackExists` was converted to cache `RawStack` via a pointer parameter (T-142), the sibling `IsNewStack` that calls it kept its value receiver. The caching side-effect silently stopped propagating to callers without any compile-time signal.

**Contributing factors:** Existing tests exercised `StackExists` directly with a `*DeployInfo` and exercised `IsNewStack` only for its boolean return value. Nothing asserted that `RawStack` was preserved on the caller after `IsNewStack`.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:307` — `IsNewStack` now has a pointer receiver (`*DeployInfo`) and passes the receiver directly into `StackExists`. The `RawStack` cached by `StackExists` now persists on the caller's `DeployInfo`.

**Approach rationale:** This is the minimal, local fix and mirrors the receiver pattern already used by the other stateful methods in this file (`GetStack`, `GetFreshStack`, `LoadDeploymentFile`). Both existing call sites (`prepareDeployment` in `cmd/deploy_helpers.go` and the unit test) already use addressable `DeployInfo` values, so the receiver change compiles and runs unchanged.

**Alternatives considered:**
- Have `prepareDeployment` explicitly call `StackExists` after `IsNewStack` to populate the cache — rejected as redundant and error-prone; it leaves the value-receiver trap in place for future callers.
- Keep the value receiver and have `outputNoChangesResult` refetch the stack from AWS when `RawStack` is nil — rejected because it adds an unnecessary API call to the no-changes path and doesn't fix the underlying lost-cache pattern.

## Regression Test

**Test file:** `lib/stacks_refactored_test.go`
**Test name:** `TestDeployInfo_IsNewStack_CachesRawStack`

**What it verifies:**
- After `IsNewStack` is called on an existing `CREATE_COMPLETE` stack, the caller's `DeployInfo.RawStack` is non-nil and reflects the fetched stack.
- After `IsNewStack` is called on a missing stack, the caller's `RawStack` remains nil.

**Run command:** `go test ./lib/ -run "TestDeployInfo_IsNewStack_CachesRawStack" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | `IsNewStack` converted from value to pointer receiver; passes the receiver pointer directly to `StackExists` so the cached `RawStack` persists on the caller. |
| `lib/stacks_refactored_test.go` | Added `TestDeployInfo_IsNewStack_CachesRawStack` covering both the existing-stack (cache populated) and missing-stack (cache remains nil) paths. |

## Verification

**Automated:**
- [x] Regression test passes (`go test ./lib/ -run TestDeployInfo_IsNewStack_CachesRawStack -v`)
- [x] Full test suite passes (`go test ./...`)
- [x] Integration tests pass (`INTEGRATION=1 go test ./...`)
- [x] `go vet ./...` clean
- [x] `golangci-lint run ./...` clean

## Prevention

**Recommendations to avoid similar bugs:**
- Methods that depend on mutating their receiver (including indirect mutation via helpers that take `&deployment`) must use pointer receivers. Consider a lint rule or code review checklist item.
- Tests for methods with mutation semantics should assert both the return value and any receiver-state changes, not just the boolean/result output.

## Related

- Transit ticket: T-832
- Related previous fix: T-142 (`StackExists` caching), which fixed the caching inside `StackExists` but didn't cover the `IsNewStack` flow.
