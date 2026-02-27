# Bugfix Report: Paginate DeployInfo.GetEvents

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

`DeployInfo.GetEvents()` called `DescribeStackEvents` only once without handling pagination. AWS returns a maximum of ~100 events per page, so stacks with more than 100 events would silently return incomplete results.

**Reproduction steps:**
1. Deploy a CloudFormation stack with many resource changes (>100 events)
2. Run any fog command that calls `DeployInfo.GetEvents` (e.g., `fog deploy` failure reporting, execution times)
3. Observe that only the first page of events is returned

**Impact:** Medium — affects any stack with more than ~100 events. Execution time reports and failure diagnostics would be incomplete, potentially missing the actual failed resource or showing incorrect timing data.

## Investigation Summary

- **Symptoms examined:** `DeployInfo.GetEvents` returns only one page of events
- **Code inspected:** `lib/stacks.go` — both `DeployInfo.GetEvents` and `CfnStack.GetEvents`
- **Hypotheses tested:** Compared the two `GetEvents` implementations; `CfnStack.GetEvents` already uses `fetchAllStackEvents` with proper pagination via `NewDescribeStackEventsPaginator`

## Discovered Root Cause

`DeployInfo.GetEvents()` made a single `DescribeStackEvents` call and returned `resp.StackEvents` directly, ignoring the `NextToken` field in the response.

**Defect type:** Missing pagination loop

**Why it occurred:** The method was likely written before pagination was needed (small stacks) or was an oversight when the corresponding `CfnStack.GetEvents` method was refactored to use pagination.

**Contributing factors:** `DeployInfo.GetEvents` uses the `CloudFormationDescribeStackEventsAPI` interface (not the concrete `*cloudformation.Client`), so it cannot use the SDK's built-in paginator which requires the concrete client type. This meant manual `NextToken` looping was needed but was never implemented.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:507-522` — Replaced single API call with a `NextToken` pagination loop that accumulates all events across pages

**Approach rationale:** Manual `NextToken` looping is the standard pattern when working with an interface rather than a concrete SDK client. This matches the pagination style already used by `GetChangeset` in the same file.

**Alternatives considered:**
- Using `fetchAllStackEvents` — requires `*cloudformation.Client` instead of the interface, would break the existing API contract and all callers
- Changing the interface to accept `*cloudformation.Client` — unnecessary coupling, breaks existing mock-based tests

## Regression Test

**Test file:** `lib/stacks_test.go`
**Test name:** `TestDeployInfo_GetEvents`

**What it verifies:** Tests single-page, multi-page (3 pages), empty, and error scenarios for `GetEvents`. The multi-page test verifies that all 5 events across 3 pages are collected.

**Run command:** `go test ./lib -v -run TestDeployInfo_GetEvents`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added `NextToken` pagination loop to `DeployInfo.GetEvents` |
| `lib/stacks_test.go` | Upgraded mock to support multi-page responses; added `TestDeployInfo_GetEvents` test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the existing `TestDeployInfo_GetExecutionTimes` test (which calls `GetEvents` internally) still passes with the updated mock

## Prevention

**Recommendations to avoid similar bugs:**
- When implementing AWS API calls, always check if the response includes a `NextToken` field and handle pagination
- Consider adding a lint rule or code review checklist item for AWS SDK pagination

## Related

- Transit ticket: T-201
