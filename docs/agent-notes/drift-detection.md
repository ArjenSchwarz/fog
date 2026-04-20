# Drift Detection

## Architecture

Drift detection lives in `cmd/drift.go` with helpers in `cmd/helpers.go`.

### Key functions

- `separateSpecialCases()` — Builds the `logicalToPhysical` map from stack resources and exports. Also categorizes NACL, route table, and TGW resources for special-case handling.
- `checkIfResourcesAreManaged()` — Compares resources discovered via `lib.ListAllResources()` against the `logicalToPhysical` map to identify UNMANAGED resources. Builds a set of managed physical IDs from `logicalToPhysical` values for O(1) lookups, since `allresources` keys are physical IDs from the Cloud Control API.
- `checkNaclEntries()`, `checkRouteTableRoutes()`, `checkTransitGatewayRouteTableRoutes()` — Special-case drift checks for resources that need deeper template comparison.

### Data flow for unmanaged resource detection

1. `lib.ListAllResources()` returns `map[string]string` where keys are resource identifiers and values are resource type strings (e.g., `"AWS::SSO::PermissionSet"`)
2. `logicalToPhysical` maps logical resource IDs (CloudFormation) to physical resource IDs (AWS)
3. `checkIfResourcesAreManaged` builds a set of physical IDs from `logicalToPhysical` values, then checks if each resource from step 1 exists in that set
4. Resources not found are reported as UNMANAGED (unless in the ignore list via `drift.ignore-unmanaged-resources` config)

### Gotchas

- `settings` is a global `*config.Config` instance backed by viper — tests that use it cannot run in parallel
- The `allresources` map from `ListAllResources` uses different key formats depending on resource type (e.g., SSO permission sets use `"instanceArn|permissionSetArn"` composite keys)
- `driftFlags` is also global state that affects test isolation
- NACL property extractors in `lib/template.go` (e.g. `extractRuleNumber`, `extractCidrBlock`) must handle the shapes they may receive at extraction time. In the drift path, `ParseTemplateString` is invoked with stack parameter overrides (from `GetParametersMap`), which typically inlines `Ref` parameter values directly into `properties` — so a parameterized `RuleNumber` usually arrives as a numeric string (e.g. `"150"`) rather than a `{"Ref": "ParamName"}` map. Extractors must still handle both: literal scalars (`float64`/`string`, including numeric strings) and, when a `Ref` map survives, resolve it via `resolveParameterValue(refname, params)`. Skipping any supported shape silently produces a zero value and collides drift-check keys (T-834 was `extractRuleNumber` returning 0 for parameterized rule numbers, causing all entries to collapse onto `I0`/`E0`).

## Tests

- `cmd/drift_test.go` — Tests for output formatting, tag handling, field validation
- `cmd/drift_managed_test.go` — Tests for `checkIfResourcesAreManaged` value-based lookup behavior (added in T-435, corrected in T-455)
- `cmd/drift_unmanaged_test.go` — Tests for `checkIfResourcesAreManaged` with realistic physical ID data (added in T-455)
