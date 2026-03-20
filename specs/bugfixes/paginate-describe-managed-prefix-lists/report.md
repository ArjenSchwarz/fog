# Bugfix Report: Paginate DescribeManagedPrefixLists for Drift Filtering

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`GetManagedPrefixLists` in `lib/ec2.go` calls the AWS EC2 `DescribeManagedPrefixLists` API exactly once and returns whatever is in the first response page. If the account/region has more managed prefix lists than fit in a single API response, subsequent pages are silently ignored.

**Reproduction steps:**
1. Have an AWS account with enough managed prefix lists to require pagination (or an API that returns a NextToken)
2. Run a drift check that invokes `checkRouteTableRoutes` in `cmd/drift.go`
3. Observe that routes referencing prefix lists from page 2+ are falsely reported as unmanaged drift

**Impact:** Drift checks produce false positives for route tables whose routes reference AWS-managed prefix lists that happen to appear on subsequent API pages. This leads to incorrect drift reports.

## Investigation Summary

- **Symptoms examined:** Missing prefix lists from drift filtering causes false drift reports on routes
- **Code inspected:** `lib/ec2.go:GetManagedPrefixLists`, `cmd/drift.go:checkRouteTableRoutes`, existing pagination patterns in `lib/stacks.go`, `lib/identitycenter.go`, `lib/outputs.go`
- **Hypotheses tested:** Single hypothesis — the function lacks pagination. Confirmed by code inspection.

## Discovered Root Cause

`GetManagedPrefixLists` makes a single API call without handling the `NextToken` in the response.

**Defect type:** Missing pagination

**Why it occurred:** The function was written to make a single API call, likely because the initial implementation assumed a small number of prefix lists. The `DescribeManagedPrefixListsOutput.NextToken` field was not checked.

**Contributing factors:** AWS SDK v2 provides a paginator (`ec2.NewDescribeManagedPrefixListsPaginator`) but it was not used, unlike other similar functions in the codebase (e.g., `GetAllStacks`, `GetPermissionSets`).

## Resolution for the Issue

**Changes made:**
- `lib/ec2.go:37-44` - Replaced single API call with `ec2.NewDescribeManagedPrefixListsPaginator` loop, accumulating prefix lists across all pages

**Approach rationale:** Using the SDK's built-in paginator is consistent with how other paginated calls are handled in this codebase (see `lib/stacks.go`, `lib/identitycenter.go`, `lib/outputs.go`). It handles NextToken management automatically and is less error-prone than a manual loop.

**Alternatives considered:**
- Manual NextToken loop — More code, more room for off-by-one or token-handling errors. Rejected in favour of the SDK paginator.

## Regression Test

**Test file:** `lib/ec2_test.go`
**Test name:** `TestGetManagedPrefixLists/paginated_results_returns_all_prefix_lists` and `TestGetManagedPrefixLists/error_on_second_page_returns_error`

**What it verifies:** The first test verifies that prefix lists from multiple API pages are all returned. The second test verifies that an error on a subsequent page is properly propagated.

**Run command:** `go test ./lib/ -run TestGetManagedPrefixLists -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/ec2.go` | Use paginator in `GetManagedPrefixLists` |
| `lib/ec2_test.go` | Add pagination and second-page-error regression tests |
| `specs/bugfixes/paginate-describe-managed-prefix-lists/report.md` | Bugfix report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When calling any AWS `Describe*` or `List*` API, always check whether the response supports pagination (NextToken) and use the SDK paginator
- Consider a linter rule or code review checklist item for unpaginated AWS API calls

## Related

- Transit ticket: T-388
