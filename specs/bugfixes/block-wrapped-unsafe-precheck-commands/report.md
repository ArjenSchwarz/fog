# Bugfix Report: Block Wrapped Unsafe Precheck Commands

**Date:** 2026-04-28
**Status:** Fixed
**Ticket:** T-1071

## Description of the Issue

`RunPrechecks` only deny-listed the first executable token returned by `splitShellArgs`, so wrapper executables such as `env` or `sh -c` could still invoke blocked commands like `rm`, `del`, or `kill`.

**Reproduction steps:**
1. Configure `templates.prechecks: ["env rm --help"]` or `templates.prechecks: ["sh -c 'rm -rf template.yaml'"]`
2. Run a deployment that triggers prechecks
3. The wrapped unsafe command executes instead of being rejected before `exec.Command`

**Impact:** High severity — a misconfigured or malicious precheck can still execute destructive commands by placing them behind a wrapper executable.

## Investigation Summary

- **Symptoms examined:** Direct `rm` commands were blocked, but wrapper forms such as `env rm --help`, `env sh -c 'rm -rf ...'`, and `sh -c 'rm -rf ...'` still ran.
- **Code inspected:** `lib/files.go` (`RunPrechecks` and `splitShellArgs`) and `lib/files_test.go`
- **Hypotheses tested:** Confirmed with failing regression tests that the deny-list only validates `separated[0]`, so it misses delegated commands embedded behind wrappers.

## Discovered Root Cause

The precheck safety guard inspects only the first parsed executable name. That works for direct invocations, but wrapper executables forward execution to a second command string or nested executable, which `RunPrechecks` never inspects.

**Defect type:** Incomplete validation of delegated execution

**Why it occurred:** The original validation assumed the executable token was always the command that would ultimately run. Wrapper semantics (`env`, `sh -c`, etc.) violate that assumption.

**Contributing factors:** Existing regression coverage only covered direct command names and path-prefixed variants, not wrapped execution paths.

## Resolution for the Issue

**Changes made:**
- `lib/files.go` — Added recursive wrapped-command inspection so `RunPrechecks` denies blocked commands when they are delegated through `env`, POSIX shells (`sh`, `bash`, `zsh`, `dash`, `ksh`, `ash`), `cmd /c`, and PowerShell command wrappers.
- `lib/files.go` — Tightened shell option parsing so only real `-c` flag combinations are treated as command wrappers, while option/value pairs such as `bash -o pipefail -c ...` and `bash -onoclobber -c ...` continue to work.
- `lib/files.go` — Normalized Windows-style executable suffixes (`.exe`, `.bat`, `.cmd`) and added PowerShell `-EncodedCommand` detection.
- `lib/files.go` — Rejects shell/PowerShell command strings that contain control operators and therefore cannot be safely reduced to a single argv vector during precheck validation.
- `lib/files_test.go` — Added regression coverage for `env`, `env -S`, nested wrappers, shell `-c`/`-lc`, `cmd /c`, PowerShell command/encoded-command wrappers, Windows executable suffixes, and safe wrapper cases.

**Approach rationale:** The fix keeps the current deny-list design but applies it to the command that will actually execute, not just the outer wrapper binary. Recursive unwrapping is still a small, local change, but it now errs on the side of safety when wrapper command strings become too shell-like to inspect reliably. The wrapper coverage is intentionally best-effort rather than exhaustive, so an allowlist remains the stronger long-term hardening option.

**Alternatives considered:**
- Allowlist precheck executables entirely — stronger but broader in scope than this bugfix

## Regression Test

**Test file:** `lib/files_test.go`
**Test name:** `TestRunPrechecksUnsafeWrappedCommand`

**What it verifies:** That unsafe commands are rejected even when invoked through wrapper executables such as `env`, `sh -c`, `bash -lc`, `cmd /c`, PowerShell command strings, or PowerShell encoded commands, while safe wrapped commands remain allowed.

**Run command:** `go test ./lib -run TestRunPrechecksUnsafeWrappedCommand -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/files.go` | Harden wrapped-command validation before executing prechecks |
| `lib/files_test.go` | Add regression coverage for wrapped unsafe commands |

## Verification

**Automated:**
- [x] Regression test passes (`go test ./lib -run TestRunPrechecksUnsafeWrappedCommand -v`)
- [x] Full test suite passes (`go test ./... -v`)
- [x] Linters/validators pass (`golangci-lint run`)

**Manual verification:**
- Not needed beyond automated coverage; the failing bypass cases are fully exercised by unit tests before command execution.

## Prevention

**Recommendations to avoid similar bugs:**
- Treat wrapper executables as delegated execution and inspect the nested command they will run
- Add regression tests whenever precheck validation logic changes
- Document wrapper-inspection limits whenever the deny-list expands so maintainers do not assume it is exhaustive
- Consider an allowlist for precheck executables as a future hardening step

## Related

- Transit ticket T-1071
