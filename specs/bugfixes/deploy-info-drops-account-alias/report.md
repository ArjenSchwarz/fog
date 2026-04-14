# Bugfix Report: deploy-info-drops-account-alias

**Date:** 2025-04-14
**Status:** Fixed

## Description of the Issue

When deploying to an existing CloudFormation stack, the deployment info message printed the bare AWS account ID instead of the formatted account display that includes the configured alias. New-stack deployments correctly showed `alias (accountID)`, creating an inconsistency.

**Reproduction steps:**
1. Configure an account alias in fog.yaml (e.g., `accountalias: staging`)
2. Deploy to an existing stack
3. Observe: the info line shows only the account ID, not `staging (987654321098)`

**Impact:** Low severity — cosmetic inconsistency in CLI output, but confusing when managing multiple AWS accounts.

## Investigation Summary

The `showDeploymentInfo` function in `cmd/deploy.go` computes the formatted account display string via `formatAccountDisplay(awsConfig.AccountID, awsConfig.AccountAlias)` and stores it in the `account` variable. However, only the new-stack branch uses `account`; the existing-stack branch directly uses `awsConfig.AccountID`.

- **Symptoms examined:** Different output for new vs existing stack deployments
- **Code inspected:** `cmd/deploy.go:showDeploymentInfo`, `cmd/deploy_helpers.go:formatAccountDisplay`
- **Hypotheses tested:** Single hypothesis — the `else` branch bypasses the `account` variable

## Discovered Root Cause

**Defect type:** Logic error — inconsistent variable usage

**Why it occurred:** The `account` variable was computed but not used in the `else` branch. The new-stack and existing-stack branches were likely written at different times, and the `else` branch was never updated to use the formatted display.

**Contributing factors:** The existing golden test (`output_golden_test.go`) mirrored the same bug, so it never caught the inconsistency.

## Resolution for the Issue

**Changes made:**
- `cmd/deploy.go:139` — Changed `awsConfig.AccountID` to `account` in the existing-stack branch
- `cmd/output_golden_test.go:75` — Fixed the same bug in the golden test
- `cmd/testdata/golden/cmd/update_stack_deployment.golden` — Updated golden file to expect alias output

**Approach rationale:** The `account` variable was already computed on line 134 for both branches — it just wasn't being used in the `else` path. Using it makes both branches consistent.

**Alternatives considered:**
- Inlining `formatAccountDisplay` in both branches — rejected as unnecessarily verbose; the variable already exists.

## Regression Test

**Test file:** `cmd/deploy_account_alias_test.go`
**Test names:** `TestShowDeploymentInfo_ExistingStackUsesAccountAlias`, `TestShowDeploymentInfo_ExistingStackNoAlias`

**What it verifies:** The first line of deployment info for existing stacks includes the account alias when configured, and shows just the account ID when no alias is set.

**Run command:** `go test ./cmd -run TestShowDeploymentInfo_ExistingStack -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy.go` | Use `account` variable in existing-stack branch |
| `cmd/output_golden_test.go` | Fix same bug in golden test |
| `cmd/testdata/golden/cmd/update_stack_deployment.golden` | Update expected output |
| `cmd/deploy_account_alias_test.go` | New regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the first line of existing-stack output now shows `staging (987654321098)` instead of `987654321098`

## Prevention

**Recommendations to avoid similar bugs:**
- When a local variable is computed for use in multiple branches, audit all branches to ensure consistent usage
- Golden tests should validate expected behavior independently, not mirror the implementation

## Related

- Transit ticket: T-676
