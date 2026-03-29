# Bugfix Report: capture-failed-changeset-delete

**Date:** 2025-03-29
**Status:** Fixed

## Description of the Issue

When a CloudFormation changeset creation fails in interactive mode, the user is prompted to confirm deletion of the failed changeset. However, the return value of `askForConfirmation(...)` was not captured, so the confirmation variable always remained `false`. This meant `deleteChangeset(...)` was never called in interactive mode, leaving orphaned failed changesets behind.

In non-interactive mode the code correctly set `deleteChangesetConfirmation = true`, so the bug only affected interactive deployments.

**Reproduction steps:**
1. Run `fog deploy` interactively (without `--non-interactive`) with a template that produces a failed changeset
2. When prompted to delete the failed changeset, confirm with "y"
3. Observe that the changeset is NOT deleted despite user confirmation

**Impact:** Medium — every interactive deployment that encountered a failed changeset left it behind, potentially blocking subsequent deployments on the same stack. Stacks in `REVIEW_IN_PROGRESS` status would also not be cleaned up.

## Investigation Summary

- **Symptoms examined:** `deleteChangeset` never called during interactive failed-changeset flow
- **Code inspected:** `cmd/deploy.go` (`createChangeset` function, lines 358–371)
- **Hypotheses tested:** Confirmed that the return value of `askForConfirmation()` was discarded on line 366

## Discovered Root Cause

On line 366 of `cmd/deploy.go`, the interactive branch called `askForConfirmation(...)` as a statement without assigning its return value:

```go
askForConfirmation(string(texts.DeployChangesetMessageDeleteConfirm))
```

The boolean return was discarded, so `deleteChangesetConfirmation` stayed at its zero value (`false`).

**Defect type:** Missing assignment — discarded return value

**Why it occurred:** Likely an oversight during initial implementation. The non-interactive branch correctly assigned the variable, but the interactive branch omitted the assignment.

**Contributing factors:** The `askForConfirmation` function's side effects (printing the prompt) may have masked the missing assignment during manual testing. Additionally, the function used `askForConfirmation` directly rather than the testable `askForConfirmationFunc` variable, making the code path harder to cover with unit tests.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy.go` — Extracted the failed-changeset cleanup logic into a new `handleFailedChangeset` helper function. The interactive branch now correctly assigns the return value: `deleteChangesetConfirmation = askForConfirmationFunc(...)`. Uses `askForConfirmationFunc` (the mockable variable) and `deleteChangesetFunc` for consistency with the rest of the codebase.

**Approach rationale:** Extracting into a helper makes the cleanup logic independently testable (the original code was embedded in `createChangeset` which calls `os.Exit`). Using the `*Func` variables follows established patterns in the codebase.

**Alternatives considered:**
- Inline one-line fix without extraction — would fix the bug but leave the code untestable due to `os.Exit(1)`
- Adding `osExitFunc` to mock `os.Exit` — more invasive change for less benefit

## Regression Test

**Test file:** `cmd/deploy_helpers_test.go`
**Test name:** `TestHandleFailedChangeset_InteractiveConfirmation`

**What it verifies:** Three scenarios:
1. Interactive mode with user confirming → `deleteChangeset` is called
2. Interactive mode with user declining → `deleteChangeset` is NOT called
3. Non-interactive mode → `deleteChangeset` is called automatically

**Run command:** `go test ./cmd/ -run TestHandleFailedChangeset -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy.go` | Extracted `handleFailedChangeset` helper; fixed missing return value assignment; uses `askForConfirmationFunc` and `deleteChangesetFunc` |
| `cmd/deploy_helpers_test.go` | Added `TestHandleFailedChangeset_InteractiveConfirmation` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters/validators pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- Enable the `errcheck` or `megacheck` linter rule that flags discarded return values from non-void functions
- Prefer using the `*Func` variable pattern (e.g. `askForConfirmationFunc`) consistently to enable test coverage of all code paths
- Extract logic away from `os.Exit` calls to keep functions testable

## Related

- Transit ticket: T-602
