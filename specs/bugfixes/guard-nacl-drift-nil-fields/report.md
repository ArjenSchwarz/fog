# Bugfix Report: Guard NACL Drift Rendering Against nil EC2 Entry Fields

**Date:** 2026-04-14
**Status:** Fixed
**Transit:** T-794

## Description of the Issue

`cmd/drift.go` panics at runtime when AWS EC2 returns a `NetworkAclEntry` with
one or more nil pointer fields.  The crash can occur in two places:

1. **`checkNaclEntries`** – dereferences `*entry.Egress` and `*entry.RuleNumber`
   while building the rule key string used to match entries against the
   CloudFormation template.
2. **`naclEntryToString`** – dereferences `*entry.Egress`, `*entry.RuleNumber`,
   `*entry.Protocol`, and the nested `PortRange.From`/`PortRange.To` and
   `IcmpTypeCode.Type`/`IcmpTypeCode.Code` fields while formatting
   human-readable drift details.

**Reproduction steps:**
1. Have a CloudFormation stack with NACL resources.
2. AWS returns a partial `NetworkAclEntry` (e.g. a just-created or
   transitional entry) where one of the pointer fields is nil.
3. Run `fog drift` — the process panics with a nil pointer dereference.

**Impact:** Any user running drift detection on a stack with NACLs could hit
an unrecoverable panic if the EC2 API returns incomplete data.

## Investigation Summary

- **Symptoms examined:** Nil pointer dereference panic in drift rendering code.
- **Code inspected:** `cmd/drift.go` lines 338-343 (`checkNaclEntries`) and
  lines 719-746 (`naclEntryToString`).
- **Hypotheses tested:** The AWS EC2 SDK models all scalar fields on
  `NetworkAclEntry` as pointers (`*bool`, `*int32`, `*string`). Other drift
  code in the same file already uses `aws.ToString` for similar pointer fields.
  The NACL code paths were the only ones that skipped nil checks.

## Discovered Root Cause

**Defect type:** Missing nil-pointer validation

**Why it occurred:** The original code assumed EC2 would always populate every
field on a `NetworkAclEntry`. AWS SDK v2 models these as pointers precisely
because they can be absent. Other parts of `drift.go` already used
`aws.ToString` / `aws.ToInt32` for safe access; the NACL helpers were written
without the same discipline.

**Contributing factors:** No unit tests existed for the NACL rendering
functions, so the gap was not caught before.

## Resolution for the Issue

**Changes made:**
- `cmd/drift.go` – extracted inline key-building logic into a new
  `naclEntryKey(entry)` function that nil-guards both `Egress` and
  `RuleNumber`.
- `cmd/drift.go` – rewrote `naclEntryToString` to nil-guard every pointer
  dereference: `Egress`, `RuleNumber`, `Protocol`, `PortRange.From`,
  `PortRange.To`, `IcmpTypeCode.Type`, `IcmpTypeCode.Code`.
- Used `aws.ToString` / `aws.ToInt32` from the AWS SDK for safe zero-value
  fallbacks where appropriate.

**Approach rationale:** Using the same AWS SDK helpers already in use elsewhere
in the file keeps the style consistent and avoids inventing custom helpers.

**Alternatives considered:**
- Skip entries with nil fields entirely — rejected because partial information
  is still useful for drift reporting.
- Create a centralised safe-dereference helper — unnecessary given the SDK
  already provides `aws.ToString` / `aws.ToInt32`.

## Regression Test

**Test file:** `cmd/drift_nacl_nil_test.go`
**Test names:**
- `TestNaclEntryToString_NilEgress`
- `TestNaclEntryToString_NilRuleNumber`
- `TestNaclEntryToString_NilProtocol`
- `TestNaclEntryToString_NilPortRangeFields`
- `TestNaclEntryToString_NilIcmpTypeCodeFields`
- `TestNaclEntryToString_AllNilFields`
- `TestNaclEntryToString_FullyPopulated`
- `TestCheckNaclEntryKey_NilEgress`
- `TestCheckNaclEntryKey_NilRuleNumber`
- `TestCheckNaclEntryKey_BothNil`

**What it verifies:** Every combination of nil pointer fields in a
`NetworkAclEntry` renders without panicking, and the fully-populated happy
path still produces the expected output.

**Run command:** `go test ./cmd/ -run "TestNacl|TestCheckNaclEntryKey" -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/drift.go` | Added `naclEntryKey` helper; rewrote `naclEntryToString` with nil guards |
| `cmd/drift_nacl_nil_test.go` | New regression tests for nil NACL entry fields |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Always nil-check AWS SDK v2 pointer fields before dereferencing; prefer
  `aws.ToString` / `aws.ToInt32` / `aws.ToBool` helpers.
- Add unit tests for rendering/formatting functions that consume AWS structs.
- Consider a linter rule or code-review checklist item for raw `*ptr`
  dereferences on SDK types.

## Related

- Transit T-794
