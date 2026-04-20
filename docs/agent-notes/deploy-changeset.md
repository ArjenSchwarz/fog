# Deploy Changeset Mode

`DeployFlags.DeployChangeset` / `--deploy-changeset` tells `fog deploy` to
execute an existing changeset instead of creating a new one.

## Flow

`deployTemplate()` in `cmd/deploy.go` branches on `deployFlags.DeployChangeset`
after `runPrechecks`:

- When the flag is unset (default), it calls `createAndShowChangeset()`,
  handles `--dry-run` and `--create-changeset`, and then deploys the freshly
  created changeset.
- When the flag is set, it calls `runDeployChangesetFlow()` (in
  `cmd/deploy_helpers.go`), which uses `fetchChangesetFunc` (wraps
  `fetchChangeset` in `cmd/deploy.go`). That helper calls
  `DeployInfo.GetChangeset` + `AddChangeset` (in `lib/stacks.go`) to load the
  existing changeset by name, attaches it to `deployment.Changeset` /
  `deployment.CapturedChangeset`, and displays it with `showChangesetFunc`.
  The flow then falls through to `confirmAndDeployChangeset` exactly like the
  normal path.

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
- `TestDeployTemplate_DeployChangeset_SkipsCreation` — verifies the new flow
  uses `fetchChangesetFunc`, does not call `createChangesetFunc`, populates
  `Changeset`/`CapturedChangeset`, and hands off to
  `confirmAndDeployChangeset`.

Tests stub `fetchChangesetFunc` / `createChangesetFunc` / `showChangesetFunc`
just like the existing `createAndShowChangeset` tests do. When adding new
tests around this flow, restore all function-variable pointers in `defer` to
avoid leaking state across test files.

Original bug: T-865 (`--deploy-changeset flag is ignored by deploy command`).
