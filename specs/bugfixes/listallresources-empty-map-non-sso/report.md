# Bugfix Report: ListAllResources Empty Map for Non-SSO Types

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

`lib.ListAllResources` returns an empty map for all resource types other than `AWS::SSO::PermissionSet` and `AWS::SSO::Assignment`. The Cloud Control `ListResources` call that should handle generic resource types was commented out, causing the function to silently return an empty result.

**Reproduction steps:**
1. Configure `drift.detect-unmanaged-resources` in fog config with a non-SSO resource type (e.g., `AWS::S3::Bucket`)
2. Run `fog stack drift` against a stack
3. Observe that no unmanaged resources are reported for the configured type, even when unmanaged resources exist

**Impact:** Unmanaged resource detection is silently non-functional for all non-SSO resource types. Users relying on `drift.detect-unmanaged-resources` for types like S3 buckets, IAM roles, etc. receive false negatives.

## Investigation Summary

- **Symptoms examined:** `ListAllResources` always returns `map[string]string{}` for non-SSO types
- **Code inspected:** `lib/drift.go` (ListAllResources), `cmd/drift.go` (caller), `lib/interfaces.go` (API interfaces)
- **Hypotheses tested:** The commented-out code block at lines 113-121 of `lib/drift.go` contains the Cloud Control ListResources implementation that was never enabled

## Discovered Root Cause

The Cloud Control `ListResources` API call in `ListAllResources` (lib/drift.go:113-121) is entirely commented out. After the SSO-specific branches, the function unconditionally returns an empty map.

**Defect type:** Incomplete implementation (commented-out code)

**Why it occurred:** The SSO-specific paths were implemented and working, but the generic Cloud Control path was left commented out, likely from an initial implementation that was never completed.

**Contributing factors:**
- No tests for the non-SSO code path
- The function uses concrete client types (`*cloudcontrol.Client`) rather than interfaces, making it difficult to test
- Silent empty-map return rather than an error, so the bug is invisible to users

## Resolution for the Issue

**Changes made:**
- `lib/interfaces.go` - Added `CloudControlListResourcesAPI` interface defining the `ListResources` operation
- `lib/drift.go:ListAllResources` - Changed function signature from concrete types (`*cloudcontrol.Client`, `*ssoadmin.Client`, `*organizations.Client`) to interfaces (`CloudControlListResourcesAPI`, SSO admin interface, `OrganizationsListAccountsAPI`). Implemented Cloud Control `ListResources` call with manual pagination via `NextToken`, replacing the commented-out code. Resources are returned as `map[identifier]typeName`.

**Approach rationale:** Manual pagination was used instead of the SDK `NewListResourcesPaginator` to stay consistent with the pagination pattern already used in `GetDefaultStackDrift` in the same file. Using interfaces instead of concrete types enables unit testing with mock clients.

**Alternatives considered:**
- Using SDK paginator (`NewListResourcesPaginator`) - Would simplify the pagination loop, but introduces a different pattern from the rest of the file and makes mocking harder since the paginator constructor requires a `ListResourcesAPIClient` which wraps the interface differently
- Returning an error for unsupported types - Rejected because Cloud Control API supports listing most CloudFormation resource types, so the generic path is the correct approach

## Regression Test

**Test file:** `lib/drift_listallresources_test.go`
**Test name:** `TestListAllResources_NonSSOType_ReturnsResources`

**What it verifies:** That ListAllResources calls Cloud Control ListResources for non-SSO types and returns the discovered resources with proper pagination and error handling.

**Run command:** `go test ./lib/ -run TestListAllResources -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/drift.go` | Implement Cloud Control ListResources with pagination |
| `lib/interfaces.go` | Add CloudControlListResourcesAPI interface |
| `lib/drift_listallresources_test.go` | Regression tests |
| `cmd/drift.go` | No changes needed (concrete clients satisfy the interfaces) |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Commented-out code should be removed or replaced with explicit error handling
- Functions should return errors for unsupported cases rather than silent empty results
- Use interfaces instead of concrete types to enable testing
