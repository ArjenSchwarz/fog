# Release Notes: Fog v1.12.2

**Release Date:** 2026-01-19

## Overview

This is a bug fix release that resolves an issue with the `--file-format` flag not being respected when outputting to files.

## Bug Fixes

### Fixed `--file-format` Flag Being Ignored

Previously, when using the `--file-format` flag with a different format than `--output`, the file output would incorrectly use the console format instead of the specified file format.

**Example of the issue:**
```bash
# Before: This would write table format to the file instead of markdown
fog drift --stackname my-stack --output table --file result.md --file-format markdown
```

**After this fix:** The file correctly contains markdown-formatted output while the console displays table format.

This fix ensures that users can now properly output different formats to files and console simultaneously, which is particularly useful for:
- Generating machine-readable files (JSON, YAML, CSV) while viewing human-readable tables in the console
- Creating documentation in markdown format while monitoring progress in table format
- CI/CD pipelines that need structured output files alongside console logs

## Installation

### Homebrew (macOS/Linux)

```bash
brew upgrade fog
```

### Go Install

```bash
go install github.com/ArjenSchwarz/fog@v1.12.2
```

### Binary Downloads

Download the appropriate binary for your platform from the [GitHub Releases](https://github.com/ArjenSchwarz/fog/releases/tag/v1.12.2) page.

## Full Changelog

See the [CHANGELOG.md](../../CHANGELOG.md) for complete details.
