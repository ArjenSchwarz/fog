# Bugfix Report: Parameterized NACL protocol and egress values may not resolve in drift checks

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

Network ACL drift checks converted CloudFormation template entries into EC2 SDK entries without resolving `Ref` values for some scalar properties. When `Protocol`, `Egress`, or similar direct NACL properties were parameterized, the helper functions silently returned zero values instead of the resolved parameter value.

**Reproduction steps:**
1. Define an `AWS::EC2::NetworkAclEntry` whose `Protocol`, `Egress`, or `RuleAction` property is set via `{"Ref": "ParamName"}`.
2. Run the template-to-EC2 conversion used by drift detection.
3. Observe that unresolved values fall back to `""`, `false`, or `allow` instead of the parameter value.

**Impact:** Drift checks can misclassify parameterized NACL entries, hiding real drift or reporting incorrect differences.

## Investigation Summary

A targeted review of the NACL conversion helpers in `lib/template.go` showed that `extractRuleNumber` and `extractCidrBlock` already resolved `Ref` parameters, but sibling scalar extractors still only handled literal values.

- **Symptoms examined:** Parameterized NACL protocol and egress values disappeared during conversion and defaulted to zero values.
- **Code inspected:** `lib/template.go`, `lib/template_helpers_test.go`, and `lib/template_test.go`.
- **Hypotheses tested:** Confirmed with a regression case in `TestNaclResourceToNaclEntry` that `Protocol`, `Egress`, and `RuleAction` parameter references were not resolved.

## Discovered Root Cause

The NACL helper functions for direct scalar properties only type-switched on literal Go values (`string`, `float64`, `bool`). CloudFormation parameter references are decoded as `map[string]any{"Ref": ...}`, so those helpers skipped the value entirely and returned their defaults.

**Defect type:** Logic error

**Why it occurred:** T-834 fixed `extractRuleNumber`, but the same `Ref`-resolution pattern was not applied to sibling helpers that convert NACL properties.

**Contributing factors:** Default return values (`""`, `false`, `allow`) are valid-looking outputs, so the bug was easy to miss without parameterized regression coverage.

## Resolution for the Issue

**Changes made:**
- `lib/template.go:399-407` - Passed stack parameters into all direct NACL scalar extractors during `NetworkAclEntry` conversion.
- `lib/template.go:460-546` - Taught `extractEgressFlag`, `extractProtocol`, and `extractRuleAction` to resolve `{"Ref": "ParamName"}` values via `resolveParameterValue`.
- `lib/template_helpers_test.go:84-388` - Added helper-level regression cases for parameterized egress, protocol, and rule action values, including resolved-value precedence and fallback paths.
- `lib/template_test.go:72-162` - Added an end-to-end regression case proving parameterized NACL entries are converted correctly for drift comparison.

**Approach rationale:** Mirror the existing `extractRuleNumber` and `extractCidrBlock` behaviour so all direct NACL scalar properties use the same parameter-resolution strategy. This keeps the fix local to the drift-conversion helpers and avoids changing unrelated CloudFormation parsing paths.

**Alternatives considered:**
- Resolve parameter references earlier during full template unmarshalling - not chosen because the bug is isolated to NACL drift conversion and the helper-level fix is safer and smaller.

## Regression Test

**Test file:** `lib/template_helpers_test.go`, `lib/template_test.go`
**Test name:** `TestExtractEgressFlag`, `TestExtractProtocol`, `TestExtractRuleAction`, `TestNaclResourceToNaclEntry`

**What it verifies:** Parameterized NACL scalar properties resolve `Ref` values correctly at both helper level and during full `NetworkAclEntry` construction for drift checks.

**Run command:** `go test ./lib -run 'TestNaclResourceToNaclEntry|TestExtractEgressFlag|TestExtractProtocol|TestExtractRuleAction' -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/template.go` | NACL property extraction logic for drift conversion |
| `lib/template_helpers_test.go` | Helper-level regression coverage for parameterized NACL properties |
| `lib/template_test.go` | End-to-end regression for parameterized NACL entry conversion |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Added a failing regression test before implementation to confirm the bug in the NACL conversion path.
- Verified the helper-level and end-to-end regression tests pass after the fix.

## Prevention

**Recommendations to avoid similar bugs:**
- Reuse shared `Ref`-resolution helpers for CloudFormation scalar properties instead of per-field literal-only parsing.
- Add parameter-reference test cases whenever a helper extracts CloudFormation resource properties.
- Review sibling extractors when fixing one member of a helper family.

## Related

- Transit ticket `T-888`
- Follow-up to `T-834`
- Source: PR #181 review by `claude[bot]`
