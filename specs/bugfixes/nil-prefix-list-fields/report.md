# Bugfix Report: nil-prefix-list-fields

**Date:** 2025-07-14
**Status:** Fixed
**Transit:** T-795

## Description of the Issue

`checkRouteTableRoutes` in `cmd/drift.go` panics when `DescribeManagedPrefixLists` returns a managed prefix list entry with a nil `OwnerId` or nil `PrefixListId`. The code dereferenced both pointer fields without nil guards, so any partial or malformed EC2 SDK response would crash drift detection.

**Reproduction steps:**
1. Run `fog drift` against a stack containing route tables.
2. Have the EC2 API return a `ManagedPrefixList` entry where `OwnerId` or `PrefixListId` is nil.
3. Observe a nil pointer dereference panic.

**Impact:** Any partial EC2 response (or a test double with nil fields) causes drift detection to crash before producing any output.

## Investigation Summary

- **Symptoms examined:** Potential nil pointer dereference at `cmd/drift.go:398-399`.
- **Code inspected:** `cmd/drift.go` (`checkRouteTableRoutes`), `lib/ec2.go` (`GetManagedPrefixLists`).
- **Hypotheses tested:** Confirmed that EC2 SDK `ManagedPrefixList` fields (`OwnerId`, `PrefixListId`) are pointer-backed (`*string`) and can be nil.

## Discovered Root Cause

The inline loop at lines 397-400 dereferenced `*prefixlist.OwnerId` and `*prefixlist.PrefixListId` without checking for nil. AWS SDK v2 represents all EC2 response fields as pointers, meaning any of them can be nil in partial responses.

**Defect type:** Missing nil validation

**Why it occurred:** The original code assumed all fields in a `ManagedPrefixList` would always be populated by the SDK.

**Contributing factors:** AWS SDK v2's pointer-heavy response types make nil dereference bugs easy to introduce when the happy-path response always has values set.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go` — Extracted inline filtering into `awsManagedPrefixListIDs` helper that guards against nil `OwnerId`, nil `PrefixListId`, and empty `PrefixListId` before appending.
- `cmd/drift_prefix_list_test.go` — Added regression tests covering nil `OwnerId`, nil `PrefixListId`, both nil, empty `PrefixListId`, non-AWS owner, valid AWS entry, and mixed entries.

**Approach rationale:** Extracting to a named helper makes the nil-guard logic independently testable and keeps `checkRouteTableRoutes` focused on route comparison.

**Alternatives considered:**
- Inline nil checks without extraction — works but harder to test in isolation.

## Regression Test

**Test file:** `cmd/drift_prefix_list_test.go`
**Test name:** `TestAwsManagedPrefixListIDs_NilFields`

**What it verifies:** That `awsManagedPrefixListIDs` does not panic on nil/empty fields, correctly filters AWS-owned entries, and skips entries with missing data.

**Run command:** `go test ./cmd/... -run TestAwsManagedPrefixListIDs -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Replaced inline filtering with `awsManagedPrefixListIDs` helper that includes nil guards |
| `cmd/drift_prefix_list_test.go` | New regression test covering nil/empty field scenarios |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always guard against nil when dereferencing AWS SDK v2 pointer fields.
- Prefer extracting SDK response processing into testable helper functions.
- Consider a project-wide lint rule or convention for SDK pointer access patterns.

## Related

- Transit ticket: T-795
