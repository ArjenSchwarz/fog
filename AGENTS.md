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

## Test instructions

1. Add or update tests for any code you change, even if nobody asked.
2. Tests should be complete and cover both failure and success states.
3. Tests should NOT recreate functions from the files that are being tested. Instead, the original function can be updated to make it possible to provide mock objects.
4. Include clear documentation of what the tests cover

## New functionality

1. If new functionality is created, ensure that the README file is updated to include this.

## Pull request requirements

- PR titles should follow the format `[fog] <Title>` and should reference the relevant issue when applicable.
- `CHANGELOG.md` must be updated with a concise message about the changes made, this should be added to the bottom of the list under the header Unreleased. If this header doesn't exist, add it to the top of the file.

## Configuration examples

See [`example-fog.yaml`](example-fog.yaml) for an annotated example configuration and [`fog.yaml`](fog.yaml) for an example used by tests.
