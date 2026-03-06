# Bugfix Report: Paginate ListExports in Drift Special-Cases

**Date:** 2026-03-06
**Status:** Fixed

## Description of the Issue

The `separateSpecialCases` function in `cmd/drift.go` called `ListExports` once without pagination. The AWS CloudFormation `ListExports` API returns a maximum of 100 exports per page. For accounts with more than 100 exports, subsequent pages were silently dropped, causing `!ImportValue` resolution to miss entries and drift reports to be incomplete.

**Reproduction steps:**
1. Have an AWS account with more than 100 CloudFormation exports
2. Run `fog drift` on a stack that uses `!ImportValue` references to exports beyond the first page
3. Observe that those import values are not resolved, leading to incomplete drift reports

**Impact:** Medium severity. Accounts with large numbers of CloudFormation exports would silently produce incomplete drift reports. The unmanaged resource detection logic relies on `logicalToPhysical` being complete, so missing exports could cause false positives.

## Investigation Summary

- **Symptoms examined:** Single `ListExports` call without pagination in drift special-cases code
- **Code inspected:** `cmd/drift.go` (line 261), `cmd/drift_specialcases_test.go`, `lib/interfaces.go` (CloudFormationListExportsAPI interface)
- **Hypotheses tested:** Confirmed that the AWS SDK provides `NewListExportsPaginator` and that the existing `CloudFormationListExportsAPI` interface satisfies the `ListExportsAPIClient` interface required by the paginator

## Discovered Root Cause

The `ListExports` call on line 261 of `cmd/drift.go` was a single non-paginated call. It did not check `NextToken` in the response nor issue follow-up requests for additional pages.

**Defect type:** Missing pagination

**Why it occurred:** The original implementation assumed all exports would fit in a single API response.

**Contributing factors:** The AWS API defaults to returning up to 100 items, which is sufficient for small accounts but not for larger ones.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go:261-271` - Replaced single `ListExports` call with `NewListExportsPaginator` loop that iterates all pages, breaking on error with a warning (preserving the non-fatal error behaviour)

**Approach rationale:** Using the SDK's built-in paginator is the idiomatic approach and matches the pattern already used elsewhere in the codebase (e.g., `DescribeStacksPaginator` in `lib/stacks.go` and `lib/resources.go`).

**Alternatives considered:**
- Manual NextToken loop - More code and more error-prone than the SDK paginator
- Increasing page size - Not supported by the ListExports API

## Regression Test

**Test file:** `cmd/drift_specialcases_test.go`
**Test name:** `TestSeparateSpecialCasesPaginatesListExports`

**What it verifies:** That exports spread across three pages (with NextToken chaining) are all collected into the `logicalToPhysical` map.

**Run command:** `go test ./cmd -run TestSeparateSpecialCasesPaginatesListExports -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Replaced single ListExports call with paginator loop |
| `cmd/drift_specialcases_test.go` | Added multi-page mock support and pagination regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When calling any AWS API that supports pagination, always use the SDK paginator or explicitly handle `NextToken`
- Consider adding a project lint rule or code review checklist item for unpaginated AWS API calls
- The codebase already uses paginators in `lib/` — follow those patterns consistently in `cmd/` as well

## Related

- Transit ticket: T-356
