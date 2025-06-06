# Contribution Guide

This repository is a Go project that uses [Cobra](https://github.com/spf13/cobra) for command-line tooling.

## Project layout

- `cmd/` – Cobra command implementations.
- `lib/` – reusable library code with accompanying unit tests.
- `config/` – configuration helpers.
- `docs` - documentation for the project
- `docs/research` - documentation for dependencies like Cobra and go-outputs.
    - As an example, the library `github.com/arjenschwarz/go-outputs` is used for enforcing the structured outputs. Documentation on the go-outputs library and how to use it can be found in `docs/research/go-outputs.md`.

## Local validation

Before opening a pull request run the following commands:

1. `gofmt`
2. `go test ./... -v`
3. `golangci-lint run`
4. Optionally `go build -o fog` to confirm the project builds.

## Code Style

- Use `gofmt` for formatting
- Follow Go best practices and maintain existing code style
- Follow standard Go naming conventions
- Add comments for exported functions and types
- Keep functions focused and testable

## Test instructions

1. Add or update tests for any code you change, even if nobody asked.
2. Tests should be complete and cover both failure and success states.
3. Tests should NOT recreate functions from the files that are being tested. Instead, the original function can be updated to make it possible to provide mock objects.
4. Include clear documentation of what the tests cover

## Code changes

1. If there are existing comments that are intended to clarify the code, leave these intact or update them accordingly. Do not delete the comments unless the related code is deleted.
2. Add comments where they add value to understanding the code. Ensure these comments are clear aid in this understanding. The comments should explain why the code does what it does, not how it does it.
3. Prefer simplicity and easy to understand code over complex solutions.
4. Try to make changes as localised as possible, but functions that are likely useful in multiple places should go in a helper file in the same directory.
5. If the changes impact the way existing functionality works, this should be reflected in the README and if required in the docs folder.
6. Documentation is aimed at aiding understanding. Don't try to be too concise.

## New functionality

1. If new functionality is created, ensure that the README file is updated to include this.
2. Documentation is aimed at aiding understanding. Don't try to be too concise.

## Pull request requirements

- PR titles should clarify if they're bug fixes, features, refactors, or documentation based in the format `[bug|feature|refactor|doc] <Title>` and should reference the relevant issue when applicable.

## Configuration examples

See [`example-fog.yaml`](example-fog.yaml) for an annotated example configuration and [`fog.yaml`](fog.yaml) for an example used by tests.
