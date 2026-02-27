# Bugfix Report: Exports Command Pagination

**Date:** 2025-02-27
**Status:** Fixed

## Description of the Issue

The `GetExports` function in `lib/outputs.go` called `DescribeStacks` once without pagination, so only the first page of results (up to 100 stacks) was processed. Exports from stacks beyond the first page were silently omitted.

**Reproduction steps:**
1. Have more than 100 CloudFormation stacks in an account/region
2. Run `fog exports` without a stack name filter
3. Observe that exports from stacks beyond the first 100 are missing

**Impact:** Medium — users with large AWS accounts would get incomplete export data with no warning.

## Investigation Summary

- **Symptoms examined:** `GetExports` only processes stacks from a single `DescribeStacks` call
- **Code inspected:** `lib/outputs.go` (GetExports), `lib/resources.go` (GetResources for comparison), `lib/interfaces.go` (API interfaces)
- **Hypotheses tested:** Confirmed that `GetResources` in the same codebase already uses `NewDescribeStacksPaginator` correctly, while `GetExports` does not

## Discovered Root Cause

`GetExports` called `svc.DescribeStacks()` directly instead of using the AWS SDK paginator. The AWS `DescribeStacks` API returns at most 100 stacks per page and includes a `NextToken` for subsequent pages. Without iterating through pages, only the first batch of stacks was processed.

**Defect type:** Missing pagination

**Why it occurred:** The original implementation did not account for paginated API responses.

**Contributing factors:** The `GetResources` function was later updated to use pagination but `GetExports` was not updated at the same time.

## Resolution for the Issue

**Changes made:**
- `lib/outputs.go:36-49` — Replaced single `svc.DescribeStacks()` call with `cloudformation.NewDescribeStacksPaginator` loop that collects all stacks before filtering outputs

**Approach rationale:** Follows the exact same paginator pattern already used by `GetResources` in `lib/resources.go`, ensuring consistency across the codebase.

**Alternatives considered:**
- Manual NextToken loop — More verbose and error-prone than using the SDK's built-in paginator

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test name:** `TestGetExports_Pagination`

**What it verifies:** That exports from stacks across three DescribeStacks pages are all collected and returned, not just the first page. Also verifies import information is correctly populated for paginated results.

**Run command:** `go test ./lib/ -run TestGetExports_Pagination -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Replaced single DescribeStacks call with paginator loop |
| `lib/outputs_test.go` | Added `TestGetExports_Pagination` regression test and supporting `paginatingExportsMockClient` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- When calling AWS APIs that return paginated results, always use the SDK paginator or handle NextToken
- Review other AWS API calls in the codebase for similar missing pagination

## Related

- Transit ticket: T-128
- Similar fix pattern: `lib/resources.go` GetResources already uses paginator correctly
