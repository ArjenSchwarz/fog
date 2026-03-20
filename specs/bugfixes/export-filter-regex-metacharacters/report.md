# Bugfix Report: Export Filter Treats Regex Metacharacters as Wildcards

**Date:** 2026-03-20
**Status:** Fixed

## Description of the Issue

The `getOutputsForStack` function in `lib/outputs.go` builds regex patterns from user-provided export and stack filters by replacing `*` with `.*`, but does not escape other regex metacharacters. This causes characters like `.`, `+`, `?`, `(`, `)` etc. to be interpreted as regex operators instead of literal characters.

**Reproduction steps:**
1. Have two exports: `my.export.name` and `myXexportXname`.
2. Filter exports with the exact name `my.export.name`.
3. Both exports match because `.` in the regex matches any character.

**Impact:** Users filtering exports by exact name get unexpected matches when the name contains regex metacharacters (most commonly `.`). This affects the `exports` command output.

## Investigation Summary

Inspected the filter-to-regex conversion logic in `getOutputsForStack`.

- **Symptoms examined:** Export filter `foo.bar` matches `fooXbar` because `.` acts as regex any-char
- **Code inspected:** `lib/outputs.go` (lines 82-98), `lib/outputs_test.go`
- **Hypotheses tested:** Whether the issue is in the regex construction (confirmed) or in the matching logic (ruled out)

## Discovered Root Cause

Lines 84-85 of `lib/outputs.go` build regex patterns by only replacing `*` with `.*`, without escaping other regex metacharacters in the user-provided filter string:

```go
stackRegex := "^" + strings.ReplaceAll(stackfilter, "*", ".*") + "$"
exportRegex := "^" + strings.ReplaceAll(exportfilter, "*", ".*") + "$"
```

**Defect type:** Missing input sanitization

**Why it occurred:** The code assumes the only special character users would use is `*` for wildcards, but AWS resource names commonly contain `.` and other characters that are also regex metacharacters.

**Contributing factors:** Go's `regexp.MatchString` interprets the full regex syntax, so any unescaped metacharacter changes the matching semantics.

## Resolution for the Issue

**Changes made:**
- `lib/outputs.go:84-85` - Use `regexp.QuoteMeta()` to escape all regex metacharacters before replacing the escaped wildcard literal (`\*`) with `.*` for wildcard support.

**Approach rationale:** `regexp.QuoteMeta` is the standard Go approach for escaping regex metacharacters. By escaping first and then converting the escaped wildcard back, we preserve wildcard functionality while treating all other characters literally.

**Alternatives considered:**
- Only use regex when filter contains `*`, otherwise use exact string comparison - would work but creates two code paths and is less maintainable.
- Manually escape known metacharacters (`.`, `+`, `?`, etc.) - fragile, easy to miss characters, and `regexp.QuoteMeta` already does this correctly.

## Regression Test

**Test file:** `lib/outputs_test.go`
**Test name:** `Test_getOutputsForStack_regexMetacharacters`

**What it verifies:** Filters containing regex metacharacters (`.`, `+`) are treated as literal characters, not regex operators. Also verifies that `*` wildcard still works correctly when combined with metacharacters.

**Run command:** `go test ./lib -run Test_getOutputsForStack_regexMetacharacters -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/outputs.go` | Escape regex metacharacters in stack and export filters |
| `lib/outputs_test.go` | Regression tests for metacharacter handling in filters |
| `specs/bugfixes/export-filter-regex-metacharacters/report.md` | Bug investigation and resolution report |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes
- [x] Linters/validators pass

**Manual verification:**
- Confirmed that `my.export.name` filter no longer matches `myXexportXname`.
- Confirmed that wildcard patterns like `my.export.*` still work correctly.

## Prevention

**Recommendations to avoid similar bugs:**
- Always use `regexp.QuoteMeta` when building regex patterns from user input.
- When supporting glob-style wildcards, escape first then replace the wildcard token.
- Test filter logic with inputs containing common metacharacters (`.`, `+`, `?`, `[`, `]`).

## Related

- Transit ticket: `T-511`
