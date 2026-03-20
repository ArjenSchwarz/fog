# Exports / Outputs / Stack Filtering

## Key Files

- `lib/outputs.go` — Export retrieval, filtering, and import population
- `lib/outputs_test.go` — Tests with mock CFN clients (including pagination)
- `lib/stacks.go` — Stack listing with wildcard filtering (line ~233), also calls `getOutputsForStack`
- `lib/resources.go` — Resource listing with wildcard filtering (line ~49)

## How Export Filtering Works

`getOutputsForStack` accepts stack and export filter strings that support `*` as a glob-style wildcard. The function converts these into regex patterns:

1. `regexp.QuoteMeta()` escapes all regex metacharacters in the filter
2. The escaped wildcard (`\*`) is replaced with `.*` for regex matching
3. The pattern is anchored with `^...$`

The stack filter only triggers regex matching when it contains `*`. The export filter always uses regex matching when non-empty (even for exact matches — the anchored pattern handles this).

## Gotchas

- `GetExports` sends the stack name directly to the AWS API when it does not contain `*`, so the API does the exact filtering. When it does contain `*`, all stacks are fetched and filtered locally via regex.
- The `getOutputsForStack` function is also called from `stacks.go` with empty filters (no filtering needed there).
- Mock clients in tests: `MockCFNClient` for single-page, `paginatingMockCFNClient` for multi-page responses.
- The same wildcard-to-regex pattern exists in three places: `lib/outputs.go`, `lib/stacks.go`, and `lib/resources.go`. All three must use `regexp.QuoteMeta()` before converting `*` to `.*`. This was the root cause of T-511.
