# Fog 1.12.1 Release Notes

**Release Date:** 2026-01-19

This is a patch release containing bug fixes and documentation improvements.

---

## 🐛 Bug Fixes

### Drift Detection Tag Handling

Fixed an issue where the `ignore-tags` configuration was only applied to modified tags during drift detection. Now the configuration correctly ignores tags for all tag difference types:

- **ADD** - Tags added to resources
- **REMOVE** - Tags removed from resources
- **MODIFY** - Tags changed on resources

This ensures consistent tag filtering behavior across all drift detection scenarios.

**Configuration example:**
```yaml
drift:
  ignore-tags:
    - AWS::EC2::TransitGatewayAttachment:Application
    - aws:cloudformation:stack-name
```

### Dependency Classification

Fixed `github.com/mattn/go-isatty` dependency to be classified as direct rather than indirect, which was flagged by `go mod tidy`.

---

## 📚 Documentation

### Output Format Documentation

Updated documentation to clarify that the `--output` flag supports all available formats:

- `table` (default)
- `json`
- `yaml`
- `csv`
- `markdown`
- `html`

The documentation updates affect:
- Global flag help text in `cmd/root.go`
- Example configuration file (`example-fog.yaml`)
- README.md

---

## 🔧 Technical Details

### Dependencies

- `go-isatty`: v0.0.20 (now correctly marked as direct dependency)

---

## 📋 Upgrade Instructions

This is a straightforward patch release with no breaking changes. To upgrade:

```bash
# Using go install
go install github.com/ArjenSchwarz/fog@v1.12.1

# Or download from GitHub releases
```

---

## 📖 Resources

- [CHANGELOG](../../CHANGELOG.md)
- [User Guide](../user-guide/README.md)
- [Configuration Reference](../user-guide/configuration-reference.md)
