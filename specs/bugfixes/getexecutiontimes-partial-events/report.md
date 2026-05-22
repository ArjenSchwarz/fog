# Bugfix Report: GetExecutionTimes Panics on Partial CloudFormation Events

**Date:** 2026-05-22
**Status:** Fixed

## Description of the Issue

`lib.DeployInfo.GetExecutionTimes` (`lib/stacks.go`) dereferenced AWS CloudFormation `StackEvent` pointer fields directly without nil checks:

- `event.Timestamp.After(...)`
- `*event.ResourceType`
- `*event.LogicalResourceId`
- `*event.Timestamp`

Other event-rendering and reporting paths had already been hardened (skipping nil timestamps, using `aws.ToString` for optional SDK fields — see T-756/T-799), but `GetExecutionTimes` was left unguarded. A single partial or malformed event returned by `DescribeStackEvents` (or a mock exercising nil fields) caused a nil-pointer panic.

**Reproduction steps:**
1. Call `GetExecutionTimes` with a stack whose `DescribeStackEvents` response contains an event with a nil `Timestamp`, `ResourceType`, or `LogicalResourceId`.
2. Fog panics with `runtime error: invalid memory address or nil pointer dereference` at `lib/stacks.go` (the `event.Timestamp.After(...)` call).

**Impact:** Any code path that builds execution-timing data (e.g. deploy timing reporting) can be aborted by one malformed CloudFormation event. The AWS SDK v2 models these fields as pointers, so nil is a valid value the function must tolerate.

## Investigation Summary

- **Symptoms examined:** Nil-pointer dereference in `GetExecutionTimes` when iterating events.
- **Code inspected:** `lib/stacks.go` — `GetExecutionTimes` (the loop dereferencing event fields). Compared against the already-hardened paths in the same file: `generateResourceEventName`, `createNewResourceEvent`, `isStackLevelEvent`, and the `safeEventTimestamp` helper used by `ReverseEvents.Less`.
- **Hypotheses tested:** Confirmed via a new failing test that `event.Timestamp` is the first field dereferenced and panics when nil. Confirmed `ResourceType` and `LogicalResourceId` are also `*string` and can be nil.

## Discovered Root Cause

**Defect type:** Missing nil guard / defensive programming.

**Why it occurred:** The function predates the project-wide hardening of event handling and was not updated when the other paths adopted nil-safe access. It assumed AWS always populates `Timestamp`, `ResourceType`, and `LogicalResourceId`.

**Contributing factors:** No regression test exercised `GetExecutionTimes` with nil event fields; existing tests only used fully-populated events.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go` (`GetExecutionTimes`) — Skip events with a nil `Timestamp` (a nil timestamp cannot be compared against the changeset creation time and carries no execution time to record). Use `aws.ToString` for `ResourceType` and `LogicalResourceId` so nil pointers resolve to empty strings (a safe label) instead of panicking. `*event.Timestamp` is now only dereferenced after the nil guard. Guard `deployment.Changeset` (a `*ChangesetInfo` pointer): compute `changesetCreationTime` once before the loop, falling back to the zero `time.Time` when `Changeset == nil` so the loop proceeds safely.

**Approach rationale:** This mirrors the established conventions already used elsewhere in `lib/stacks.go` (`aws.ToString` for optional strings, explicit nil checks before dereferencing `Timestamp`, as in `createNewResourceEvent`). It is a minimal change confined to the buggy loop, keeps valid events fully intact, and safely labels partial events rather than dropping the rest of the batch.

**Alternatives considered:**
- Treating a nil `Timestamp` as the zero time via the existing `safeEventTimestamp` helper rather than skipping — rejected because the zero time would never satisfy `.After(changeset.CreationTime)`, so the event is filtered out anyway, but skipping makes the intent explicit and avoids recording a meaningless zero timestamp.
- Returning an error on malformed events — rejected because partial events should not abort timing collection for the valid events in the same response.

## Regression Test

**Test file:** `lib/stacks_test.go`
**Test names:** `TestDeployInfo_GetExecutionTimes_NilFields`, `TestDeployInfo_GetExecutionTimes_NilChangeset`

**What it verifies:** That `GetExecutionTimes` does not panic when the event slice contains nil `Timestamp`, nil `ResourceType`, nil `LogicalResourceId`, and all-nil combinations, while still returning correct timing data for valid events and safely labelling events with nil string fields. `TestDeployInfo_GetExecutionTimes_NilChangeset` additionally confirms that a nil `Changeset` does not panic, with all timestamped events returned (zero `time.Time` fallback means every event is "after" it).

**Run command:** `go test ./lib/ -run 'TestDeployInfo_GetExecutionTimes' -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Guard `GetExecutionTimes` against nil `Timestamp` and nil `Changeset`; use `aws.ToString` for `ResourceType`/`LogicalResourceId` |
| `lib/stacks_test.go` | New regression tests `TestDeployInfo_GetExecutionTimes_NilFields`, `TestDeployInfo_GetExecutionTimes_NilChangeset` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`make lint` / `golangci-lint run` — 0 issues)

**Manual verification:**
- Confirmed the new test panics before the fix (at `lib/stacks.go` `event.Timestamp.After(...)`) and passes after.

## Prevention

**Recommendations to avoid similar bugs:**
- Always nil-check or use `aws.ToString` / `safeEventTimestamp` on AWS SDK pointer fields before dereferencing.
- When hardening one event-handling path, audit sibling functions in the same file for the same pattern.
- Add nil-field test cases when working with AWS SDK response types.

## Related

- Transit ticket: T-1087
- Related prior fixes: T-756 (deploy event rendering nil guards), T-799 (`processStackEvents`/`generateResourceEventName`/`ReverseEvents` nil guards)
