# Bugfix Report: Validate Lambda stack-id before report generation

**Date:** 2026-04-28
**Status:** Fixed

## Description of the Issue

The Lambda entrypoint accepted EventBridge payloads with an empty or missing `detail.stack-id`.
When that happened, `HandleRequest` passed the empty value into report generation, allowing downstream
code to treat the request like an account-wide `all-stacks` report instead of rejecting the invalid event.

**Reproduction steps:**
1. Configure the Lambda environment with `ReportS3Bucket` and `ReportOutputFormat`.
2. Invoke `HandleRequest` with an EventBridge message whose `detail.stack-id` is empty or omitted.
3. Observe that the handler forwards the empty stack identifier instead of failing fast.

**Impact:** Invalid CloudFormation events can trigger the wrong report scope and produce an unintended account-wide report.

## Investigation Summary

Reviewed the Lambda handler path and the Lambda-specific report entrypoint to trace how `detail.stack-id`
flows through report generation.

- **Symptoms examined:** Empty `detail.stack-id` values did not cause handler validation failures.
- **Code inspected:** `main.go`, `main_test.go`, and `cmd/report.go`.
- **Hypotheses tested:** Confirmed that handler validation existed for required environment variables but not for the EventBridge stack identifier.

## Discovered Root Cause

`HandleRequest` validated Lambda configuration but trusted the incoming EventBridge payload. An empty
`detail.stack-id` therefore reached `GenerateReportFromLambda`, which uses the provided value as the report
target without its own guard in this call path.

**Defect type:** Missing validation

**Why it occurred:** Handler-side validation was added for environment variables, but the required event payload field was overlooked.

**Contributing factors:** The handler directly called report generation, making it hard to regression-test whether invalid events were rejected before any downstream work started.

## Resolution for the Issue

**Changes made:**
- `main.go:46` - Introduced an injectable `generateReportFromLambda` function reference so handler behaviour can be isolated in unit tests.
- `main.go:103` - Added `detail.stack-id` validation with whitespace trimming and fail-fast error handling before report generation starts.
- `main_test.go:51` - Added a regression test proving empty EventBridge stack identifiers return an error and never invoke report generation.
- `main_test.go:82` - Added a success-path test confirming valid stack identifiers are trimmed and forwarded to report generation.

**Approach rationale:** Validate the event payload at the Lambda boundary, because that is the narrowest point where malformed EventBridge data can be rejected before any AWS-facing report logic runs. The small injection hook keeps the tests fast and deterministic without changing the production code path.

**Alternatives considered:**
- Add validation inside `cmd.GenerateReportFromLambda` - not chosen because the bug specifically originates in the Lambda handler path and should fail before invoking report generation.

## Regression Test

**Test file:** `main_test.go`
**Test name:** `TestHandleRequestReturnsErrorOnMissingStackID`

**What it verifies:** `HandleRequest` rejects an EventBridge payload with an empty `detail.stack-id` before report generation is invoked.

**Run command:** `go test ./... -v`

## Affected Files

| File | Change |
|------|--------|
| `main.go` | Added stack-id validation and a test seam for the Lambda report call |
| `main_test.go` | Added regression and success-path coverage for Lambda stack-id handling |
| `specs/bugfixes/validate-lambda-stack-id-before-report-generation/report.md` | Documented investigation, fix, and verification |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- `go build -o fog` completed successfully to confirm the binary still builds.

## Prevention

**Recommendations to avoid similar bugs:**
- Validate required EventBridge fields at the Lambda boundary before invoking business logic.
- Keep handler dependencies injectable so fast-fail behaviour can be regression-tested without AWS calls.
- Add regression tests for malformed event payloads whenever Lambda event schemas change.

## Related

- Transit ticket `T-1072`
