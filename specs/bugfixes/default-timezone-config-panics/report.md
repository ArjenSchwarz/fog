# Bugfix Report: Default Timezone Config Panics

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

The `GetTimezoneLocation()` method in `config/config.go` called `time.LoadLocation` on the configured timezone string and panicked if an error was returned. When the timezone configuration was unset (empty string) or set to an invalid value, `time.LoadLocation` returned an error, causing the entire CLI to crash with a panic.

**Reproduction steps:**
1. Run any fog command that displays timestamps (e.g. `fog history`) without a `timezone` config value set and without the viper default from `root.go`
2. Alternatively, set `timezone: "Invalid/Timezone"` in `fog.yaml`
3. Observe the CLI crashes with a panic

**Impact:** High — any misconfiguration or missing timezone setting causes an unrecoverable crash, providing a poor user experience with no actionable error message.

## Investigation Summary

- **Symptoms examined:** `GetTimezoneLocation()` panics when `time.LoadLocation` fails
- **Code inspected:** `config/config.go:137-143`, `cmd/root.go:167` (viper default), `config/config_test.go` (existing tests asserted panic as expected behaviour)
- **Hypotheses tested:** The viper default `"Local"` set in `cmd/root.go` protects against empty strings in normal CLI usage, but the config package is used independently (e.g. from Lambda via `GenerateReportFromLambda`) where defaults may not be set. Invalid timezone strings always cause a panic regardless.

## Discovered Root Cause

The `GetTimezoneLocation()` function used `panic(err)` as its error handling strategy for `time.LoadLocation` failures. This is inappropriate for a recoverable configuration error.

**Defect type:** Missing input validation / inappropriate error handling

**Why it occurred:** The original implementation assumed the timezone would always be valid. The viper default of `"Local"` in `cmd/root.go` masked the empty-string case during normal CLI usage, but the function is also called from code paths (like Lambda report generation) where defaults may not be initialized.

**Contributing factors:** The existing test suite validated that the panic occurred on invalid input, treating the panic as intended behaviour rather than a bug.

## Resolution for the Issue

**Changes made:**
- `config/config.go:82-86` — Add `cachedTimezone` and `cachedLocation` fields to Config struct for timezone caching
- `config/config.go:137-162` — Replace panic with graceful fallback to `time.Local`. Cache the resolved location so repeated calls in loops avoid redundant `LoadLocation` lookups and don't spam the log with repeated warnings.

**Approach rationale:** Falling back to local timezone is the most sensible default — it matches user expectations (times shown in their system timezone) and is what `time.LoadLocation("Local")` returns. A warning log message surfaces the misconfiguration without crashing. Caching prevents log spam since `GetTimezoneLocation()` is called in event-processing loops.

**Alternatives considered:**
- Returning an error from `GetTimezoneLocation()` — rejected because it would require changing the function signature and updating all ~15 call sites, a much larger change for minimal benefit since a sensible fallback exists.
- Silently falling back without warning — rejected because users should know their timezone config is invalid.

## Regression Test

**Test file:** `config/config_test.go`
**Test names:** `TestConfig_GetTimezoneLocation/invalid_timezone_falls_back_to_local`, `TestConfig_GetTimezoneLocation/empty_timezone_falls_back_to_local`, `TestConfig_GetTimezoneLocation/unset_timezone_falls_back_to_local`

**What it verifies:** That invalid, empty, and unset timezone configurations all return `time.Local` instead of panicking.

**Run command:** `go test ./config/... -v -run TestConfig_GetTimezoneLocation`

## Affected Files

| File | Change |
|------|--------|
| `config/config.go` | Replace panic with graceful fallback to `time.Local` with warning log |
| `config/config_test.go` | Replace panic assertion with fallback assertions; add empty/unset test cases |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed the warning message is printed for invalid timezone values
- Confirmed valid timezones still work correctly

## Prevention

**Recommendations to avoid similar bugs:**
- Prefer returning errors or using sensible defaults over `panic()` for configuration validation
- Treat panics in tests as a code smell — if a function panics on bad input, consider whether it should handle the error gracefully instead
- Configuration-dependent functions should be defensive about missing or invalid values

## Related

- Transit ticket: T-82
