# Release Notes: Fog v1.12.3

**Release Date:** 2026-03-05

## Overview

This is a stability-focused patch release that fixes multiple crash-causing bugs across several fog commands. Functions that previously panicked on unexpected input now return proper errors, and pagination issues that caused missing data in large accounts have been resolved.

## Bug Fixes

### Panic-to-Error Conversions

Several functions have been hardened to return errors instead of panicking:

- **`ParseDeploymentFile`** — no longer panics on empty or whitespace-only deployment file content
- **Drift detection helpers** — AWS API failures now produce clean CLI errors instead of crashes
- **`GetStack`** — handles empty results and ambiguous multi-stack matches gracefully
- **`GetStackAndChangesetFromURL`** — returns an error on invalid URL input instead of calling `log.Fatal`/`panic`
- **`ReadAllLogs`** — skips malformed JSON log lines and continues processing valid entries
- **Lambda report handler** — no longer panics when the `ReportTimezone` environment variable is empty or unset

### Pagination Fixes

- **`GetExports`** and **`GetResources`** now process all pages of `DescribeStacks` results. Previously, accounts with more than 100 stacks would have exports and resources silently omitted.

### Caching Fix

- **`StackExists`** now correctly caches `RawStack` on success instead of only on error, eliminating unnecessary duplicate AWS API calls.

### Nil Pointer Guards

- Fixed nil `PhysicalResourceId` dereferences in resource and drift mapping operations.

## Installation

### Go Install

```bash
go install github.com/ArjenSchwarz/fog@v1.12.3
```

### Binary Downloads

Download the appropriate binary for your platform from the [GitHub Releases](https://github.com/ArjenSchwarz/fog/releases/tag/v1.12.3) page.

## Full Changelog

See the [CHANGELOG.md](../../CHANGELOG.md) for complete details.
