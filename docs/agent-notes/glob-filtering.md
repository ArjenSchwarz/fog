# Glob Filtering

## How it works

Several commands support glob-style filtering where `*` matches any sequence of characters in stack names, export names, etc. The conversion from glob pattern to regex is handled by `lib.GlobToRegex()` in `lib/glob.go`.

`GlobToRegex` uses `regexp.QuoteMeta` to escape all regex metacharacters, then replaces the escaped wildcard (`\*`) with `.*`, and anchors with `^...$`.

## Call sites

The glob filtering pattern is used in these locations:
- `lib/stacks.go` — `GetCfnStacks` filters stacks by name
- `lib/outputs.go` — `getOutputsForStack` filters by stack name and export name
- `lib/resources.go` — `GetResources` filters stacks by name
- `cmd/dependencies.go` — `getFilteredStacks` filters dependency graph by stack name

All sites follow the same pattern: check if the filter contains `*`, and if so use `GlobToRegex` to match. If no wildcard is present, the filter is passed directly to the AWS API via `DescribeStacksInput.StackName` for server-side filtering.

## Gotchas

- The glob `*` matches everything including empty strings (like `.*` in regex). It does not behave like shell globbing where `*` doesn't match `/`.
- When adding new glob filtering, always use `lib.GlobToRegex()` rather than building regex inline to avoid metacharacter escaping bugs (see T-443).
