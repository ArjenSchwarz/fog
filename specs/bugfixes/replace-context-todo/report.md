# Bugfix Report: Replace context.TODO in AWS API calls

**Date:** 2026-03-29
**Status:** Fixed

## Description of the Issue

All AWS SDK API calls throughout the fog CLI use `context.TODO()` instead of a real `context.Context`. This means:
- CLI operations can hang indefinitely if an AWS API call never returns
- There is no way to cancel in-flight operations
- No timeout protection exists for any AWS interaction

The only exception is `lib/tgw_routetables.go` which correctly accepts and propagates a `context.Context` with a 30-second timeout.

**Reproduction steps:**
1. Run any fog command (e.g., `fog deploy`, `fog drift`) against an AWS account
2. If an AWS API call hangs (network issue, service outage), the CLI hangs indefinitely
3. The only way to stop is to kill the process

**Impact:** High — every fog command is affected. Any AWS service disruption causes the CLI to hang with no recovery mechanism.

## Investigation Summary

Systematic audit of all `context.TODO()` usage across the codebase.

- **Symptoms examined:** 37 `context.TODO()` call sites across 11 files in lib/, config/, and cmd/ packages
- **Code inspected:** All lib/*.go files, config/awsconfig.go, cmd/drift.go, and all cmd command handlers
- **Hypotheses tested:** Confirmed that no function in lib/ or config/ (except `GetTransitGatewayRouteTableRoutes`) accepts `context.Context` as a parameter

## Discovered Root Cause

Every lib and config function that makes AWS SDK calls creates its own `context.TODO()` locally, preventing callers from providing cancellation or timeout controls.

**Defect type:** Missing context propagation (design gap)

**Why it occurred:** The codebase was initially written without context propagation. When `lib/tgw_routetables.go` was added with the correct pattern, the rest of the codebase was not updated to match.

**Contributing factors:** Decision 14 in `specs/transit-gateway-drift/decision_log.md` mandated this change but it was only applied to the transit gateway code.

## Resolution for the Issue

**Changes made:**
- All lib functions that make AWS API calls now accept `ctx context.Context` as their first parameter
- `config.DefaultAwsConfig()` accepts `ctx context.Context` as first parameter
- All cmd command handlers pass `context.Background()` to lib/config functions
- Direct `context.TODO()` calls in cmd/drift.go replaced with propagated context
- All existing tests updated to pass `context.Background()`
- New regression tests verify context cancellation propagates correctly

**Approach rationale:** Thread context through the entire call chain following Go best practices and the established pattern in `lib/tgw_routetables.go`. This enables callers to control timeouts and cancellation for all AWS operations.

**Alternatives considered:**
- Adding per-call 30s timeouts inside each lib function — Not chosen because it's inflexible; callers should control timeout policy
- Only fixing high-priority functions — Not chosen because partial fixes leave the same fundamental problem

## Regression Test

**Test file:** `lib/context_propagation_test.go`
**Test name:** `TestContextPropagation_*`

**What it verifies:** That a cancelled context passed to lib functions results in the context error being propagated back to the caller, proving the context is actually used in AWS API calls.

**Run command:** `go test ./lib -run TestContextPropagation -v`

## Affected Files

| File | Change |
|------|--------|
| `config/awsconfig.go` | Add ctx parameter to DefaultAwsConfig, setCallerInfo, setAlias |
| `lib/stacks.go` | Add ctx parameter to GetStack, GetCfnStacks, CreateChangeSet, GetChangeset, GetEvents, fetchAllStackEvents, DeleteStack and all methods calling them |
| `lib/drift.go` | Add ctx parameter to StartDriftDetection, WaitForDriftDetectionToFinish, GetDefaultStackDrift, GetResource, ListAllResources |
| `lib/changesets.go` | Add ctx parameter to DeleteChangeset, DeployChangeset, GetStack |
| `lib/ec2.go` | Add ctx parameter to GetNacl, GetRouteTable, GetManagedPrefixLists |
| `lib/outputs.go` | Add ctx parameter to GetExports, FillImports |
| `lib/resources.go` | Add ctx parameter to GetResources |
| `lib/identitycenter.go` | Add ctx parameter to all functions |
| `lib/files.go` | Add ctx parameter to UploadTemplate |
| `lib/template.go` | Add ctx parameter to GetTemplateBody |
| `cmd/drift.go` | Pass context to all lib/config calls |
| `cmd/deploy.go` | Pass context to lib calls |
| `cmd/deploy_helpers.go` | Update loadAWSConfig type signature |
| `cmd/exports.go` | Pass context to lib/config calls |
| `cmd/resources.go` | Pass context to lib/config calls |
| `cmd/dependencies.go` | Pass context to lib/config calls |
| `cmd/report.go` | Pass context to lib/config calls |
| `cmd/describe_changeset.go` | Pass context to lib/config calls |
| `cmd/history.go` | Pass context to lib/config calls |
| `lib/context_propagation_test.go` | New regression tests for context propagation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Verified no remaining `context.TODO()` in non-test .go files

## Prevention

**Recommendations to avoid similar bugs:**
- Lint rule to flag `context.TODO()` in non-test code
- All new functions making AWS API calls must accept `context.Context` as first parameter
- Follow the pattern established in `lib/tgw_routetables.go`

## Related

- Transit ticket: T-559
- Decision 14 in `specs/transit-gateway-drift/decision_log.md`
- Pattern reference: `lib/tgw_routetables.go`
