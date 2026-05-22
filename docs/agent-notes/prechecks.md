# Prechecks System

## Overview

Prechecks are user-configured commands that run before deployment to validate templates. They are defined in `fog.yaml` under `templates.prechecks` as a list of command strings.

## Key Files

- `lib/files.go` - `RunPrechecks()` executes precheck commands, `splitShellArgs()` parses command strings
- `cmd/deploy_helpers.go` - `runPrechecks()` wraps `RunPrechecks` with UI output and logging

## How It Works

1. User configures precheck commands in `fog.yaml` with `$TEMPLATEPATH` placeholder
2. `RunPrechecks` substitutes `$TEMPLATEPATH` with the actual template path
3. `splitShellArgs` parses the command string into command + arguments (respects single/double quotes)
4. Each command is executed via `exec.Command`; failures are collected but don't stop execution unless `templates.stop-on-failed-prechecks` is true

## Important Details

- `splitShellArgs` handles single quotes, double quotes, and backslash-escaped quotes inside double-quoted strings
- Returns an error on unbalanced quotes or empty commands
- Unsafe commands (`rm`, `del`, `kill`) are blocked before execution
- Commands must be findable via `exec.LookPath`
- Wrapped prechecks (`env`, `sh -c`, `bash -c`, `cmd /c`, `powershell -Command`/`-EncodedCommand`) are recursively unwrapped so the delegated command is also deny-listed (T-1071)
- `parseWrappedCommandString` rejects a wrapper command string when its command-position token (`argv[0]`) is not a simple literal executable: it rejects control operators (`hasUnquotedCommandOperators`) and dynamic expansion in the command position (`commandPositionHasExpansion`: `$`/backtick for shell+PowerShell, `%` for cmd). This blocks shell-expansion bypasses like `sh -c 'r${EMPTY:-m} ...'` and `sh -c '$DANGEROUS ...'` (T-1157). Expansion in argument position (e.g. `cfn-lint "$TEMPLATEPATH"`) is still allowed — only argv[0] is checked
- `splitShellArgs` preserves `$` even inside double quotes, mirroring real-shell expansion, which is why the command-position check operates on the post-split token
- Test functions that use viper to set prechecks should call `t.Cleanup(viper.Reset)` to avoid state leaking between tests
