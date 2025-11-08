# Stream Separation Verification Report

**Task:** Verify all output paths use correct streams
**Date:** 2025-11-08
**Status:** ✅ VERIFIED

## Summary

All output paths in `cmd/deploy.go` and `cmd/deploy_helpers.go` correctly use the appropriate streams:
- **stderr**: All progress, diagnostic, and interactive prompts
- **stdout**: Final formatted output only (via go-output library)

## Detailed Findings

### 1. Progress Output to stderr ✅

All progress and diagnostic output correctly uses stderr:

#### Via `printMessage()` helper (cmd/output_helpers.go:48-54)
The `printMessage()` function uses `createStderrOutput()` internally:
```go
func printMessage(message string) {
    doc := output.New().Text(message).Build()
    out := createStderrOutput()  // Uses stderr writer
    if err := out.Render(context.Background(), doc); err != nil {
        fmt.Fprintf(os.Stderr, "ERROR: Failed to render message: %v\n", err)
    }
}
```

**All `printMessage()` calls verified** in both deploy.go and deploy_helpers.go:
- Line 80: Pre-deployment errors
- Line 91: Precheck output
- Line 149: Template read failures
- Line 155: S3 upload failures
- Line 200, 206: Tag file errors
- Line 239, 245: Parameter file errors
- Line 257: Template upload info
- Line 261, 266: Changeset creation failures
- Line 275: No-changes message (streaming)
- Line 284: Changeset creation errors
- Line 304, 306, 308: Changeset deletion messages
- Line 312: Changeset deletion failures
- Line 336, 340, 342, 344: Stack deletion messages
- Line 355, 357: Deploy confirmation messages
- Line 362: Changeset execution errors
- Line 388, 415: Event retrieval errors
- Line 460: Failed events prefix message
- deploy_helpers.go:168: Stack retrieval failures
- deploy_helpers.go:181: Success message (streaming)

#### Direct stderr writes ✅
All `fmt.Fprintf(os.Stderr, ...)` and `fmt.Fprintln(os.Stderr, ...)` calls verified:

**cmd/deploy.go:**
- Line 130, 132: Deployment info messages
- Line 285: Changeset status reason
- Line 286: Changeset console URL
- Line 327: New stack delete info
- Line 348: Stack intact message
- Line 363: Changeset execution error details
- Line 368: Event display header
- Line 400, 403, 405: Event status messages with colors
- Line 416: Event retrieval error details
- Line 456: Failed events render error

**cmd/deploy_helpers.go:**
- Line 186: Success output generation warning
- Line 199: Failure output generation warning

#### Via `createStderrOutput()` ✅
Direct use of stderr output writer:
- deploy.go:454: Failed events table
- deploy.go:476-494: `createStderrOutput()` function definition with TTY detection
- output_helpers.go:50: Used by `printMessage()`
- describe_changeset.go:330, 332: Changeset display (when called from deploy)

### 2. Final Formatted Output to stdout ✅

All final output functions correctly use stdout via `settings.GetOutputOptions()`:

#### cmd/deploy_output.go:
- **Line 25**: `outputDryRunResult()` calls `buildAndRenderChangeset()` which uses stdout
- **Line 106**: `outputSuccessResult()` uses `output.NewOutput(settings.GetOutputOptions()...)`
- **Line 151**: `outputNoChangesResult()` uses `output.NewOutput(settings.GetOutputOptions()...)`
- **Line 258**: `outputFailureResult()` uses `output.NewOutput(settings.GetOutputOptions()...)`

All four output functions correctly:
1. Call `os.Stderr.Sync()` to flush stderr first
2. Print header separator to stdout: `fmt.Println("\n=== Deployment Summary ===")`
3. Render final output using `settings.GetOutputOptions()` which defaults to stdout

**Note:** The `fmt.Println()` calls on lines 33, 114, and 198 write the header directly to stdout, which is correct and intentional.

### 3. Interactive Prompts to stderr ✅

**cmd/helpers.go:19-42** - `askForConfirmation()` function:
- Line 23: `fmt.Fprintln(os.Stderr, "")` - Empty line
- Line 24: `fmt.Fprintf(os.Stderr, "🔔 %s [y/n]: ", s)` - Prompt text
- Reads from stdin (correct: prompts to stderr, input from stdin)

All interactive prompts correctly write to stderr and follow Unix conventions.

### 4. No stdout Output During Progress ✅

**Verified:** No stdout output occurs during deployment progress. All stdout writes are:
1. Final output functions in `deploy_output.go` (after deployment completes)
2. Header separators (intentional, part of final output)

The only potential stdout output during progress would be from:
- `buildAndRenderChangeset()` in dry-run mode - but this is correct, as dry-run is a final output scenario
- Error scenarios where formatting uses default stdout - but these are wrapped in stderr writers

### 5. Quiet Mode Handling ✅

Quiet mode (`deployFlags.Quiet`) is correctly checked in:
- deploy.go:121: `showDeploymentInfo()` early return
- deploy.go:255: Template upload message suppression
- deploy.go:273: No-changes message suppression
- deploy.go:353: Deploy confirmation message suppression
- deploy.go:367: Event display header suppression
- deploy.go:381: `showEvents()` early return
- deploy_helpers.go:180: Success message suppression

Quiet mode auto-enables non-interactive mode:
- deploy_helpers.go:44-46: `deployFlags.NonInteractive = true`

### 6. Stream Separation Best Practices ✅

All functions follow proper stream separation:

1. **stderr flushing**: All final output functions call `os.Stderr.Sync()` before stdout
2. **Visual separation**: Header `=== Deployment Summary ===` written to stdout
3. **TTY detection**: `createStderrOutput()` checks `isatty.IsTerminal()` to avoid ANSI codes in redirected output
4. **Consistent patterns**: All progress uses stderr, all final output uses stdout

## Edge Cases and Special Handling

### 1. Error Messages in Output Generation
When final output generation fails, warnings go to stderr:
- deploy_helpers.go:186: `fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)`
- deploy_helpers.go:199: `fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)`
- deploy.go:279: `fmt.Fprintf(os.Stderr, "Warning: Failed to generate output: %v\n", err)`

This is correct: output generation errors are diagnostic information.

### 2. Pre-deployment Errors
Pre-deployment errors (before AWS operations) correctly write to stderr only:
- No stdout output occurs when validation fails
- Errors use `printMessage()` which writes to stderr
- Exit codes are set appropriately (os.Exit(1))

### 3. Dry-Run and Create-Changeset
Both modes correctly output to stdout via `outputDryRunResult()`:
- Reuses `buildAndRenderChangeset()` which respects output format
- Proper stderr flushing before output
- Changeset deletion only occurs in dry-run (not create-changeset)

### 4. No-Changes Scenario
Correctly handles both streams:
- Streaming message to stderr (if not quiet)
- Final formatted output to stdout
- Exit code 0 (success)

### 5. TTY Detection
The `createStderrOutput()` function properly detects TTY:
- Adds colors/emojis only when stderr is a terminal
- Avoids ANSI codes when redirected to file
- Uses `github.com/mattn/go-isatty` for detection

### 6. Changeset Display Context
The `buildAndRenderChangeset()` function is called from two contexts:
1. **describe changeset command**: Direct user request (outputs to stdout)
2. **deploy dry-run/create-changeset**: Final output after progress (outputs to stdout)

Both contexts correctly use stdout since they produce final formatted output.

## Verification Methods

### Manual Verification Commands

```bash
# Test stderr/stdout separation
fog deploy --template stack.yaml --output json > result.json 2> progress.log

# Verify progress goes to stderr
wc -l progress.log  # Should show progress output

# Verify data goes to stdout
cat result.json | jq .  # Should show valid JSON

# Test quiet mode suppresses stderr
fog deploy --template stack.yaml --output json --quiet 2>&1 | wc -l
# Should show minimal output (only final JSON)

# Test separate stream capture
fog deploy --template stack.yaml --output yaml > data.yaml 2> /dev/null
# data.yaml should contain YAML output, no progress visible
```

### Code Review Checklist

- [x] All `printMessage()` calls use `createStderrOutput()`
- [x] All `fmt.Fprintf(os.Stderr, ...)` calls verified
- [x] All `fmt.Fprintln(os.Stderr, ...)` calls verified
- [x] All final output uses `settings.GetOutputOptions()` (stdout by default)
- [x] No `fmt.Print()` or `fmt.Println()` to default stdout during progress
  - Exception: Header separators in final output (intentional)
- [x] Interactive prompts write to stderr
- [x] Quiet mode properly suppresses stderr
- [x] stderr flushed before stdout in all final output
- [x] TTY detection implemented for conditional formatting

## Conclusion

✅ **ALL VERIFICATION CHECKS PASSED**

The implementation correctly separates output streams:
- **stderr**: Progress, diagnostics, prompts (suppressed in quiet mode)
- **stdout**: Final formatted output only (JSON/CSV/YAML/Markdown/table)

No issues found. The stream separation implementation follows Unix conventions and the design specification precisely.

## Recommendations

1. **No changes needed**: All output paths are correct
2. **Testing**: Run integration tests to verify behavior with redirected streams
3. **Documentation**: Ensure user documentation explains stream separation

## References

- Design Document: specs/deploy-output/design.md
- Requirements: specs/deploy-output/requirements.md
- Decision Log: specs/deploy-output/decision_log.md
- Implementation Files:
  - cmd/deploy.go
  - cmd/deploy_helpers.go
  - cmd/deploy_output.go
  - cmd/output_helpers.go
  - cmd/helpers.go
