# Bugfix Report: Paginate SSO Permission Sets

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

`GetPermissionSetArns` and `GetAccountAssignmentArnsForPermissionSet` in `lib/identitycenter.go` each made a single API call without following pagination tokens. AWS SSO Admin APIs (`ListPermissionSets` and `ListAccountAssignments`) are paginated, so organisations with more permission sets or assignments than fit in a single response page would silently miss results.

**Reproduction steps:**
1. Have an AWS organisation with more SSO permission sets or account assignments than the API page size (default 100).
2. Run a fog command that calls `GetPermissionSetArns` or `GetAssignmentArns`.
3. Observe that only the first page of results is returned.

**Impact:** Organisations with many permission sets or assignments would get incomplete data, leading to missing resources in reports and drift detection.

## Investigation Summary

- **Symptoms examined:** `GetPermissionSetArns` calls `ListPermissionSets` once; `GetAccountAssignmentArnsForPermissionSet` calls `ListAccountAssignments` once per account. Neither follows `NextToken`.
- **Code inspected:** `lib/identitycenter.go`, `lib/interfaces.go`, `lib/identitycenter_test.go`. Noted that `GetAccountIDs` already correctly uses `organizations.NewListAccountsPaginator`.
- **Hypotheses tested:** Confirmed the AWS SDK v2 provides `ssoadmin.NewListPermissionSetsPaginator` and `ssoadmin.NewListAccountAssignmentsPaginator`, and that the existing interfaces satisfy the paginator client requirements.

## Discovered Root Cause

**Defect type:** Missing pagination

**Why it occurred:** The original implementation made single API calls and collected only the first page of results, ignoring `NextToken` in the response. This was an oversight during initial implementation — the pattern was already correctly applied for `ListAccounts` but not for the SSO Admin APIs.

**Contributing factors:** Single-page responses in small test environments masked the bug.

## Resolution for the Issue

**Changes made:**
- `lib/identitycenter.go:26-40` — Replaced single `ListPermissionSets` call with `ssoadmin.NewListPermissionSetsPaginator` loop
- `lib/identitycenter.go:99-116` — Replaced single `ListAccountAssignments` call with `ssoadmin.NewListAccountAssignmentsPaginator` loop

**Approach rationale:** The AWS SDK v2 provides built-in paginators that handle `NextToken` management. Using them is consistent with the existing `GetAccountIDs` implementation and is the idiomatic approach.

**Alternatives considered:**
- Manual `NextToken` loop — More code, more error-prone, and inconsistent with existing patterns.

## Regression Test

**Test file:** `lib/identitycenter_test.go`
**Test names:** `TestGetPermissionSetArnsPagination`, `TestGetAccountAssignmentArnsForPermissionSetPagination`

**What they verify:** Both tests configure mock clients to return two pages of results (with `NextToken` set on the first page) and assert that all results from both pages are collected.

**Run command:** `go test ./lib/ -run "Pagination" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/identitycenter.go` | Use SDK paginators for `ListPermissionSets` and `ListAccountAssignments` |
| `lib/identitycenter_test.go` | Update mock to support multi-page responses; add pagination regression tests |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When adding new AWS API calls, always check whether the API is paginated and use the SDK paginator if so.
- Consider adding a code review checklist item for pagination handling.

## Related

- Transit ticket: T-232
