# Bugfix Report: Escape Regex Metacharacters in Stack/Export Glob Filters

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

Glob-style filters used to match stack names and export names treated regex metacharacters (`.`, `+`, `[`, `]`, `(`, `)`, `?`, `^`, `$`, `{`, `}`, `|`, `\`) as regex operators instead of literal characters. Only `*` was intended to be a wildcard, but the conversion from glob to regex was done via a naive `strings.ReplaceAll(input, "*", ".*")` without escaping other metacharacters first.

**Reproduction steps:**
1. Have stacks named `stack.v2-prod` and `stackXv2-prod`.
2. Run a fog command that filters stacks with pattern `stack.v2-*`.
3. Observe that both stacks match, because `.` is treated as regex any-char rather than a literal dot.

**Impact:** False positive matches in stack, export, and resource filtering. Any stack or export name containing regex metacharacters would produce incorrect filter results, potentially returning unrelated stacks in reports, exports, dependencies, and resource listings.

## Investigation Summary

Searched all call sites that convert glob patterns to regex.

- **Symptoms examined:** Glob patterns with `.` in stack names match unintended stacks.
- **Code inspected:** `lib/stacks.go`, `lib/outputs.go`, `lib/resources.go`, `cmd/dependencies.go`.
- **Hypotheses tested:** Confirmed that `strings.ReplaceAll` alone does not escape metacharacters.

## Discovered Root Cause

The glob-to-regex conversion used `strings.ReplaceAll(input, "*", ".*")` without first escaping regex metacharacters in the input. This caused characters like `.`, `+`, `[`, etc. to be interpreted as regex operators.

**Defect type:** Input sanitization defect (missing escaping).

**Why it occurred:**
- Why did the filter produce false matches? Because `.` in `stack.v2-*` was treated as regex any-char.
- Why was `.` treated as regex? Because the input was not escaped before wildcard expansion.
- Why was escaping missing? The original implementation only considered `*` as special, not other regex metacharacters.

**Contributing factors:** The same pattern was copy-pasted to 5 locations across 4 files, so the bug was replicated each time.

## Resolution for the Issue

**Changes made:**
- `lib/glob.go` (new) - Added `GlobToRegex` function that uses `regexp.QuoteMeta` to escape all metacharacters, then replaces the escaped wildcard (`\*`) with `.*`.
- `lib/stacks.go` - Replaced inline regex construction with `GlobToRegex` call.
- `lib/outputs.go` - Replaced inline regex construction with `GlobToRegex` calls for both stack and export filters.
- `lib/resources.go` - Replaced inline regex construction with `GlobToRegex` call.
- `cmd/dependencies.go` - Replaced inline regex construction with `lib.GlobToRegex` call.

**Approach rationale:** Centralizing the glob-to-regex conversion into a single exported function eliminates code duplication, ensures consistent behavior, and makes it easy to test the conversion logic in isolation.

**Alternatives considered:**
- Fix each call site individually by adding `regexp.QuoteMeta` inline - rejected because it perpetuates duplication and is harder to test.
- Use `filepath.Match` or `path.Match` instead of regex - rejected because these don't support `**` and have different semantics; the existing glob behavior (where `*` matches any characters including `/`) is correct for stack name matching.

## Regression Test

**Test file:** `lib/glob_test.go`
**Test names:**
- `TestGlobToRegex` - 15 subtests covering all common metacharacters (`.`, `+`, `[`, `]`, `(`, `)`, `?`, `^`, `$`, `|`, `\`) plus wildcard behavior.
- `TestGetOutputsForStack_MetacharacterInFilter` - 4 subtests verifying that `getOutputsForStack` correctly handles metacharacters in both stack and export filter patterns.

**What it verifies:** Regex metacharacters in glob patterns are treated as literal characters, and only `*` acts as a wildcard.

**Run command:** `go test ./lib -run 'TestGlobToRegex|TestGetOutputsForStack_Metacharacter'`

## Affected Files

| File | Change |
|------|--------|
| `lib/glob.go` | New file with `GlobToRegex` helper function |
| `lib/glob_test.go` | New file with regression tests |
| `lib/stacks.go` | Replaced inline regex construction with `GlobToRegex` |
| `lib/outputs.go` | Replaced inline regex construction with `GlobToRegex` |
| `lib/resources.go` | Replaced inline regex construction with `GlobToRegex` |
| `cmd/dependencies.go` | Replaced inline regex construction with `lib.GlobToRegex` |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed all 5 call sites now use `GlobToRegex` by searching for the old pattern.
- Verified no remaining instances of the naive `ReplaceAll` pattern exist in the codebase.

## Prevention

**Recommendations to avoid similar bugs:**
- Use a shared helper function for pattern conversions rather than inline regex construction.
- When building regex from user input, always escape the input first with `regexp.QuoteMeta`.
- Add tests with metacharacter-containing inputs when implementing glob/pattern matching features.

## Related

- Transit ticket: `T-443`
