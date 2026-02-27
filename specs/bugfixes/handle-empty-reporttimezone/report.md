# Bugfix Report: Handle Empty ReportTimezone in Lambda Reports

**Date:** 2026-02-27
**Status:** Fixed

## Description of the Issue

When the `ReportTimezone` environment variable is not set (or set to an empty string), the Lambda handler passes an empty string to `GenerateReportFromLambda`, which unconditionally calls `viper.Set("timezone", "")`. This overrides the viper default of `"Local"`, causing `GetTimezoneLocation()` to call `time.LoadLocation("")` which returns an error and triggers a panic.

**Reproduction steps:**
1. Deploy the fog Lambda function without setting the `ReportTimezone` environment variable
2. Trigger the Lambda via an EventBridge event
3. Observe a panic from `GetTimezoneLocation` due to `time.LoadLocation("")` failing

**Impact:** Any Lambda deployment that omits the optional `ReportTimezone` environment variable will crash on every invocation, making automated reporting non-functional.

## Investigation Summary

- **Symptoms examined:** `time.LoadLocation("")` returns an error; `GetTimezoneLocation` panics on that error
- **Code inspected:** `main.go` (Lambda handler), `cmd/report.go` (`GenerateReportFromLambda`), `config/config.go` (`GetTimezoneLocation`), `cmd/root.go` (viper defaults)
- **Hypotheses tested:** Confirmed that `viper.Set` overrides `viper.SetDefault` even with an empty string

## Discovered Root Cause

`GenerateReportFromLambda` unconditionally calls `viper.Set("timezone", timezone)` regardless of whether the timezone string is empty. `viper.Set` takes precedence over `viper.SetDefault`, so the default `"Local"` is overridden with `""`. `time.LoadLocation("")` then returns an error, which `GetTimezoneLocation` propagates as a panic.

**Defect type:** Missing input validation

**Why it occurred:** The function assumed the caller would always provide a non-empty timezone value. The Lambda CloudFormation template defines `ReportTimezone` as an optional parameter with no default, so it can easily be empty.

**Contributing factors:** The `GetTimezoneLocation` function uses `panic` for error handling rather than returning an error, making the failure mode severe for any invalid input.

## Resolution for the Issue

**Changes made:**
- `cmd/report.go:113-116` — Guard `viper.Set("timezone", ...)` with an emptiness check so the viper default `"Local"` is preserved when no timezone is specified

**Approach rationale:** This is the minimal fix that preserves the existing default behaviour. An empty env var is treated as "not set" which matches user expectations for optional configuration.

**Alternatives considered:**
- Validating in `GetTimezoneLocation` and falling back to `time.Local` — would change the function's contract and mask configuration errors for genuinely invalid timezone strings
- Validating in `main.go` at the call site — would duplicate the concern; the setting logic belongs in `GenerateReportFromLambda`

## Regression Test

**Test file:** `cmd/report_test.go`
**Test name:** `TestGenerateReportFromLambdaEmptyTimezone`

**What it verifies:** Confirms that an empty timezone string preserves the viper default `"Local"`, while non-empty timezone strings correctly override it.

**Run command:** `go test ./cmd/ -run TestGenerateReportFromLambdaEmptyTimezone -v`

## Affected Files

| File | Change |
|------|--------|
| `cmd/report.go` | Guard `viper.Set("timezone", ...)` with emptiness check |
| `cmd/report_test.go` | Add regression test for empty timezone handling |
| `specs/bugfixes/handle-empty-reporttimezone/report.md` | Bugfix report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed `time.LoadLocation("Local")` succeeds (the default path)
- Confirmed `time.LoadLocation("")` fails (the bug path)

## Prevention

**Recommendations to avoid similar bugs:**
- Validate environment variable values before passing them to configuration setters — treat empty strings as unset for optional config
- Consider returning errors from `GetTimezoneLocation` instead of panicking, to allow graceful error handling
- Add integration test coverage for Lambda handler paths with missing optional environment variables

## Related

- Transit ticket: T-262
