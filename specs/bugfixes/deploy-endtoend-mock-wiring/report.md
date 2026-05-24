# Bugfix Report: deploy-endtoend-mock-wiring

**Date:** 2026-05-24
**Status:** Fixed
**Ticket:** T-1340

## Description of the Issue

`TestDeploymentWorkflow_EndToEnd` (`cmd/deploy_integration_test.go`) failed under `INTEGRATION=1 go test ./cmd/` once it was actually compiled and run (it had been dead code behind a `//go:build integration` tag the test commands never activated — see T-1325). It made a **real** `CreateChangeSet` AWS call ("failed to resolve service endpoint ... Missing Region"), and its inline tags were treated as a file path ("no file found for 'tags' matching 'Environment=test'").

**Reproduction steps:**
1. `INTEGRATION=1 go test ./cmd/ -run TestDeploymentWorkflow_EndToEnd`
2. Observe: a real `CreateChangeSet` call fails with "Missing Region", and/or tag loading fails on the inline tag string.

**Impact:** The end-to-end deploy workflow test could not pass. Low blast radius (test-only), but it blocked the integration suite from being green.

## Investigation Summary

- **Symptoms examined:** real AWS endpoint error from `CreateChangeSet`; tags-as-file error; no `--- FAIL` line (the deploy flow's uncaught `os.Exit`/error path terminated the run).
- **Code inspected:** `cmd/deploy_helpers.go` (`prepareDeployment`, `createAndShowChangeset`, `confirmAndDeployChangeset`, the `loadAWSConfig`/`getCfnClient`/`createChangesetFunc`/`deployChangesetFunc`/`deleteChangesetFunc`/`askForConfirmationFunc` seams), `cmd/deploy.go` (`createChangeset`, `setDeployTags`/`loadTagsFromFiles`, `deployTemplate`), `config/awsconfig.go` (`AWSConfig.CloudformationClient`).
- **Hypotheses tested:** confirmed `createChangeset` calls `awsConfig.CloudformationClient()` — a concrete `*cloudformation.Client` with no mock seam — so overriding `getCfnClient` (which only feeds `prepareDeployment`'s stack lookup) can never mock changeset creation. Confirmed `deployFlags.Tags` is always parsed as comma-separated file paths. Confirmed the original test's per-case logic (dry-run, interactive) did not match the actual flow.

## Discovered Root Cause

The test was written against an incorrect mental model and never ran (the build tag excluded it from every test command), so the drift was never caught:
1. It expected `mockCFN` (wired via `getCfnClient`) to cover changeset creation, but `createChangeset` builds a real client from `config.AWSConfig`.
2. It set inline tags into `deployFlags.Tags`, which the code treats as file paths.
3. Several table cases (dry-run, interactive) asserted outcomes the test's own execution path never produced.

**Defect type:** Stale/incorrect test (wrong injection model + drifted assertions).

## Resolution for the Issue

**Changes made (test-only):**
- `cmd/deploy_integration_test.go` - rewrote `TestDeploymentWorkflow_EndToEnd` to drive the real orchestration (`prepareDeployment` -> `createAndShowChangeset` -> `confirmAndDeployChangeset`) through the package's actual seams. It overrides `createChangesetFunc`, `deployChangesetFunc`, `deleteChangesetFunc`, `askForConfirmationFunc`, `loadAWSConfig`, and `getCfnClient`, removed the misleading `mockCFN.*Fn` changeset stubs and the inline tags/params, and reduced the cases to the three cleanly-testable scenarios. Removed the now-unused `strings` import and the `t.Skip` placeholder.

**Approach rationale:** `createChangeset` and `deleteChangeset` use a concrete CloudFormation client with no injection seam, and `deployTemplate`'s dry-run path calls `deleteChangeset`/`printDeploymentResults` directly (also unmockable), so a faithful test must use the orchestration func-var seams rather than the AWS client. The three retained cases (non-interactive deploy of a new and an existing stack, and interactive decline-deletes) exercise the real `createAndShowChangeset` + `confirmAndDeployChangeset` logic (including T-1284's preserve-on-decline behaviour).

**Alternatives considered:**
- Add injection seams to production `createChangeset`/`deleteChangeset` so the mock client is used - rejected: production change to support one test, higher risk.
- Call `deployTemplate()` for a true end-to-end run - rejected: it makes non-injectable direct calls (`deleteChangeset`, `printDeploymentResults`) that would hit a real client.

## Regression Test

**Test file:** `cmd/deploy_integration_test.go`
**Test name:** `TestDeploymentWorkflow_EndToEnd` (subtests: new-stack deploy, existing-stack deploy, interactive decline)

**What it verifies:** the deploy orchestration creates a changeset, deploys on confirmation (non-interactive), and deletes the changeset on an interactive decline.

**Run command:** `INTEGRATION=1 go test ./cmd/ -run TestDeploymentWorkflow_EndToEnd`

## Affected Files

| File | Change |
|------|--------|
| `cmd/deploy_integration_test.go` | Rewrote `TestDeploymentWorkflow_EndToEnd` to use the orchestration seams; removed `strings` import and the `t.Skip` placeholder |

## Verification

**Automated:**
- [x] Regression test passes (`INTEGRATION=1 go test ./cmd/ -run TestDeploymentWorkflow_EndToEnd`)
- [x] Full test suite passes (`go test ./...` and `INTEGRATION=1 go test ./...`)
- [x] Linter passes (`golangci-lint run ./...`, 0 issues)

## Prevention

- With the build tag gone (T-1325), this test now compiles in the default build and runs under `INTEGRATION=1`, so future drift is caught immediately.
- Prefer testing orchestration via the established func-var seams; the AWS client path is not mockable through `config.AWSConfig`.

## Related

- T-1325 - removed the `//go:build integration` tag that had hidden this test
- T-1284 - preserve-on-decline behaviour exercised by the decline case
