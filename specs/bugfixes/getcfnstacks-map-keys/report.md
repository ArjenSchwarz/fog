# Bugfix Report: GetCfnStacks Map Keys Used as Stack IDs

**Date:** 2026-03-30
**Status:** Fixed
**Transit:** T-545

## Description of the Issue

`GetCfnStacks` returned a `map[string]CfnStack` keyed by `StackId` (an AWS ARN like `arn:aws:cloudformation:us-west-2:123456789012:stack/my-stack/guid`), but all callers treated the keys as human-readable stack names. This caused stack name filtering in the dependencies command to never match, and report sorting to use ARNs instead of names.

**Reproduction steps:**
1. Run `fog stack dependencies --stackname "my-stack"`
2. The filter compares "my-stack" against ARN keys — no match
3. All stacks are silently excluded from the output

**Impact:** Stack name filtering was completely broken in the `dependencies` command. The `report` command sorted and printed stacks by ARN rather than name.

## Investigation Summary

- **Symptoms examined:** `dependencies` command filtering produced empty output for valid stack names
- **Code inspected:** `lib/stacks.go:GetCfnStacks`, `cmd/dependencies.go:showDependencies`, `cmd/dependencies.go:getFilteredStacks`, `cmd/report.go:generateReport`
- **Hypotheses tested:** Confirmed that `GetCfnStacks` on line 260 used `*stack.StackId` (ARN) as the map key, while `getFilteredStacks` and `showDependencies` expected keys to be stack names

## Discovered Root Cause

On `lib/stacks.go:260`, the map was populated with `result[*stack.StackId] = stackobject`, using the AWS Stack ID (an ARN) as the key. However, callers iterate the map treating keys as stack names for filtering, display, and sorting.

**Defect type:** Logic error — wrong field used as map key

**Why it occurred:** `StackId` and `StackName` are both string fields on the AWS Stack type, and the original code likely confused the two. The function also stores both values in the `CfnStack` struct (`Name` and `Id`), but used the wrong one for the key.

**Contributing factors:** No unit tests existed for `GetCfnStacks` to catch this, because the function previously accepted a concrete `*cloudformation.Client` that was difficult to mock.

## Resolution for the Issue

**Changes made:**
- `lib/stacks.go:260` — Changed map key from `*stack.StackId` to `*stack.StackName`
- `lib/stacks.go:217` — Changed function signature from `*cloudformation.Client` to `CFNExportsAPI` interface to enable testability (follows the pattern already used by `GetExports`)

**Approach rationale:** The fix is a one-line key change. The interface refactoring follows the existing pattern in `lib/outputs.go:GetExports` and enables proper unit testing with `MockCFNClient`.

**Alternatives considered:**
- Updating all callers to use `CfnStack.Name` instead of the map key — rejected because it would require more changes and still leave a confusing API where keys don't match stack names

## Regression Test

**Test file:** `lib/getcfnstacks_test.go`
**Test names:**
- `TestGetCfnStacks_MapKeysAreStackNames` — verifies keys are names, not ARNs
- `TestGetCfnStacks_StackNameFieldMatchesKey` — verifies key equals `CfnStack.Name`
- `TestGetCfnStacks_GlobFilterUsesStackName` — verifies glob filtering works
- `TestGetCfnStacks_SpecificStackFilter` — verifies exact name filtering works

**What it verifies:** Map keys returned by `GetCfnStacks` are stack names (not stack ID ARNs), and stack name filtering works correctly.

**Run command:** `go test ./lib -run TestGetCfnStacks -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/stacks.go` | Changed map key from StackId to StackName; changed function signature to accept `CFNExportsAPI` interface |
| `lib/getcfnstacks_test.go` | New regression tests for `GetCfnStacks` |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When returning maps keyed by a "name", always test that the key matches expectations
- Prefer interface parameters over concrete AWS client types to enable testability
- Add unit tests for any function that builds maps from AWS API responses

## Related

- Transit ticket: T-545
