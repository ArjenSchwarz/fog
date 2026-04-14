# Bugfix Report: deployments-directory-default

**Date:** 2025-07-14
**Status:** Fixed
**Ticket:** T-776

## Description of the Issue

`cmd/root.go` set the `deployments.directory` viper default to `[]string{"."}` (a string slice), while `lib/files.go` reads the directory using `viper.GetString("deployments.directory")`. When the underlying value is a `[]string`, `GetString` returns an empty string `""` instead of `"."`, silently breaking deployment file resolution for users who rely on defaults (no explicit config).

All other `*.directory` defaults (`templates.directory`, `tags.directory`, `parameters.directory`) were already plain strings, making this an inconsistency.

Additionally, line 179 was a duplicate `viper.SetDefault("parameters.directory", "parameters")` that was already set on line 176.

**Reproduction steps:**
1. Remove any `deployments.directory` setting from `fog.yaml`
2. Place a deployment file (e.g., `mystack.yaml`) in the current directory
3. Run a fog command that resolves a deployment file by name (e.g., `fog deploy mystack`)
4. Observe that the file is not found because `GetString` returns `""` for the slice default

**Impact:** Deployment file resolution fails silently when users rely on the default configuration. Only users who explicitly set `deployments.directory` in their config file are unaffected.

## Investigation Summary

- **Symptoms examined:** `viper.GetString()` returns `""` when the underlying default is `[]string{"."}`
- **Code inspected:** `cmd/root.go` (default registration), `lib/files.go` (directory lookup)
- **Hypotheses tested:** Confirmed that `viper.GetString` on a `[]string` default returns empty string, not the first element

## Discovered Root Cause

The default value for `deployments.directory` was set as `[]string{"."}` instead of `"."`.

**Defect type:** Type mismatch

**Why it occurred:** The deployments defaults were likely copy-pasted from the extensions config (which correctly uses `[]string`), and the directory value was wrapped in a slice by mistake.

**Contributing factors:** Viper silently returns zero-values for type mismatches rather than reporting an error, making this class of bug difficult to catch without specific tests.

## Resolution for the Issue

**Changes made:**
- `cmd/root.go:178` — Changed `[]string{"."}` to `"."` to match the string type expected by `viper.GetString()`
- `cmd/root.go:179` — Removed duplicate `viper.SetDefault("parameters.directory", "parameters")` (already set on line 176)

**Approach rationale:** Aligning the default type with how the value is consumed is the minimal, correct fix. All other `*.directory` defaults already use plain strings.

**Alternatives considered:**
- Changing `lib/files.go` to use `GetStringSlice` — rejected because it would require changing all directory lookups and is inconsistent with how templates/tags/parameters work

## Regression Test

**Test file:** `lib/files_test.go`
**Test names:** `TestReadDeploymentFileWithDefaultConfig`, `TestDeploymentsDirectoryDefaultIsString`

**What they verify:**
- `TestReadDeploymentFileWithDefaultConfig` — end-to-end test that `ReadDeploymentFile` resolves a file in `"."` when using only viper defaults
- `TestDeploymentsDirectoryDefaultIsString` — unit test that `viper.GetString("deployments.directory")` returns `"."` when using the correct default

**Run command:** `go test ./lib/ -run 'TestReadDeploymentFileWithDefaultConfig|TestDeploymentsDirectoryDefaultIsString' -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/root.go` | Fixed `deployments.directory` default from `[]string{"."}` to `"."`, removed duplicate parameters.directory line |
| `lib/files_test.go` | Added two regression tests for T-776 |

## Verification

**Automated:**
- [x] Regression tests pass
- [x] Full test suite passes
- [x] Linter passes (golangci-lint: 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- All `*.directory` viper defaults should be plain strings, never slices — add a comment in `cmd/root.go` near the defaults block noting this convention
- Consider adding a startup self-check that validates viper default types match their consumption patterns
- Viper's silent type coercion makes these bugs hard to spot; prefer explicit type assertions in tests for critical config keys

## Related

- T-776: Fix deployments.directory default type mismatch in root config
