# Drift Detection

## Architecture

Drift detection lives in `cmd/drift.go` with helpers in `cmd/helpers.go`.

### Key functions

- `separateSpecialCases()` — Builds the `logicalToPhysical` map from stack resources and exports. Also categorizes NACL, route table, and TGW resources for special-case handling.
- `checkIfResourcesAreManaged()` — Compares resources discovered via `lib.ListAllResources()` against the `logicalToPhysical` map to identify UNMANAGED resources. Uses map key lookup on `logicalToPhysical` (keys = logical IDs, values = physical IDs).
- `checkNaclEntries()`, `checkRouteTableRoutes()`, `checkTransitGatewayRouteTableRoutes()` — Special-case drift checks for resources that need deeper template comparison.

### Data flow for unmanaged resource detection

1. `lib.ListAllResources()` returns `map[string]string` where keys are resource identifiers and values are resource type strings (e.g., `"AWS::SSO::PermissionSet"`)
2. `logicalToPhysical` maps logical resource IDs (CloudFormation) to physical resource IDs (AWS)
3. `checkIfResourcesAreManaged` checks if each resource from step 1 exists as a **key** in `logicalToPhysical`
4. Resources not found are reported as UNMANAGED (unless in the ignore list via `drift.ignore-unmanaged-resources` config)

### Gotchas

- `settings` is a global `*config.Config` instance backed by viper — tests that use it cannot run in parallel
- The `allresources` map from `ListAllResources` uses different key formats depending on resource type (e.g., SSO permission sets use `"instanceArn|permissionSetArn"` composite keys)
- `driftFlags` is also global state that affects test isolation

## Tests

- `cmd/drift_test.go` — Tests for output formatting, tag handling, field validation
- `cmd/drift_managed_test.go` — Tests for `checkIfResourcesAreManaged` key lookup behavior (added in T-435)
