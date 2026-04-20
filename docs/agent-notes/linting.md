# Linting

## Toolchain

- `.golangci.yml` uses the golangci-lint v2 schema (`version: "2"`). A v1
  binary cannot read it and will exit with "config v2 / binary v1".
- Install v2 explicitly (the plain `cmd/golangci-lint@latest` resolves to v1):
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`
- CI (`.github/workflows/push.yml`) uses `golangci/golangci-lint-action@v8`
  with `version: latest`, so CI always runs the current v2.

## Preflight

- `make lint` and `test/validate_tests.sh` both invoke
  `scripts/check-golangci-lint.sh` before running `golangci-lint run`. The
  script fails fast with an install command when the binary is missing or not
  v2.
- Regression coverage for the preflight lives in
  `test/lint/check_golangci_lint_version_test.sh`. It stubs `golangci-lint` on
  PATH with fake v1/v2/missing binaries and exercises the exit codes and
  messages.

## Gotchas

- `golangci-lint --version` formats differ between majors:
  - v1: `golangci-lint has version v1.64.8 built ...`
  - v2: `golangci-lint has version 2.1.6 built ...` (no leading `v`)
  The preflight parser tolerates both (optional leading `v`).
- When bumping the config schema, update `scripts/check-golangci-lint.sh`
  (`REQUIRED_MAJOR`) alongside.
