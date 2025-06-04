# Contribution Guide

This repository is a Go project that uses [Cobra](https://github.com/spf13/cobra) for command-line tooling.

## Project layout

- `cmd/` – Cobra command implementations.
- `lib/` – reusable library code with accompanying unit tests.
- `config/` – configuration helpers.

## Local validation

Before opening a pull request run the following commands:

1. `go test ./... -v`
2. `golangci-lint run`
3. Optionally `go build -o fog` to confirm the project builds.
4. Add or update tests for any code you change, even if nobody asked.

## Pull request requirements

- PR titles should follow the format `[fog] <Title>` and should reference the relevant issue when applicable.
- `CHANGELOG.md` must be updated with a concise message about the changes made.

## Configuration examples

See [`example-fog.yaml`](example-fog.yaml) for an annotated example configuration and [`fog.yaml`](fog.yaml) for an example used by tests.
