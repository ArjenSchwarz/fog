# Bugfix Report: AddChangeset panics on partial DescribeChangeSet responses

**Date:** 2026-05-22
**Status:** Fixed
**Ticket:** T-1111

## Description of the Issue

`lib.DeployInfo.AddChangeset` processed a `[]cloudformation.DescribeChangeSetOutput`
by directly dereferencing `resp[0]` and several SDK pointer fields. When the
response slice was empty, or when a response was missing a required-looking
pointer field, the function panicked (nil pointer dereference / index out of
range) instead of reporting an invalid changeset response.

**Reproduction steps:**
1. Trigger any flow that builds a changeset view: `fog describe changeset`,
   regular deploy changeset display, or the `--deploy-changeset` flow.
2. Have the underlying `DescribeChangeSet` call return an empty result set, or a
   response missing one of `StackId` / `StackName` / `ChangeSetId` /
   `ChangeSetName` / `CreationTime`.
3. Observe a panic instead of a clean error message.

**Impact:** Medium. A crash (panic) rather than a handled error on the three
changeset-display code paths. No data loss, but a poor failure mode.

## Investigation Summary

- **Symptoms examined:** Panic on unchecked dereference at `lib/stacks.go:473-483`.
- **Code inspected:** `lib/stacks.go` (`AddChangeset`, `WaitUntilChangesetDone`),
  `cmd/deploy.go` (`fetchChangeset`), `cmd/describe_changeset.go`,
  `lib/changesets.go` (`ChangesetInfo`).
- **Hypotheses tested:** Confirmed all three callers reach `AddChangeset`. Two of
  them (`fetchChangeset`, `WaitUntilChangesetDone`) had partial guards but still
  passed unvalidated data into `AddChangeset`; `describe changeset` had no guard.

## Discovered Root Cause

`AddChangeset` was written for the happy path only. It assumed the response slice
was non-empty and that the AWS SDK always populated the identifying pointer
fields. Neither assumption is guaranteed.

**Defect type:** Missing input validation (unchecked nil/slice dereference).

**Why it occurred:** No validation layer existed between the raw SDK response and
the construction of `ChangesetInfo`. The function dereferenced the required
pointer fields directly, and the inner loop dereferenced `change.ResourceChange`
without a nil check.

**Contributing factors:** The function returned only `ChangesetInfo` (no error),
so callers had no channel to surface a validation failure.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go` - `AddChangeset` now returns `(ChangesetInfo, error)`. It
  validates the slice is non-empty and that required pointer fields (`StackId`,
  `StackName`, `ChangeSetId`, `ChangeSetName`, `CreationTime`) are present before
  dereferencing, returning a descriptive error listing the missing field(s). The
  optional `StatusReason` is still skipped when nil. The per-change loop skips
  entries with a nil `ResourceChange`.
- `lib/stacks.go` - `WaitUntilChangesetDone` guards against an empty response
  before indexing `resp[0]` and propagates the new error.
- `cmd/deploy.go` - `fetchChangeset` handles the new error, reporting and exiting.
- `cmd/describe_changeset.go` - the describe path handles the new error.

**Approach rationale:** Returning an error from `AddChangeset` keeps validation in
one place and gives every caller a uniform way to report invalid responses.

**Alternatives considered:**
- Guarding only at each call site - Rejected: duplicates logic and leaves the
  panic latent in the helper.
- Substituting empty strings for missing required fields - Rejected: hides
  malformed responses and produces misleading output.

## Regression Test

**Test file:** `lib/stacks_refactored_test.go`
**Test name:** `TestDeployInfo_AddChangeset_PartialResponses`

**What it verifies:** That `AddChangeset` returns an error (never panics) for an
empty slice, a nil slice, and responses missing each required pointer field; and
that it succeeds for a valid full response and a valid response missing the
optional `StatusReason` field.

**Run command:** `go test ./lib/ -run TestDeployInfo_AddChangeset_PartialResponses`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Validate response before dereferencing; `AddChangeset` returns an error; guard empty response in `WaitUntilChangesetDone` |
| `cmd/deploy.go` | Handle `AddChangeset` error in `fetchChangeset` |
| `cmd/describe_changeset.go` | Handle `AddChangeset` error in describe path |
| `lib/stacks_refactored_test.go` | Add regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...` and `INTEGRATION=1 go test ./...`)
- [x] Linters pass (`golangci-lint run` - 0 issues)

**Manual verification:**
- Confirmed the regression test fails to compile against the old single-return
  signature (red), then passes after the fix (green).

## Prevention

- Treat AWS SDK pointer fields as potentially nil; validate before dereferencing.
- Parse-and-validate helpers should return an error so callers can surface
  failures cleanly.
- Guard slice indexing with a length check at the parse boundary.

## Related

- Transit ticket T-1111
- Similar prior fix: T-1026 (Guard Nil CloudFormation Pointer Fields)
