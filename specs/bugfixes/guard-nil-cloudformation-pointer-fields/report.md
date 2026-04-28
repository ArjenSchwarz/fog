# Bugfix Report: Guard nil CloudFormation pointer fields

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

Several CloudFormation listing helpers dereference AWS SDK pointer fields without
checking for nil first. When AWS returns partial data, or tests inject malformed
stack/output records, the code panics instead of degrading gracefully or
returning a useful error.

**Reproduction steps:**
1. Call `GetCfnStacks`, `GetExports`, or `GetResources` with paginated
   `DescribeStacks` results that include malformed stack/output entries.
2. Include entries with nil `StackName`, nil `StackId`, nil `OutputKey`, or nil
   `OutputValue`.
3. Observe a panic caused by direct pointer dereferences in the listing code.

**Impact:** Stack, export, and resource listing commands can crash on malformed
or partial CloudFormation responses instead of returning usable data or a
contextual error.

## Investigation Summary

The investigation focused on the stack, export, and resource listing helpers
named in Transit ticket T-1026.

- **Symptoms examined:** panics while iterating paginated `DescribeStacks`
  results containing nil pointer fields.
- **Code inspected:** `lib/stacks.go`, `lib/outputs.go`, `lib/resources.go`,
  and their associated unit tests.
- **Hypotheses tested:** whether each code path should skip malformed entries or
  return contextual errors, and whether paginated tests can reproduce the bug
  consistently.

## Discovered Root Cause

The AWS SDK models CloudFormation fields like `StackName`, `StackId`,
`OutputKey`, and `OutputValue` as pointers, but the affected functions treat
them as always present and dereference them directly.

**Defect type:** Missing validation

**Why it occurred:** The listing helpers assume normal AWS responses and do not
guard malformed or partial objects before filtering, constructing result
records, or formatting error messages.

**Contributing factors:** Existing tests covered happy paths and pagination, but
did not cover nil pointer fields inside paginated responses.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:232-269` - validate stack names and stack IDs before building
  `CfnStack` results, returning contextual errors for malformed stack records.
- `lib/outputs.go:105-139` - use nil-safe stack/export name handling and skip
  outputs missing required key/value fields instead of panicking.
- `lib/resources.go:44-97` - skip stacks without usable names during filtering
  and use safe stack labels for resource lookups and error messages.
- `lib/getcfnstacks_test.go` - added paginated regression coverage for nil
  `StackName` and nil `StackId`.
- `lib/outputs_test.go` - added paginated regression coverage for nil
  `StackName`, nil `OutputKey`, and nil `OutputValue`.
- `lib/resources_test.go` - added wildcard-filter regression coverage for nil
  `StackName`.

**Approach rationale:** `GetCfnStacks` now returns explicit errors because stack
name and stack ID are required to build correct `CfnStack` results and support
downstream event lookups. `GetExports` and `GetResources` skip malformed items
because those listings can still return useful results from the remaining valid
entries.

**Alternatives considered:**
- Return errors for every malformed stack/output entry - likely too disruptive
  for export/resource listings that can safely skip incomplete records.

## Regression Test

**Test file:** `lib/getcfnstacks_test.go`, `lib/outputs_test.go`,
`lib/resources_test.go`
**Test name:** `TestGetCfnStacks_ReturnsErrorForMissingStackNameInPaginatedResults`,
`TestGetCfnStacks_ReturnsErrorForMissingStackIDInPaginatedResults`,
`TestGetExports_SkipsStacksWithoutStackNameInPaginatedResults`,
`TestGetExports_SkipsOutputsMissingKeyOrValueInPaginatedResults`,
`TestGetResourcesSkipsStacksWithoutNameDuringWildcardFiltering`

**What it verifies:** Malformed paginated stack/output records do not panic;
callers either receive a contextual error (`GetCfnStacks`) or malformed entries
are skipped (`GetExports`, `GetResources`).

**Run command:** `go test ./lib -run 'TestGetCfnStacks_ReturnsErrorForMissingStack(Name|ID)InPaginatedResults|TestGetExports_SkipsStacksWithoutStackNameInPaginatedResults|TestGetExports_SkipsOutputsMissingKeyOrValueInPaginatedResults|TestGetResourcesSkipsStacksWithoutNameDuringWildcardFiltering'`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added contextual validation for missing stack name and stack ID |
| `lib/outputs.go` | Guarded stack/output pointers and skip malformed export entries |
| `lib/resources.go` | Guarded wildcard filtering and stack resource lookups |
| `lib/getcfnstacks_test.go` | Added regression coverage for missing stack identifiers |
| `lib/outputs_test.go` | Added regression coverage for malformed export records |
| `lib/resources_test.go` | Added regression coverage for nil stack names during wildcard filtering |

## Verification

**Automated:**
- [x] Regression test fails before fix
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Reviewed the pre-fix panic traces and post-fix behavior to confirm malformed
  paginated entries are now either skipped or converted into contextual errors.

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer `aws.ToString` or explicit nil guards whenever AWS SDK response fields
  are pointer-typed.
- Add regression tests for malformed paginated SDK responses when touching list
  aggregation code.
- Favor contextual errors for malformed entries that are required for correct
  downstream behavior, and skipping for optional/incomplete list items.

## Related

- Transit ticket `T-1026`
