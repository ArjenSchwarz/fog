# Bugfix Report: Paginate ListImports in Export Lookups

**Date:** 2026-03-20
**Status:** Investigating

## Description of the Issue

The `GetExports` function and `CfnOutput.FillImports` method call the AWS CloudFormation `ListImports` API once per export and ignore pagination. When an export is imported by many stacks (enough to exceed a single API page), only the first page of importing stacks is captured. This means `ImportedBy` is incomplete for heavily-used exports.

**Reproduction steps:**
1. Have a CloudFormation export imported by more stacks than fit in a single `ListImports` response page
2. Run `fog exports` to list exports and their importers
3. Observe that only the first page of importers is shown

**Impact:** Medium severity. Affects any account where exports are imported by enough stacks to exceed the API page size. The user sees an incomplete list of importing stacks, which could lead to incorrect decisions about whether an export can be safely removed.

## Investigation Summary

The investigation was straightforward — the bug is clearly described by the ticket name and confirmed by code review.

- **Symptoms examined:** `ImportedBy` field contains only the first page of stacks for exports with many importers
- **Code inspected:** `lib/outputs.go` (both `GetExports` and `FillImports`), `lib/interfaces.go` (`CFNListImportsAPI` interface)
- **Hypotheses tested:** Single hypothesis — missing pagination on `ListImports` calls. Confirmed by code inspection showing no `NextToken` handling.

## Discovered Root Cause

Both `GetExports` (line 63) and `FillImports` (line 119) in `lib/outputs.go` make a single `ListImports` call and use `imports.Imports` directly from the response without checking `NextToken` for additional pages.

**Defect type:** Missing pagination handling

**Why it occurred:** Pagination was implemented correctly for `DescribeStacks` (using `NewDescribeStacksPaginator`) but was not applied to the `ListImports` calls in the same file.

**Contributing factors:** The AWS SDK v2 provides both a paginator helper (`NewListImportsPaginator`) and a raw `ListImports` method. The code used the raw method without implementing pagination logic.

## Resolution for the Issue

_To be filled in after fix is implemented._

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test names:** `TestFillImportsPagination`, `TestGetExportsListImportsPagination`

**What it verifies:** That imports are collected across multiple `ListImports` response pages. Uses a mock that returns imports split across two pages with `NextToken` pagination.

**Run command:** `go test ./lib/ -run "TestFillImportsPagination|TestGetExportsListImportsPagination" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Add pagination to `ListImports` calls in `GetExports` and `FillImports` |
| `lib/outputs_test.go` | Add regression tests for paginated `ListImports` |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- When calling any AWS API that returns paginated results, always use the SDK paginator or implement a pagination loop
- Consider adding a linter rule or code review checklist item for AWS API pagination

## Related

- Transit ticket: T-499
