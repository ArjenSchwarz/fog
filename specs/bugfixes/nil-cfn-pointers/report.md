# Bugfix Report: nil-cfn-pointers

**Date:** 2026-04-14
**Status:** Fixed

## Description of the Issue

Fog panics with a nil pointer dereference when CloudFormation returns parameter entries that have nil `ParameterKey` or `ParameterValue` pointers. This can happen during drift detection and template route processing when AWS returns incomplete parameter data.

**Reproduction steps:**
1. Have a CloudFormation stack with parameters where AWS returns a parameter entry with a nil `ParameterKey` or `ParameterValue` pointer
2. Run fog drift detection or template route comparison
3. Observe a panic: `runtime error: invalid memory address or nil pointer dereference`

**Impact:** Crash (panic) during drift detection or route table comparison. Any stack with such parameter entries becomes unusable with fog.

## Investigation Summary

Systematic review of all parameter iteration loops in `lib/template.go` and `lib/tgw_routetables.go`.

- **Symptoms examined:** Nil pointer dereference panic in `resolveParameterValue`
- **Code inspected:** `resolveParameterValue`, `stringPointer` (both in `lib/template.go`), and `extractStringProperty` (in `lib/tgw_routetables.go`)
- **Hypotheses tested:** All three functions iterate over `[]cfntypes.Parameter` and dereference pointer fields without nil guards

## Discovered Root Cause

Three functions dereference `*parameter.ParameterKey` and `*parameter.ParameterValue` without checking for nil first:

1. `resolveParameterValue` (template.go:465) — dereferences `*parameter.ParameterKey` directly
2. `stringPointer` (template.go:600) — dereferences `*parameter.ParameterKey` and `*parameter.ParameterValue`
3. `extractStringProperty` (tgw_routetables.go) — dereferences `*parameter.ParameterKey` and `*parameter.ParameterValue`

**Defect type:** Missing nil validation

**Why it occurred:** The AWS SDK v2 CloudFormation types use `*string` for ParameterKey and ParameterValue, meaning they can be nil. The code assumed these would always be populated.

**Contributing factors:** The AWS SDK uses pointer types for optional fields. While these fields are typically populated, there's no contract guaranteeing non-nil values.

## Resolution for the Issue

**Changes made:**
- `lib/template.go:resolveParameterValue` — skip parameter entries with nil `ParameterKey`; return empty if matched key has nil `ResolvedValue` and nil `ParameterValue`
- `lib/template.go:stringPointer` — skip parameter entries with nil `ParameterKey`; guard `ParameterValue` dereference with nil check
- `lib/tgw_routetables.go:extractStringProperty` — skip parameter entries with nil `ParameterKey`; guard `ParameterValue` dereference with nil check

**Approach rationale:** Defensive nil checks before every pointer dereference. Entries with nil keys are skipped (they cannot match any reference name). Entries with nil values are treated as unresolvable (return empty string).

**Alternatives considered:**
- Filtering parameters at a higher level — not chosen because it would require changes to many callers and the defensive checks are simple and localized.

## Regression Test

**Test file:** `lib/template_test.go`, `lib/tgw_routetables_test.go`
**Test names:**
- `TestResolveParameterValue_NilPointerKey`
- `TestResolveParameterValue_NilPointerValue`
- `TestStringPointer_NilParameterKeyAndValue`
- `TestRouteResourceToRoute_NilParameterFields`
- `TestExtractStringProperty_NilParameterKeyAndValue`
- `TestTGWRouteResourceToTGWRoute_NilParameterFields`

**What it verifies:** That nil `ParameterKey` and `ParameterValue` fields do not cause panics, and that valid parameters are still resolved correctly in the presence of nil entries.

**Run command:** `go test ./lib/ -run 'TestResolveParameterValue_NilPointer|TestStringPointer_NilParameter|TestRouteResourceToRoute_NilParameter|TestExtractStringProperty_NilParameter|TestTGWRouteResourceToTGWRoute_NilParameter' -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/template.go` | Add nil guards in `resolveParameterValue` and `stringPointer` |
| `lib/tgw_routetables.go` | Add nil guards in `extractStringProperty` |
| `lib/template_test.go` | Add regression tests for nil parameter fields |
| `lib/tgw_routetables_test.go` | Add regression tests for nil parameter fields |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Always nil-check AWS SDK pointer fields before dereferencing
- Consider adding a linter rule or code review checklist item for AWS SDK pointer dereferences

## Related

- Transit ticket: T-775
