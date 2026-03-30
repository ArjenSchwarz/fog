# Bugfix Report: Block Unsafe Precheck Commands When Invoked via Path

**Date:** 2026-03-29
**Status:** Fixed
**Ticket:** T-611

## Description of the Issue

`RunPrechecks` only blocked exact command names (`rm`, `del`, `kill`) before calling `exec.LookPath`, so absolute or relative paths bypassed the safety guard entirely.

**Reproduction steps:**
1. Configure `templates.prechecks: ["/bin/rm -rf /tmp/something"]` in fog.yaml
2. Run a deployment that triggers prechecks
3. The `/bin/rm` command is executed instead of being rejected as unsafe

**Impact:** High severity — a misconfigured or malicious precheck command with a path prefix could execute destructive commands that the denylist was designed to prevent.

## Investigation Summary

- **Symptoms examined:** The denylist check compared the raw command string (including any path prefix) against bare command names
- **Code inspected:** `lib/files.go` — `RunPrechecks` function, specifically the `stringInSlice` check at the original line 150
- **Hypotheses tested:** Confirmed that `filepath.Base` normalisation before the denylist check resolves the bypass

## Discovered Root Cause

The `command` variable extracted from `splitShellArgs` preserved the full path when the user specified one (e.g. `/bin/rm`). The denylist check compared this full path against bare names like `"rm"`, so the match always failed for path-prefixed commands.

**Defect type:** Missing input normalisation

**Why it occurred:** The original implementation only anticipated bare command names in the precheck configuration. The TODO comment on the line acknowledged the check needed improvement.

**Contributing factors:** No test coverage existed for path-prefixed command variants.

## Resolution for the Issue

**Changes made:**
- `lib/files.go:149-152` — Extract the base name of the command using `filepath.Base()` before comparing against the denylist. The error message still reports the original command string for clarity.

**Approach rationale:** `filepath.Base` is the minimal, correct way to normalise any path-prefixed executable name down to its base name. It handles absolute paths (`/bin/rm`), relative paths (`./rm`), and directory-prefixed paths (`../bin/rm`) uniformly.

**Alternatives considered:**
- Checking both the raw command and the resolved `exec.LookPath` result — more complex with no benefit over `filepath.Base`
- Using an allowlist instead of a denylist — larger scope change, better handled as a separate feature

## Regression Test

**Test file:** `lib/files_test.go`
**Test name:** `TestRunPrechecksUnsafeCommandWithPath`

**What it verifies:** That unsafe commands are blocked regardless of how the executable is referenced — absolute paths (`/bin/rm`), relative paths (`./rm`, `../bin/rm`), and bare names (`rm`). Also verifies the error message explicitly mentions "unsafe command".

**Run command:** `go test ./lib/ -run TestRunPrechecksUnsafeCommandWithPath -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/files.go` | Normalise command name with `filepath.Base` before denylist check |
| `lib/files_test.go` | Add `TestRunPrechecksUnsafeCommandWithPath` regression test |

## Verification

**Automated:**
- [x] Regression test passes
- [x] Full test suite passes (`go test ./...`)
- [x] Linter passes (`golangci-lint run`)

## Prevention

**Recommendations to avoid similar bugs:**
- When validating user-supplied command strings, always normalise paths before checking against a denylist
- Consider adding an allowlist approach for precheck commands as a future enhancement
- Add test cases for path-variant inputs whenever a command name is validated

## Related

- Transit ticket T-611
