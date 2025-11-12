# Fog 1.12.0 Release Notes

## 🚨 Breaking Changes

### Stream Separation for Deploy Command

Fog now follows Unix conventions by separating progress output from structured data:

- **Progress output** (stack info, changeset details, deployment status, prompts) → **stderr**
- **Structured results** (deployment summary in JSON/YAML/CSV/etc.) → **stdout**

**Migration Guide:**

```bash
# Old - combined output to stdout
fog deploy --stack mystack | grep "Status"

# New - Option 1: Combine streams
fog deploy --stack mystack 2>&1 | grep "Status"

# New - Option 2: Parse structured output (recommended)
fog deploy --stack mystack --output json | jq '.status'

# New - Option 3: Suppress progress with --quiet
fog deploy --stack mystack --quiet --output json | jq '.status'
```

**Impact:**
- Scripts using `fog deploy ... | grep` or `fog deploy ... > file` will only capture final results
- CI/CD pipelines parsing combined output need updates

See the [deployment output specification](specs/deploy-output/design.md) for complete details.

---

## ✨ New Features

### Multi-Format Output Support

Both `fog deploy` and `fog describe changeset` now support multiple output formats:

- **JSON** - Machine-readable structured data
- **YAML** - Human-readable structured data
- **CSV** - Spreadsheet-compatible format
- **Markdown** - Documentation-friendly tables
- **Table** - Default interactive format
- **HTML** - Web-ready output (changeset only)

```bash
# Deploy with JSON output
fog deploy --stack myapp --output json

# Changeset as Markdown
fog describe changeset --changeset-name my-change --output markdown
```

### Quiet Mode

New `--quiet` flag suppresses all progress output (stderr) while showing only the final structured result:

```bash
# Perfect for CI/CD pipelines
fog deploy --stack myapp --quiet --output json > result.json
```

### Golden File Testing

Added comprehensive golden file test infrastructure for deployment output validation, ensuring consistent output across formats and scenarios.

---

## 🔄 Changes

### Go-Output v2 Migration

Upgraded from go-output v1.4.0 to v2.6.0, bringing:
- Thread-safe format functions
- Better parallel test support
- Improved styling and formatting APIs
- Modern Go patterns throughout

All commands migrated to the v2 Builder pattern:
- resources, deploy, drift, report
- describe changeset, demo tables, history

### Code Modernization

- Replaced `interface{}` with `any` (Go 1.18+)
- Removed manual loop variable captures (Go 1.22+ handles automatically)
- Simplified inline styling by using `output.Style*()` functions directly
- Improved test validation focusing on data correctness

### Behavior Changes

- Deploy command no longer enforces table output format
- Describe changeset command respects global `--output` flag
- Test validation focuses on content correctness rather than exact byte matching

---

## 🐛 Bug Fixes

### Critical Fixes

- **Nil pointer safety**: Added checks for `FinalStackState`, `event.Timestamp`, and other potential nil dereferences
- **JSON/YAML parsing**: Removed hardcoded headers from deploy output that were breaking structured output parsing
- **Duration calculation**: Fixed zero-time validation to prevent incorrect time calculations

### Output Fixes

- Fixed missing emoji and color formatting in deployment status messages (ℹ️, ✅, 🚨)
- ANSI color codes no longer appear in JSON/CSV/YAML output
- Fixed file output, HTML format rendering, and report frontmatter issues

### Dependency Updates

- Replaced deprecated `github.com/mitchellh/go-homedir` with standard library `os.UserHomeDir()`

### Test Improvements

- Fixed race conditions in parallel test execution that caused `fatal error: concurrent map writes`
- Improved test reliability and removed flaky tests

---

## 🏗️ Refactoring

Major code refactoring for improved maintainability:

- **`lib/stacks.go:GetEvents`**: Decomposed ~120 lines into 15 focused helper functions
- **`lib/template.go:NaclResourceToNaclEntry`**: Broke down ~110 lines into 8 specialized helpers
- **`cmd/deploy.go`**: Extracted 11 helper functions from deployment workflow

All refactored functions:
- Now under 50 lines each
- Reduced cyclomatic complexity
- Enhanced defensive programming (nil checks, type assertion safety, error visibility)

---

## 📚 Documentation

### New User Guides

Comprehensive documentation added in `docs/user-guide/`:
- **User Guide** - Installation, quick start, features, best practices
- **Configuration Reference** - All config options with examples
- **Deployment Files Spec** - Field reference and best practices
- **Advanced Usage** - Multi-stack, cross-stack refs, multi-region, CI/CD integration
- **Troubleshooting** - Solutions for common problems

### Architecture Diagrams

- Architecture overview visualization
- Configuration flow and precedence
- Deployment workflow diagram

### Testing Documentation

- Stream separation verification report
- Golden file test structure and usage
- Test coverage documentation

---

## 🧪 Testing Improvements

### Test Infrastructure

- Stream separation test suite with comprehensive stderr/stdout validation
- Golden file test infrastructure with `UPDATE_GOLDEN=1` support
- Unit and integration tests for all deployment output scenarios
- Test utilities package (`lib/testutil`) with assertion helpers and mock clients

### Test Coverage

Added extensive test coverage for:
- Deploy output builder functions (success, failure, no-changes)
- Stream separation across all output modes
- Multi-format output validation
- Quiet mode behavior
- Error handling scenarios

---

## 🔧 Technical Details

### Dependencies

- go-output: v1.4.0 → v2.6.0
- go-isatty: v0.0.20 (new)
- Removed: github.com/mitchellh/go-homedir

### Available Updates (Non-Critical)

Several AWS SDK and other dependencies have minor updates available. These are non-critical and don't impact functionality. Run `go list -m -u all` to see the full list.

---

## 📋 Upgrade Instructions

1. **Review Breaking Changes**: Understand the stream separation change if you have scripts that parse `fog deploy` output
2. **Update Scripts**: Update any CI/CD pipelines or scripts as shown in the migration guide above
3. **Test**: Run your deployment workflows in a test environment first
4. **Take Advantage**: Use the new `--output` and `--quiet` flags for cleaner CI/CD integration

## 🙏 Contributors

This release represents a significant effort in improving code quality, test coverage, and user experience. Special thanks to all contributors who helped shape this release through issues, discussions, and code reviews.

---

## 📖 Resources

- [CHANGELOG](CHANGELOG.md)
- [User Guide](docs/user-guide/README.md)
- [Deployment Output Specification](specs/deploy-output/design.md)
- [Configuration Reference](docs/user-guide/configuration-reference.md)
- [Troubleshooting Guide](docs/user-guide/troubleshooting.md)
