# Bugfix Report: Escaped Spaces in Precheck Command Parsing

**Date:** 2026-03-29
**Status:** Fixed
**Ticket:** T-612

## Description of the Issue

`splitShellArgs` correctly handled quoted strings and escaped quotes inside double-quoted strings, but did not handle backslash-escaped spaces outside quotes. This meant commands like `cfn-lint -t path\ with\ spaces/template.yaml` were incorrectly split into separate arguments at each escaped space.

**Reproduction steps:**
1. Configure a precheck command with a path containing backslash-escaped spaces, e.g. `cfn-lint -t path\ with\ spaces/template.yaml`
2. Run a deployment that triggers prechecks
3. Observe that the command is split into `["cfn-lint", "-t", "path\", "with\", "spaces/template.yaml"]` instead of `["cfn-lint", "-t", "path with spaces/template.yaml"]`

**Impact:** Any user whose template paths contained spaces and who relied on backslash escaping (a standard shell convention) would have broken precheck commands.

## Investigation Summary

- **Symptoms examined:** `splitShellArgs` treats `\ ` as a literal backslash followed by a space delimiter
- **Code inspected:** `lib/files.go` — the `splitShellArgs` function's switch cases
- **Hypotheses tested:** The backslash-escape case only existed for `\"` inside double quotes; no handling existed for backslash outside quotes

## Discovered Root Cause

The `splitShellArgs` parser only handled backslash as an escape character inside double-quoted strings (for `\"`). Outside of quotes, backslashes were passed through as literal characters via the `default` case, and the following space was still treated as an argument delimiter.

**Defect type:** Missing feature / incomplete parser logic

**Why it occurred:** The original implementation focused on quoted-string handling and escaped quotes within double-quoted strings, but did not account for the common shell convention of escaping spaces with backslashes outside of quotes.

**Contributing factors:** The function comment didn't mention backslash-escaped spaces, so the omission wasn't obvious during review.

## Resolution for the Issue

**Changes made:**
- `lib/files.go:109-111` — Added a new case in the switch statement to handle backslash escapes outside quotes. When a backslash is encountered outside single and double quotes and is followed by another character, the next character is consumed literally.

**Approach rationale:** This mirrors standard POSIX shell behaviour where `\X` outside quotes treats `X` as a literal character. The fix is minimal (3 lines) and correctly ordered before the quote-handling cases so it doesn't interfere with the existing `\"` handling inside double quotes.

**Alternatives considered:**
- Using a full shell lexer library — overkill for the limited parsing needs
- Only handling `\ ` (backslash-space) — too narrow; handling all backslash escapes outside quotes is more correct and prevents future edge cases

## Regression Test

**Test file:** `lib/files_test.go`
**Test names:** Six new cases in `TestSplitShellArgs`

**What they verify:**
- `Escaped space outside quotes` — the primary bug scenario (`path\ with\ spaces`)
- `Multiple escaped spaces in different arguments` — multiple args with escaped spaces
- `Mixed escaped spaces and quoted strings` — interaction between escape styles
- `Escaped space at start of argument` — edge case with leading escaped space
- `Backslash followed by non-space outside quotes` — `\\` produces literal `\`
- `Trailing backslash preserved` — a trailing `\` with nothing following is kept as-is

**Run command:** `go test ./lib -run TestSplitShellArgs -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/files.go` | Added backslash-escape handling outside quotes in `splitShellArgs` |
| `lib/files_test.go` | Added 6 regression test cases for escaped-space parsing |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linters pass (`golangci-lint run` — 0 issues)

## Prevention

**Recommendations to avoid similar bugs:**
- When implementing shell-like parsing, reference POSIX shell quoting rules as the baseline specification
- Consider property-based / fuzz testing for parsers to catch edge cases
- Document which shell features are and are not supported in the function comment

## Related

- T-378: Original quoted-argument parsing fix for `splitShellArgs`
