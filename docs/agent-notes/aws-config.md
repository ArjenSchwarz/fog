# AWS Config Loading

## Entry Point: `config.DefaultAwsConfig`

`config/awsconfig.go` `DefaultAwsConfig(ctx, Config)` is the single entry point used by every `cmd/*` command to build an `AWSConfig` (profile, region, account ID, alias, caller identity). It wraps `external.LoadDefaultConfig` from the AWS SDK v2 config package.

## Case Sensitivity Rules

When reading fog config values used by `DefaultAwsConfig`:

- **Profile** — use `GetString` (case-preserving). AWS shared config profile names in `~/.aws/config` and `~/.aws/credentials` are case-sensitive. A profile named `ProdAdmin` will not match `prodadmin`. Reading profile via `GetLCString` was a real bug (T-880) that prevented users with mixed-case profile names from using fog at all.
- **Region** — currently uses `GetLCString`. AWS region codes are lowercase by convention (`us-east-1`), so this works in practice, but prefer `GetString` if ever touching this code path again.

The helper `sharedConfigProfile(profileReader)` exists so the case-preserving behaviour can be exercised in unit tests without spinning up AWS SDK loaders.

## Testing Constraint

`DefaultAwsConfig` calls `external.LoadDefaultConfig` directly (not through the `AWSConfigLoader` interface defined in `config/interfaces.go`). It also calls `setCallerInfo` (STS) and `setAlias` (IAM) against the loaded config. As a result, `TestDefaultAwsConfig` is skipped — it would require a wholesale refactor to inject AWS clients.

When you need to test something in the profile/region resolution path, factor the logic into a helper that takes a small interface (see `profileReader`) rather than refactoring the whole entry point.
