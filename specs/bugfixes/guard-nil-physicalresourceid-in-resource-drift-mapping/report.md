# Bugfix Report: Guard nil PhysicalResourceId in resource/drift mapping

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

`fog resource list` and drift-analysis logic could panic when CloudFormation returned resources with `PhysicalResourceId == nil` (common for transitional/deleted resources).

**Reproduction steps:**
1. Return a stack resource from CloudFormation with `PhysicalResourceId: nil`.
2. Run code paths through `lib.GetResources` or `cmd.separateSpecialCases`.
3. Observe panic from direct `*PhysicalResourceId` dereference.

**Impact:** CLI commands could crash during normal CloudFormation transitional states, interrupting resource listing and drift checks.

## Investigation Summary

Reviewed the panic paths and added focused regression tests to reproduce both crash sites.

- **Symptoms examined:** nil pointer panic in `resource list` and drift special-case mapping.
- **Code inspected:** `lib/resources.go`, `cmd/drift.go`, and related tests.
- **Hypotheses tested:** whether nil `PhysicalResourceId` from CloudFormation reaches those loops and causes direct pointer dereference panics.

## Discovered Root Cause

Both code paths assumed `PhysicalResourceId` was always non-nil and directly dereferenced it.

**Defect type:** Missing validation / nil-safety

**Why it occurred:** Resource loops were written for steady-state resources and did not account for transitional/deleted states where CloudFormation omits physical IDs.

**Contributing factors:** No regression tests covered nil `PhysicalResourceId` cases in these paths.

## Resolution for the Issue

**Changes made:**
- `lib/resources.go` - added guard to skip resources with nil `PhysicalResourceId`; used `aws.ToString` for safer field extraction.
- `cmd/drift.go` - added nil guards in `separateSpecialCases` before writing `logicalToPhysical` and special-case maps.
- `lib/resources_test.go` - added regression test `TestGetResourcesSkipsNilPhysicalResourceID`.
- `cmd/drift_specialcases_test.go` - added regression test `TestSeparateSpecialCasesSkipsNilPhysicalResourceID`.

**Approach rationale:** Skip entries that cannot be safely mapped because they have no physical ID, preventing panics and avoiding invalid map values.

**Alternatives considered:**
- Store empty strings for nil physical IDs - rejected because it introduces ambiguous/invalid identifiers into downstream drift/resource logic.

## Regression Test

**Test file:** `lib/resources_test.go`, `cmd/drift_specialcases_test.go`
**Test name:** `TestGetResourcesSkipsNilPhysicalResourceID`, `TestSeparateSpecialCasesSkipsNilPhysicalResourceID`

**What it verifies:** nil `PhysicalResourceId` entries are skipped safely and no panic occurs.

**Run command:** `go test ./lib -run TestGetResourcesSkipsNilPhysicalResourceID -count=1 && go test ./cmd -run TestSeparateSpecialCasesSkipsNilPhysicalResourceID -count=1`

## Affected Files

| File | Change |
|------|--------|
| `lib/resources.go` | Guard nil `PhysicalResourceId` in resource enumeration. |
| `cmd/drift.go` | Guard nil pointers before building logical/physical and special-case maps. |
| `lib/resources_test.go` | Added regression test for nil physical resource IDs. |
| `cmd/drift_specialcases_test.go` | Added regression test for nil-safe special-case mapping. |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the new tests fail on pre-fix code paths and pass after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat AWS SDK pointer fields as optional unless guaranteed otherwise by API contract.
- Add targeted regression tests whenever production panic paths are fixed.
- Prefer safe helpers (`aws.ToString`) plus explicit nil guards at map insertion points.

## Related

- Transit ticket: T-352
