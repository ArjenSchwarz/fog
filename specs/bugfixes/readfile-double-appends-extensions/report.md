# Bugfix Report: ReadFile Double-Appends Extensions

**Date:** 2026-03-20
**Status:** In Progress

## Description of the Issue

When a user passes a template filename that already includes an extension (e.g., `my-template.yaml`) and the file lives in the configured templates directory (not CWD), `ReadFile` constructs candidate paths like `templates/my-template.yaml.yaml` instead of first trying `templates/my-template.yaml`. The file is never found, and the command fails.

**Reproduction steps:**
1. Configure a templates directory (e.g., `templates/`) with extensions `[.yaml, .yml, .json]`
2. Place a template at `templates/my-template.yaml`
3. Run `fog deploy --template my-template.yaml` from the project root (where `my-template.yaml` does not exist in CWD)
4. Observe: ReadFile fails with "no file found" because it only tried `templates/my-template.yaml.yaml`, `.yml`, `.json`

**Impact:** Any user who passes a filename with an extension to a fog command and relies on the configured directory lookup will get a "file not found" error, even though the file exists.

## Investigation Summary

The bug is in `lib/files.go`, function `ReadFile` (lines 23-46).

- **Symptoms examined:** ReadFile fails when given a filename with extension that exists in configured directory
- **Code inspected:** `lib/files.go` (ReadFile function), `lib/files_test.go`, `example-fog.yaml` (for configured extensions)
- **Hypotheses tested:** The extension-appending loop is the sole file-discovery mechanism after the initial `os.Stat` check fails

## Discovered Root Cause

When the initial `os.Stat(filePath)` check fails (file not found at the literal path), ReadFile enters a loop that only tries `<directory>/<name><extension>`. It never tries `<directory>/<name>` without appending an extension. If the name already contains an extension (e.g., `testfile.yaml`), the constructed paths become `dir/testfile.yaml.yaml`, `dir/testfile.yaml.yml`, etc.

**Defect type:** Missing code path (incomplete file resolution logic)

**Why it occurred:** The original implementation assumed users would always pass base names without extensions, relying on the extension list to find the actual file. The case where a user passes a complete filename (with extension) was not handled.

**Contributing factors:** No test existed for this scenario.

## Resolution for the Issue

_To be filled after fix is implemented._

## Regression Test

**Test file:** `lib/files_test.go`
**Test name:** `TestReadFile/File_name_with_extension_in_configured_directory`

**What it verifies:** That ReadFile finds a file in the configured directory when the filename already includes its extension.

**Run command:** `go test ./lib -run "TestReadFile/File_name_with_extension_in_configured_directory" -v`

## Affected Files

| File | Change |
|------|--------|
| `lib/files.go` | Fix ReadFile to try bare name in directory before appending extensions |
| `lib/files_test.go` | Add regression test for filename-with-extension case |

## Verification

**Automated:**
- [ ] Regression test passes
- [ ] Full test suite passes
- [ ] Linters/validators pass

## Prevention

**Recommendations to avoid similar bugs:**
- Always test file-lookup functions with filenames that already include extensions
- Consider using `filepath.Ext()` to detect existing extensions before appending more

## Related

- Transit ticket: T-529
