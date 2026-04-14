# Bugfix Report: Guard Deploy Event Rendering Against nil StackEvent Fields

**Date:** 2025-07-14
**Status:** Fixed

## Description of the Issue

The `showEvents`, `showFailedEvents`, and `ReverseEvents.Less` functions in `cmd/deploy.go` dereference AWS CloudFormation `StackEvent` pointer fields (`Timestamp`, `ResourceType`, `LogicalResourceId`, `ResourceStatusReason`) without nil checks. A single malformed or partial event returned by AWS causes a nil-pointer panic that aborts the entire deploy command.

**Reproduction steps:**
1. Deploy a CloudFormation stack that produces a partial `StackEvent` with a nil `Timestamp`, `ResourceType`, or `ResourceStatusReason`
2. Fog panics with a nil pointer dereference during event rendering

**Impact:** Any deploy operation can be aborted by a single malformed CloudFormation event. The codebase already modelled nil timestamps/reasons as possible in `lib/stacks_helpers_test.go`, but the deploy event rendering paths were not guarded.

## Investigation Summary

- **Symptoms examined:** Potential nil-pointer dereference in three code paths
- **Code inspected:** `cmd/deploy.go` — `showEvents`, `showFailedEvents`, `ReverseEvents.Less`
- **Hypotheses tested:** Confirmed that all `*string` and `*time.Time` fields on `types.StackEvent` are pointer types and can be nil per the AWS SDK contract

## Discovered Root Cause

**Defect type:** Missing nil guard / defensive programming

**Why it occurred:** The original code assumed AWS always populates all StackEvent fields. The AWS SDK v2 models these as pointers, signalling that nil is a valid value, but the dereferencing code did not account for this.

**Contributing factors:** No unit-level tests existed for the event rendering paths with nil fields.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy.go` — Extracted `renderEvent` and `renderFailedEvent` helper functions that are nil-safe and independently testable
- `cmd/deploy.go` — Added `safeTimestamp` and `safeString` helpers for nil-safe dereferencing
- `cmd/deploy.go` — Made `ReverseEvents.Less` nil-safe using `safeTimestamp`
- `cmd/deploy.go` — Updated `showEvents` and `showFailedEvents` to use the new helpers

**Approach rationale:** Extracting testable helpers keeps the nil-safety logic verifiable in isolation while maintaining the original streaming output behaviour. Events with nil timestamps are skipped in rendering (since they cannot be ordered) and sort to the beginning (treated as zero-time) during sorting.

**Alternatives considered:**
- Filtering nil-timestamp events before sorting — rejected because it would lose those events entirely from failed-event tables where they might still carry useful information (if timestamp is the only nil field)

## Regression Test

**Test file:** `cmd/deploy_nil_event_fields_test.go`
**Test names:** `TestReverseEvents_NilTimestamps`, `TestReverseEvents_NilTimestampOrdering`, `TestShowEventsNilFields`, `TestShowFailedEventsNilFields`

**What it verifies:** That sorting and rendering events with any combination of nil pointer fields does not panic, and that valid timestamps maintain correct ordering.

**Run command:** `go test ./cmd/ -run 'TestReverseEvents_Nil|TestShowEventsNil|TestShowFailedEventsNil' -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy.go` | Added nil-safe helpers; refactored `showEvents`, `showFailedEvents`, and `ReverseEvents.Less` |
| `cmd/deploy_nil_event_fields_test.go` | New regression tests for nil StackEvent fields |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always nil-check AWS SDK pointer fields before dereferencing
- Extract rendering/formatting logic into testable helpers rather than inlining in I/O functions
- Add nil-field test cases when working with AWS SDK response types

## Related

- Transit ticket: T-756
