# Bugfix Report: golangci-lint-v2-config

**Date:** 2026-04-20
**Status:** Fixed
**Ticket:** T-846

## Description of the Issue

`make lint` runs `golangci-lint run` without validating the installed binary.
The repository's `.golangci.yml` declares `version: "2"`, which is only
understood by golangci-lint v2. When a developer has v1 installed (for example
v1.64.8, the final v1 release), the linter aborts immediately with:

```
Error: you are using a configuration file for golangci-lint v2 with golangci-lint v1: please use golangci-lint v2
make: *** [lint] Error 3
```

The error is clear in isolation, but the Makefile/README/AGENTS flow gives no
hint that v2 is required. Because `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
installs the v1 line (the v2 module lives at `.../v2/cmd/golangci-lint`), anyone
following the documented install command ends up with a broken local lint
target while CI still passes (the GitHub action pins `version: latest` which
resolves to v2).

**Reproduction steps:**

1. Install the documented v1 path: `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`
2. From the repo root run `make lint`
3. Observe the failure: the v1 binary rejects the v2 config

**Impact:** Local validation is broken for any contributor who follows the
current install instructions, and the failure mode is ambiguous — developers
may think the project itself is misconfigured rather than their toolchain.

## Investigation Summary

- **Symptoms examined:** `make lint` exits with "you are using a configuration file for golangci-lint v2 with golangci-lint v1" when the local binary is v1.64.8.
- **Code inspected:** `Makefile` (`lint` target), `.golangci.yml` (version pin), `test/validate_tests.sh` (step 6 also invokes `golangci-lint run`), `.github/workflows/push.yml` (CI uses `version: latest` in `golangci/golangci-lint-action@v8`), `README.md`, `AGENTS.md`, `CLAUDE.md` (install instructions).
- **Hypotheses tested:**
  - "Downgrade the config to v1" — rejected, v2 is the supported line and carries forward-looking features.
  - "Add a `.tool-versions` pin" — possible, but still relies on the user having mise/asdf and wouldn't produce a clear error from `make lint` alone.
  - "Check the binary version inside `make lint`" — chosen, because it fails fast with a direct message wherever the Makefile target runs.

## Discovered Root Cause

`make lint` (and `test/validate_tests.sh`) assume the installed `golangci-lint`
matches the v2 schema of `.golangci.yml`, but nothing enforces that
assumption. The documented install command (`go install .../cmd/golangci-lint@latest`)
resolves to v1 because v2 moved to a separate module path (`.../v2/cmd/golangci-lint`).
Combined, these produce an ambiguous error from the linter binary itself rather
than a build-system-level diagnostic that points at the real cause (wrong major
version installed).

**Defect type:** Missing preflight check + stale install documentation.

**Why it occurred:** The `.golangci.yml` was upgraded to `version: "2"` after
golangci-lint released v2, but the Makefile target, helper script, and
developer-facing install instructions were not updated to match.

**Contributing factors:** CI uses `version: latest`, so breakage is invisible
in CI and only surfaces locally.

## Resolution for the Issue

**Changes made:**

- `scripts/check-golangci-lint.sh` — new preflight script. Looks up
  `golangci-lint` on PATH, parses `--version`, and fails with a clear message
  ("requires golangci-lint v2; found vX...; install with `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`") when the major version is not 2 or the binary is missing.
- `Makefile` (`lint` target) — invokes the preflight script before
  `golangci-lint run` so misconfigured environments fail with a targeted
  diagnostic instead of the raw linter error.
- `test/validate_tests.sh` — same preflight invocation in step 6.
- `README.md`, `CLAUDE.md`, `AGENTS.md` — update install command to the v2
  module path (`github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`)
  and note that v2 is required.
- `test/lint/check_golangci_lint_version_test.sh` — regression test that stubs
  `golangci-lint` with v1/v2/missing binaries and verifies the preflight's
  exit codes and messaging.

**Approach rationale:** A Makefile preflight centralises the version contract
in one place, produces a targeted error, and doesn't require contributors to
adopt a new tool manager. Updating the docs at the same time removes the
install command that silently lands on v1.

**Alternatives considered:**

- **Pin via `.tool-versions` / `mise.toml`** — rejected as the sole fix: users
  without mise/asdf would still hit the raw linter error. The preflight works
  regardless of tool manager; a `.tool-versions` file could be added later as
  an additive convenience.
- **Downgrade the config to v1** — rejected because v2 is the supported line
  and CI already runs v2.
- **Inline the check in the Makefile recipe** — rejected in favour of a
  dedicated script so `test/validate_tests.sh` can reuse the same logic and so
  the check is testable with stub PATHs.

## Regression Test

**Test file:** `test/lint/check_golangci_lint_version_test.sh`

**What it verifies:**

- v1 binary on PATH causes the preflight to exit non-zero and the message
  mentions the v2 requirement.
- v2 binary on PATH causes the preflight to succeed (exit 0).
- Missing binary causes the preflight to exit non-zero with an install hint.

**Run command:** `./test/lint/check_golangci_lint_version_test.sh`

## Affected Files

| File | Change |
|------|--------|
| `scripts/check-golangci-lint.sh` | New preflight script that enforces v2 and produces actionable error messages |
| `Makefile` | `lint` target now runs the preflight before `golangci-lint run` |
| `test/validate_tests.sh` | Step 6 now runs the preflight before invoking the linter |
| `README.md` | Updated install instructions to the v2 module path and documented the v2 requirement |
| `CLAUDE.md` | Same update for project guidance |
| `AGENTS.md` | Same update for contributor guidance |
| `test/lint/check_golangci_lint_version_test.sh` | New regression test for T-846 |

## Verification

**Automated:**

- [x] Regression test passes (`./test/lint/check_golangci_lint_version_test.sh`)
- [x] Full test suite passes (`go test ./...`)
- [x] `make lint` with a v2 binary succeeds; with v1 it fails with the new targeted message

**Manual verification:**

- Ran `make lint` with the local v1.64.8 binary: produced the new preflight
  diagnostic instead of the raw v2-config error.

## Prevention

**Recommendations to avoid similar bugs:**

- When bumping config schema versions for tools invoked from the Makefile, add
  or update the preflight checks alongside the config change.
- Keep install commands in docs aligned with the module path actually required
  — for golangci-lint specifically, v2 lives at `.../v2/cmd/golangci-lint`.
- Consider adding a `.tool-versions` file as a future improvement so mise/asdf
  users get the right binary automatically.

## Related

- T-846: `make lint` fails with unpinned golangci-lint v2 config
