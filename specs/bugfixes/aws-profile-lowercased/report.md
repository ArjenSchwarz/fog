# Bugfix Report: AWS Profile Names Are Lowercased Before Loading

**Date:** 2026-04-20
**Status:** Fixed

## Description of the Issue

`config.DefaultAwsConfig` read the AWS profile name with `config.GetLCString("profile")`, which lowercases the value. AWS shared config profile names in `~/.aws/config` and `~/.aws/credentials` are case-sensitive, so a configured profile such as `ProdAdmin` was looked up as `prodadmin`. This caused AWS SDK configuration loading to fail with "profile not found" errors, or — when a different lower-case profile happened to exist — silently select the wrong profile.

**Reproduction steps:**
1. Create an AWS profile with mixed-case name, e.g. `[profile ProdAdmin]` in `~/.aws/config`
2. Run fog with `--profile ProdAdmin` (or set the value in a fog config file)
3. The AWS SDK reports the profile cannot be found, because fog passed `prodadmin` to `external.WithSharedConfigProfile`

**Impact:** Users with mixed-case or upper-case AWS profile names cannot use fog with that profile. If a lower-cased sibling profile exists, fog selects it silently and can operate against the wrong AWS account.

## Investigation Summary

- **Symptoms examined:** Profile name passed to AWS SDK is lowercased despite being written with mixed case in the config source.
- **Code inspected:** `config/awsconfig.go` `DefaultAwsConfig`, `config/config.go` `GetLCString`/`GetString`, all call sites of `DefaultAwsConfig` in `cmd/`.
- **Hypotheses tested:** Confirmed `Config.GetLCString` applies `strings.ToLower` to the stored value. `DefaultAwsConfig` called it three times for `"profile"` (conditional check, `ProfileName` assignment, `WithSharedConfigProfile` argument).

## Discovered Root Cause

`DefaultAwsConfig` used `config.GetLCString("profile")` to read the profile:

```go
if config.GetLCString("profile") != "" {
    awsConfig.ProfileName = config.GetLCString("profile")
    cfg, err := external.LoadDefaultConfig(ctx, external.WithSharedConfigProfile(config.GetLCString("profile")))
    ...
}
```

`GetLCString` unconditionally lowercases the value. Because AWS profile lookup is case-sensitive, the SDK tried to load the wrong profile.

**Defect type:** Incorrect normalization — case-insensitive read used for a case-sensitive identifier.

**Why it occurred:** `GetLCString` is appropriate for fog-level values that are known to be case-insensitive (output format, table style). It was used here without consideration of the downstream AWS semantics.

**Contributing factors:** No regression coverage for mixed-case profile names. Region lowercasing works in practice because AWS region codes are lowercase by convention, masking the problem for that field.

## Resolution for the Issue

**Changes made:**
- `config/awsconfig.go` — Extracted a small helper `sharedConfigProfile(profileReader)` that reads the profile via `GetString` (preserving case). `DefaultAwsConfig` now calls the helper once, stores the result in a local `profile` variable, and reuses it for the `ProfileName` field and `external.WithSharedConfigProfile`. Region lookup continues to use `GetLCString` because AWS region codes are lowercase.
- `config/awsconfig_test.go` — Added `TestSharedConfigProfile_PreservesCase` covering mixed-case, upper-case, lower-case, and alphanumeric profile names, plus the empty-profile case. Fixed the test-local `mockConfig.GetLCString` to match production behaviour (it previously returned the stored value unchanged, which would have hidden the regression).

**Approach rationale:** A focused helper with a tiny interface keeps `DefaultAwsConfig`'s behaviour unchanged except for the case handling, and makes the behaviour testable without refactoring the broader AWS-config loading path. Only profile reading is changed — region reading keeps its existing `GetLCString` call because lower-casing region codes has never caused problems and the ticket explicitly asks to keep intentional normalization intact.

**Alternatives considered:**
- Replacing the three `GetLCString("profile")` calls inline with `GetString("profile")`. Functionally equivalent but gives no natural test seam and repeats the lookup three times.
- Introducing a full `AWSConfigLoader` injection path for `DefaultAwsConfig` and asserting on `LoadOptions.SharedConfigProfile`. Rejected as out of scope — it would require refactoring the entry point and every caller in `cmd/` for no extra safety over the simple helper.

## Regression Test

**Test file:** `config/awsconfig_test.go`
**Test name:** `TestSharedConfigProfile_PreservesCase`

**What it verifies:**
1. Mixed-case profile names (`ProdAdmin`) are returned unchanged.
2. Upper-case profile names (`PRODUCTION`) are returned unchanged.
3. Lower-case profile names (`dev`) continue to work.
4. Profile names with digits and non-letter characters (`Account-123_Admin`) are returned unchanged.
5. An unset profile returns an empty string.

**Run command:** `go test ./config -run TestSharedConfigProfile_PreservesCase -v`

Before the fix: mixed/upper/alphanumeric cases fail because `GetLCString` lowercases the value.
After the fix: all cases pass.

## Affected Files

| File | Change |
|------|--------|
| `config/awsconfig.go` | Added `profileReader` interface and `sharedConfigProfile` helper; `DefaultAwsConfig` now reads the profile through the helper (case-preserving) |
| `config/awsconfig_test.go` | Added `TestSharedConfigProfile_PreservesCase`; corrected `mockConfig.GetLCString` to lowercase like the production implementation |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Integration suite passes (`INTEGRATION=1 go test ./...`)
- [x] `go fmt ./config/...` clean
- [x] `go vet ./config/...` clean
- [x] `golangci-lint run ./config/...` clean

## Prevention

**Recommendations to avoid similar bugs:**
- Default to `GetString` for any value that is forwarded to an external system; only reach for `GetLCString` when the value is known to be normalized (output format, table style, fog-internal enums).
- When adding a new use of `GetLCString`, document the reason in a short comment so reviewers can catch case-sensitivity mistakes.

## Related

- Transit ticket: T-880
