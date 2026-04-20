# Deploy Changeset Mode

`DeployFlags.DeployChangeset` / `--deploy-changeset` tells `fog deploy` to
execute an existing changeset instead of creating a new one.

## Flow

`deployTemplate()` in `cmd/deploy.go` branches on `deployFlags.DeployChangeset`:

- Prechecks are skipped when `--deploy-changeset` is set. They use
  `$TEMPLATEPATH` substitution against the local template path, which is not
  populated in this mode (no template is loaded). Running them would produce
  unreliable results.
- When the flag is unset (default), it runs `runPrechecks`, then calls
  `createAndShowChangeset()`, handles `--dry-run` and `--create-changeset`,
  and then deploys the freshly created changeset.
- When the flag is set, it calls `runDeployChangesetFlow()` (in
  `cmd/deploy_helpers.go`), which uses `fetchChangesetFunc` (wraps
  `fetchChangeset` in `cmd/deploy.go`). That helper calls
  `DeployInfo.GetChangeset` + `AddChangeset` (in `lib/stacks.go`) to load the
  existing changeset by name, attaches it to `deployment.Changeset` /
  `deployment.CapturedChangeset`, and displays it with `showChangesetFunc`.
  The flow then falls through to `confirmAndDeployChangeset` exactly like the
  normal path.

`fetchChangeset` uses `osExitFunc` (not `os.Exit` / `log.Fatalln` directly) so
tests can intercept its failure paths, and it distinguishes retrieval errors
(`DeployChangesetMessageRetrieveFailed`) from "no results"
(`DeployChangesetMessageNotFound`). It returns `nil` after `osExitFunc(1)` so
test stubs that don't actually exit don't cause a nil-dereference downstream.
`runDeployChangesetFlow` nil-guards the fetch result for the same reason.

`prepareDeployment()` skips `setDeployTemplate`, `setDeployTags`,
`setDeployParameters` when `--deploy-changeset` is set — the template and
inputs are already captured inside the existing changeset.

## Validation

`DeployFlags.Validate()` (in `cmd/flaggroups.go`) enforces:

- `--deploy-changeset` requires `--changeset` (so the existing changeset has a
  name to look up).
- `--deploy-changeset` is mutually exclusive with `--dry-run`,
  `--create-changeset`, `--template`, `--parameters`, `--tags`, and
  `--deployment-file`. Those flags only apply to creating a new changeset.

## Tests

`cmd/deploy_changeset_flag_test.go` covers:

- `TestDeployFlags_Validate_DeployChangeset` — required and mutually exclusive
  flag combinations.
- `TestRunDeployChangesetFlow_DeployChangeset_SkipsCreation` — verifies the
  new flow uses `fetchChangesetFunc`, does not call `createChangesetFunc`,
  populates `Changeset`/`CapturedChangeset`, and hands off to
  `confirmAndDeployChangeset`.
- `TestRunDeployChangesetFlow_NilFetchReturn` — verifies that if
  `fetchChangesetFunc` returns nil the flow bails out without dereferencing
  the nil changeset.

Tests stub `fetchChangesetFunc` / `createChangesetFunc` / `showChangesetFunc`
just like the existing `createAndShowChangeset` tests do. When adding new
tests around this flow, restore all function-variable pointers in `defer` to
avoid leaking state across test files.

Original bug: T-865 (`--deploy-changeset flag is ignored by deploy command`).
