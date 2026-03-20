# Bugfix Report: Return Errors from Lambda Handler

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `HandleRequest` function in `main.go` has a signature of `func(message EventBridgeMessage)` — it does not return an `error`. The AWS Lambda Go SDK supports handler signatures that return `error`, and when the handler does not return one, Lambda always reports the invocation as successful regardless of whether the internal report generation failed.

Additionally, `GenerateReportFromLambda` in `cmd/report.go` does not return an error. It delegates to `generateReport()` which uses `failWithError` (calling `os.Exit(1)`) on failures. In a Lambda context, `os.Exit(1)` terminates the runtime without reporting an error back to Lambda, causing silent failures.

Required environment variables (`ReportS3Bucket`, `ReportOutputFormat`) are not validated before use.

**Reproduction steps:**
1. Deploy the fog Lambda function with a missing or empty `ReportS3Bucket` environment variable
2. Trigger the Lambda via an EventBridge event
3. Lambda reports success even though no report was generated

**Impact:** Lambda invocations silently swallow all errors. Failures in report generation (missing env vars, AWS API errors, S3 write failures) are invisible to operators monitoring Lambda execution status.

## Investigation Summary

- **Symptoms examined:** `HandleRequest` returns no value; Lambda runtime cannot detect failures
- **Code inspected:** `main.go` (Lambda handler), `cmd/report.go` (`GenerateReportFromLambda`, `generateReport`), `cmd/helpers.go` (`failWithError`)
- **Hypotheses tested:** Confirmed `failWithError` calls `os.Exit(1)` which terminates the Lambda process without error reporting

## Discovered Root Cause

Three defects work together:

1. `HandleRequest` has no error return value, so the Lambda SDK cannot report failures
2. `GenerateReportFromLambda` has no error return value, so `HandleRequest` has no error to propagate
3. `generateReport` uses `failWithError`/`os.Exit(1)` instead of returning errors, which is appropriate for CLI but not for Lambda

**Defect type:** Missing error propagation

**Why it occurred:** The Lambda handler was written following the CLI pattern where `os.Exit(1)` is the standard way to report errors. In Lambda, errors must be returned from the handler function.

**Contributing factors:** `generateReport` was designed for CLI use only, calling `failWithError` directly rather than returning errors for the caller to handle.

## Resolution for the Issue

**Changes made:**

1. `main.go` — Changed `HandleRequest` signature from `func(message EventBridgeMessage)` to `func(message EventBridgeMessage) error`. Added validation for required environment variables `ReportS3Bucket` and `ReportOutputFormat`, returning descriptive errors when they are missing. The return value from `GenerateReportFromLambda` is now propagated to the Lambda runtime.

2. `cmd/report.go` — Changed `generateReport` signature from `func()` to `func() error`, replacing all `failWithError(err)` calls with `return err`. Changed `GenerateReportFromLambda` signature from `func(...) ` to `func(...) error`, returning the error from `generateReport`. Updated `stackReport` (the CLI path) to call `failWithError` on the returned error, preserving existing CLI behavior.

**Approach rationale:** By making `generateReport` return errors instead of calling `os.Exit`, both the CLI and Lambda paths can handle errors appropriately — the CLI path calls `failWithError` (which prints and exits), while the Lambda path returns the error to the runtime. This avoids duplicating report generation logic.

## Regression Test

**Test file:** `main_test.go`
**Test names:** `TestHandleRequestReturnsErrorOnMissingEnvVars`, `TestHandleRequestReturnsErrorOnMissingBucket`, `TestHandleRequestReturnsErrorOnMissingFormat`

**What they verify:** Confirm that `HandleRequest` returns an error when required environment variables (`ReportS3Bucket`, `ReportOutputFormat`) are missing.

**Run command:** `go test . -run TestHandleRequest -v`

## Affected Files

| File | Change |
|------|--------|
| `main.go` | Update `HandleRequest` to return `error`, validate env vars |
| `cmd/report.go` | Update `GenerateReportFromLambda` to return `error`, update `generateReport` to return `error` |
| `main_test.go` | Regression tests for error return from Lambda handler |
| `specs/bugfixes/return-errors-from-lambda-handler/report.md` | Bugfix report |
