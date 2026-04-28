# Bugfix Report: fix-nil-unsafe-test-assertions-blocking-lint

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

`lib/testutil/assertions.go` contains assertion helpers that call `t.Fatal(...)` for nil `stack` and `changeset` inputs but then continue into code that dereferences those values. The same file also dereferences optional AWS SDK `*string` fields in mismatch paths for stack parameters, outputs, and tags.

**Reproduction steps:**
1. Run `make lint` with repository-local cache directories.
2. Inspect `lib/testutil/assertions.go` and exercise the assertion helpers with malformed AWS responses containing nil optional fields.
3. Observe staticcheck SA5011 warnings and nil-pointer panics from the assertion helpers instead of clean test failures.

**Impact:** Linting is blocked and malformed AWS test fixtures can panic inside assertion helpers, hiding the actual test failure.

## Investigation Summary

- **Symptoms examined:** SA5011 nil-dereference warnings and panic-prone mismatch code paths in assertion helpers.
- **Code inspected:** `lib/testutil/assertions.go`, `Makefile`, `.golangci.yml`.
- **Hypotheses tested:** Nil optional AWS SDK string fields reach mismatch branches; `t.Fatal(...)` alone is not enough to satisfy static analysis without an explicit return.

## Discovered Root Cause

The assertion helpers assumed optional AWS SDK `*string` fields were always non-nil when rendering mismatch messages, and they relied on `t.Fatal(...)` without explicit returns after nil guards.

**Defect type:** Missing nil validation.

**Why it occurred:** The helpers were written for happy-path test data, but AWS SDK v2 models use pointer fields for optional values and malformed fixtures can leave them nil.

**Contributing factors:** The mismatch messages dereferenced pointer fields directly, and staticcheck treats code after `t.Fatal(...)` as reachable unless the function returns explicitly.

## Resolution for the Issue

**Changes made:**
- `lib/testutil/assertions.go:55-199` — added explicit returns after fatal nil guards in stack and changeset helpers so static analysis no longer sees reachable dereferences.
- `lib/testutil/assertions.go:104-107,129-132,154-157,321-327` — routed mismatch rendering through a nil-safe helper so malformed AWS SDK pointers show `<nil>` instead of panicking.
- `lib/testutil/assertions_test.go:14-110` — added subprocess-based regression tests that prove nil parameter, output, and tag values fail cleanly without panics.

**Approach rationale:** Keep the fix local to the assertion helpers. Explicit early returns address the lint warning directly, while a shared nil-safe formatter keeps mismatch messages readable without changing the helper APIs.

**Alternatives considered:**
- Broader assertion helper refactors — not chosen because the bug is isolated to a few nil-unsafe branches and does not require API changes.

## Regression Test

**Test file:** `lib/testutil/assertions_test.go`
**Test names:**
- `TestAssertStackParameter_NilValue`
- `TestAssertStackOutput_NilValue`
- `TestAssertStackTag_NilValue`

**What it verifies:** Each helper reports a clean mismatch when the matching AWS SDK value pointer is nil, instead of panicking.

**Run command:** `go test ./lib/testutil -run 'TestAssertStack(Parameter|Output|Tag)_NilValue' -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/testutil/assertions.go` | Added explicit returns after fatal nil checks and nil-safe mismatch formatting |
| `lib/testutil/assertions_test.go` | Adds regression coverage for nil optional AWS SDK fields |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Treat AWS SDK pointer fields as nilable in both success and failure paths.
- Return immediately after `t.Fatal(...)` in helpers when static analysis otherwise sees a reachable dereference path.

## Related

- Transit ticket: T-898
