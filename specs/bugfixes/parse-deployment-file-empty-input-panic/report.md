# Bugfix Report: ParseDeploymentFile Empty Input Panic

**Date:** 2026-03-04
**Status:** Fixed

## Description of the Issue

`ParseDeploymentFile` panicked when called with an empty deployment file string because it indexed `deploymentFile[0]` without checking length first.

**Reproduction steps:**
1. Call `ParseDeploymentFile("")`.
2. Function evaluates `deploymentFile[0]` to detect JSON vs YAML.
3. Runtime panics with `index out of range [0] with length 0`.

**Impact:** Any empty deployment file input crashes execution instead of returning a validation error; whitespace-only input was also accepted incorrectly.

## Investigation Summary

A focused inspection was performed on deployment parsing flow in `lib/stacks.go` and tests in `lib/stacks_test.go`.

- **Symptoms examined:** panic stack trace and parser behavior for empty/whitespace inputs.
- **Code inspected:** `ParseDeploymentFile` implementation and related parsing tests.
- **Hypotheses tested:**
  - Empty string triggers direct out-of-bounds index access.
  - Whitespace-only input bypasses explicit validation and can decode into an unintended zero-value object.

## Discovered Root Cause

`ParseDeploymentFile` assumed the input string always had at least one byte and accessed index `0` before validating content.

**Defect type:** Missing validation / boundary-condition bug.

**Why it occurred:**
- Why panic happened: code read the first character unconditionally.
- Why this was unsafe: no guard existed for empty input.
- Why guard was missing: parser had format detection logic but no upfront validation path.
- Why whitespace issue existed: no normalization/trim check before parsing.

**Contributing factors:** Existing tests covered valid JSON/YAML and invalid JSON, but not empty or whitespace-only deployment file inputs.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:380-388` - Added `strings.TrimSpace` guard that returns a descriptive validation error for empty/whitespace input and uses trimmed content for format detection.
- `lib/stacks_test.go:608-656` - Added regression coverage for empty and whitespace-only inputs and verified error message content.

**Approach rationale:**
Validate input once at function start to prevent panics and make invalid input handling explicit and user-friendly.

**Alternatives considered:**
- Recovering from panic at call sites - rejected because validation belongs in parser and panic recovery hides defects.

## Regression Test

**Test file:** `lib/stacks_test.go`
**Test name:** `TestParseDeploymentFile`

**What it verifies:** Empty and whitespace-only deployment content now returns a descriptive error and does not panic.

**Run command:** `go test ./lib -run TestParseDeploymentFile -count=1`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Added empty/whitespace validation before JSON/YAML detection. |
| `lib/stacks_test.go` | Added regression test cases for empty and whitespace-only input plus error assertions. |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed pre-fix regression test failed with panic (`index out of range`) on empty input.
- Confirmed post-fix behavior returns validation errors for both empty and whitespace-only input.

## Prevention

**Recommendations to avoid similar bugs:**
- Add boundary-input validation before string indexing.
- Include empty and whitespace-only cases in parser test matrices.
- Prefer explicit error returns over panic-prone assumptions in parsing paths.

## Related

- Transit ticket: `T-326`
