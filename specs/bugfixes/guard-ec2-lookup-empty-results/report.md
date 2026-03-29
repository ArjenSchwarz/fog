# Bugfix Report: Guard EC2 Lookup Helpers Against Empty Describe Results

**Date:** 2026-03-29
**Status:** Fixed

## Description of the Issue

`GetNacl` and `GetRouteTable` in `lib/ec2.go` directly index `result.NetworkAcls[0]` and `result.RouteTables[0]` after a successful AWS API call. If AWS returns a valid response with an empty list (e.g., resource deleted between filter and describe, eventual consistency, or a filtered response with no matches), fog panics with an `index out of range [0] with length 0` runtime error.

**Reproduction steps:**
1. Call `GetNacl` or `GetRouteTable` with an ID that produces a successful API response but an empty result list
2. Observe a panic: `runtime error: index out of range [0] with length 0`

**Impact:** Any caller of these helpers crashes the process instead of receiving a handleable error.

## Investigation Summary

- **Symptoms examined:** Both functions unconditionally index `[0]` on the result slice after checking only for API errors
- **Code inspected:** `lib/ec2.go` lines 20 and 33
- **Hypotheses tested:** The only hypothesis was missing length guards — confirmed immediately by the code

## Discovered Root Cause

Both `GetNacl` and `GetRouteTable` assume that a nil error from the AWS API guarantees at least one element in the response slice. This assumption is incorrect — AWS can return a successful response with an empty list.

**Defect type:** Missing validation (bounds check)

**Why it occurred:** The original code assumed the AWS SDK would always return an error when no matching resource exists. In practice, some describe calls can return an empty list without an error.

**Contributing factors:** No test coverage for the empty-result path.

## Resolution for the Issue

**Changes made:**
- `lib/ec2.go:19-21` — Added `len(result.NetworkAcls) == 0` guard before indexing, returning a descriptive error
- `lib/ec2.go:32-34` — Added `len(result.RouteTables) == 0` guard before indexing, returning a descriptive error

**Approach rationale:** A simple length check before indexing is the minimal, idiomatic Go fix. The error messages include the requested ID to aid debugging.

**Alternatives considered:**
- Returning a sentinel error type — overkill for this use case; a formatted error string is sufficient

## Regression Test

**Test file:** `lib/ec2_test.go`
**Test names:**
- `TestGetNacl/Test_Get_Nacl_Empty_Result`
- `TestGetNacl/Test_Get_Nacl_Nil_Result`
- `TestGetRouteTable/Error_-_empty_result_list`
- `TestGetRouteTable/Error_-_nil_result_list`

**What it verifies:** When the AWS API returns a successful response with an empty (or nil) slice, the functions return an error instead of panicking.

**Run command:** `go test ./lib -run "TestGetNacl$|TestGetRouteTable$" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/ec2.go` | Added empty-slice guard in `GetNacl` and `GetRouteTable` |
| `lib/ec2_test.go` | Added 4 regression tests for empty/nil result lists |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always check slice length before indexing AWS SDK response slices, even after a nil-error check
- Consider adding a project-wide lint or review checklist item for unchecked slice indexing on API responses

## Related

- Transit ticket: T-619
