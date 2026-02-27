# Bugfix Report: Exports Pagination

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

The `fog resource exports` command only returned exports from stacks on the first page of AWS `DescribeStacks` results. When an account had more than ~100 CloudFormation stacks, exports from stacks beyond the first page were silently omitted.

**Reproduction steps:**
1. Have an AWS account with more than 100 CloudFormation stacks
2. Run `fog resource exports`
3. Observe that only exports from the first ~100 stacks are returned

**Impact:** Medium severity — users with many stacks would see incomplete export data without any error or warning.

## Investigation Summary

Investigated all `DescribeStacks` call sites in the `lib/` package to identify non-paginated usage.

- **Symptoms examined:** Missing stacks beyond the first page of `DescribeStacks` results
- **Code inspected:** `lib/outputs.go`, `lib/resources.go`, `lib/stacks.go`, `cmd/exports.go`
- **Hypotheses tested:** The same non-paginated `DescribeStacks` bug previously fixed in `GetResources` (commit c8e4714) also existed in `GetExports`

## Discovered Root Cause

`GetExports` in `lib/outputs.go` called `svc.DescribeStacks()` directly without using the AWS SDK paginator, only processing the first page of results (typically up to 100 stacks).

**Defect type:** Missing pagination

**Why it occurred:** When `GetResources` was fixed to use `NewDescribeStacksPaginator` in PR #81, the same pattern in `GetExports` was not updated.

**Contributing factors:** Both functions had identical non-paginated code, but only one was fixed.

## Resolution for the Issue

**Changes made:**
- `lib/outputs.go:36-48` — Replaced direct `DescribeStacks` call with `NewDescribeStacksPaginator` loop, accumulating stacks from all pages before processing exports

**Approach rationale:** Mirrors the exact pagination pattern already used in `GetResources` and `GetCfnStacks`, ensuring consistency across the codebase.

**Alternatives considered:**
- Manual NextToken loop — Not chosen because the SDK paginator is the idiomatic approach and is already used elsewhere in this codebase

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test name:** `TestGetExportsPagination`

**What it verifies:** That exports from stacks across multiple `DescribeStacks` response pages are all returned. Uses a paginating mock with two pages, each containing one stack with one export, and asserts both exports appear in the result.

**Run command:** `go test ./lib/ -run TestGetExportsPagination -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Replaced direct `DescribeStacks` call with paginator loop |
| `lib/outputs_test.go` | Added `paginatingMockCFNClient` and `TestGetExportsPagination` |
| `specs/bugfixes/exports-pagination/report.md` | Bugfix report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- When fixing a pagination bug in one function, audit all similar call sites for the same pattern
- Consider adding a linter rule or code review checklist item for raw `DescribeStacks` calls without pagination

## Related

- Commit c8e4714: "Fix GetResources to paginate DescribeStacks results (#81)" — the same fix applied to `GetResources`
- Transit T-130
