# Release Notes: Fog v1.12.4

**Release Date:** 2026-03-06

## Overview

Patch release fixing two drift detection bugs and reducing release binary size.

## Bug Fixes

### Transit Gateway Drift Detection

- **Route table ID resolution** — `FilterTGWRoutesByLogicalId` no longer panics when `TransitGatewayRouteTableId` is a `Ref` or `Fn::ImportValue` map instead of a preprocessed string. All property formats are now handled correctly.

### Drift Special-Cases Pagination

- **`ListExports` pagination** — The `ListExports` call in `separateSpecialCases` now paginates through all results. Previously, only the first page (up to 100 exports) was processed, silently dropping the rest.

## Build Improvements

- Release binaries are now built with `-s -w` ldflags, stripping symbol table and DWARF debug information for smaller downloads.

## Installation

### Go Install

```bash
go install github.com/ArjenSchwarz/fog@1.12.4
```

### Binary Downloads

Download the appropriate binary for your platform from the [GitHub Releases](https://github.com/ArjenSchwarz/fog/releases/tag/1.12.4) page.

## Full Changelog

See the [CHANGELOG.md](../../CHANGELOG.md) for complete details.
