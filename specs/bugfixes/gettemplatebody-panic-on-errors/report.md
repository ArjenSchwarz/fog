# Bugfix Report: GetTemplateBody Panic on Errors

**Date:** 2026-02-27
**Status:** Fixed
**Transit:** T-254

## Description of the Issue

`lib.GetTemplateBody` panicked instead of returning an error when the AWS `GetTemplate` API call failed or when the response contained a nil `TemplateBody`. This caused the entire CLI to crash rather than presenting a user-friendly error message.

**Reproduction steps:**
1. Call any command that retrieves a CloudFormation template (e.g., `fog drift`)
2. Target a stack where the `GetTemplate` API returns an error (e.g., permissions issue, non-existent stack) or a nil `TemplateBody`
3. Observe an unrecoverable panic crash

**Impact:** High — any AWS API failure in template retrieval caused a full process crash with a stack trace instead of a graceful error message. This affected the `drift` command and any future commands using `GetTemplateBody`.

## Investigation Summary

- **Symptoms examined:** `GetTemplateBody` used `panic(err)` for API errors and dereferenced `*result.TemplateBody` without a nil check
- **Code inspected:** `lib/template.go:GetTemplateBody`, `cmd/drift.go` (the only caller), `lib/template_body_test.go`
- **Hypotheses tested:** Confirmed the function had no error return path — both failure modes (API error and nil body) would panic

## Discovered Root Cause

The function used `panic(err)` for error handling and performed an unchecked nil pointer dereference on `result.TemplateBody`.

**Defect type:** Missing error handling / nil pointer dereference

**Why it occurred:** The original implementation used panic as a quick error propagation mechanism rather than idiomatic Go error returns.

**Contributing factors:** The function signature returned only `CfnTemplateBody` with no `error` return value, making it impossible to signal failures to callers gracefully.

## Resolution for the Issue

**Changes made:**
- `lib/template.go:206` — Changed return type from `CfnTemplateBody` to `(CfnTemplateBody, error)`, replaced `panic(err)` with wrapped error return, added nil check for `TemplateBody`
- `cmd/drift.go:176` — Updated caller to handle the new error return with `log.Fatal(err)`
- `lib/template_body_test.go` — Updated test cases: replaced `wantPanic`/`assert.Panics` with `wantErr`/`require.Error`, added "nil TemplateBody returns error" test case

**Approach rationale:** Idiomatic Go uses error returns. The caller (`cmd/drift.go`) already used `log.Fatal` for other errors in the same function, so the pattern is consistent.

**Alternatives considered:**
- Recover-based panic handling in callers — rejected because it hides the root issue and is not idiomatic Go

## Regression Test

**Test file:** `lib/template_body_test.go`
**Test names:** `TestGetTemplateBody/API_error_returns_error`, `TestGetTemplateBody/nil_TemplateBody_returns_error`

**What it verifies:** That `GetTemplateBody` returns a descriptive error (not a panic) when the API fails or when `TemplateBody` is nil.

**Run command:** `go test ./lib/ -run TestGetTemplateBody -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/template.go` | Changed `GetTemplateBody` to return `(CfnTemplateBody, error)` with nil check |
| `cmd/drift.go` | Updated caller to handle error return |
| `lib/template_body_test.go` | Updated tests for error returns, added nil body test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run`)
- [x] Build succeeds (`go build -o fog`)

## Prevention

**Recommendations to avoid similar bugs:**
- Avoid using `panic` for expected error conditions in library code; always use error returns
- Use `golangci-lint` with `nilerr` or similar checks to catch unchecked nil dereferences
- When wrapping AWS SDK calls, always check both the error return and nil-ness of response fields

## Related

- Transit ticket: T-254
