# Go-Output v2 Migration Evaluation for Fog

**Date**: 2025-10-17 (Updated)
**Purpose**: Evaluate whether go-output v2 contains all features required to migrate fog from v1 to v2
**Evaluator**: AI Agent Analysis
**Recommendation**: ✅ **PROCEED WITH CONFIDENCE** - All critical gaps have been resolved in v2.2.1

---

## Executive Summary

**🎉 UPDATE: All identified gaps have been resolved in go-output v2.2.1!**

The go-output v2.2.1 library provides **complete feature parity** with v1 and includes all functionality that fog currently uses. The three previously identified gaps have been addressed:

1. ✅ **Inline color/styling methods** - Now available as `StyleWarning()`, `StylePositive()`, etc.
2. ✅ **Table column width configuration** - Now available via `TableWithMaxColumnWidth()`
3. ✅ **Format-aware separator methods** - Automatic array handling with format-appropriate separators

**Migration is now straightforward** with no workarounds needed. The core functionality—table output with key ordering, multiple formats, file output, and styling—is fully supported in v2 with significantly improved patterns (Builder, functional options, thread safety).

---

## Quick Reference: Key Changes

### What Changed in v2.2.1 (Fog-Specific)

| v1 Feature | v2.2.1 Replacement | Migration Complexity |
|------------|-------------------|---------------------|
| `StringWarningInline(text)` | `StyleWarning(text)` | ✅ Simple find-replace |
| `StringPositiveInline(text)` | `StylePositive(text)` | ✅ Simple find-replace |
| `StringNegativeInline(text)` | `StyleNegative(text)` | ✅ Simple find-replace |
| `settings.TableMaxColumnWidth = 50` | `TableWithMaxColumnWidth(50)` | ⚠️ Update config pattern |
| `strings.Join(arr, GetSeparator())` | Pass array directly OR keep GetSeparator() | 🔵 Optional optimization |
| `OutputArray{Keys: [...]}` | `Table("", data, WithKeys(...))` | ⚠️ Pattern change |
| `settings.OutputFile = "x"` | `WithWriter(NewFileWriter(...))` | ⚠️ Pattern change |

### Migration Effort Summary

- **Low effort** (60%): Inline styling replacements, format names
- **Medium effort** (35%): OutputArray → Builder, Settings → Options
- **High effort** (5%): Complex multi-table commands

### Key Benefits of v2.2.1

1. **No workarounds needed** - All fog features have direct replacements
2. **Better patterns** - Builder pattern, functional options, thread safety
3. **Enhanced features** - Data pipelines, collapsible content (future use)
4. **Reduced timeline** - 1 week migration (vs 2-3 weeks previously estimated)

---

## Current Fog Usage Analysis

### Usage Statistics
- **go-output import locations**: 26 files
- **OutputArray instances**: ~64 occurrences
- **OutputSettings usage**: Heavy (configuration hub)
- **Inline styling methods**: 10+ occurrences (drift detection)
- **GetSeparator calls**: 5 occurrences

### Core Features Used by Fog

#### 1. **Table Output with Column Ordering** ✅ SUPPORTED
```go
// v1 (current fog usage)
output := format.OutputArray{
    Keys: []string{"LogicalId", "Type", "ChangeType", "Details"},
    Settings: settings.NewOutputSettings(),
}

// v2 equivalent
doc := output.New().
    Table("", data, output.WithKeys("LogicalId", "Type", "ChangeType", "Details")).
    Build()
```
**Status**: Fully supported via Builder pattern and WithKeys()

#### 2. **Multiple Output Formats** ✅ SUPPORTED
Fog uses: table, csv, json, dot (Graphviz)

v2 provides all these formats:
- `output.Table` - Terminal table output
- `output.CSV` - CSV spreadsheet format
- `output.JSON` - Structured JSON
- `output.DOT` - Graphviz diagrams

**Status**: Fully supported, v2 adds many more formats (HTML, YAML, Markdown, Mermaid, DrawIO)

#### 3. **File Output** ✅ SUPPORTED
```go
// v1 (current fog usage)
settings.OutputFile = "report.csv"
settings.OutputFileFormat = "csv"

// v2 equivalent
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithFormat(output.CSV),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithWriter(output.NewFileWriter(".", "report.csv")),
)
```
**Status**: Fully supported with improved multi-destination support

#### 4. **Table Styling** ✅ SUPPORTED
```go
// v1 (current fog usage)
settings.TableStyle = format.TableStyles["Default"]

// v2 equivalent
out := output.NewOutput(
    output.WithFormat(output.TableWithStyle("Default")),
)
```
**Status**: Fully supported. v2 provides: Default, Bold, ColoredBright, Light, Rounded

#### 5. **Color and Emoji Support** ✅ SUPPORTED
```go
// v1 (current fog usage)
settings.UseEmoji = true
settings.UseColors = true

// v2 equivalent
out := output.NewOutput(
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.ColorTransformer{}),
)
```
**Status**: Fully supported via Transformer system

#### 6. **Sorting** ✅ SUPPORTED
```go
// v1 (current fog usage)
settings.SortKey = "LogicalId"

// v2 equivalent (Option A: Byte transformer)
out := output.NewOutput(
    output.WithTransformer(&output.SortTransformer{Key: "LogicalId", Ascending: true}),
)

// v2 equivalent (Option B: Data pipeline - BETTER)
doc := output.New().Table("", data, output.WithKeys(...)).Build()
transformedDoc := doc.Pipeline().
    SortBy("LogicalId", output.Ascending).
    Execute()
```
**Status**: Fully supported with two approaches (transformers or data pipeline)

#### 7. **Multiple Tables with Different Keys** ✅ SUPPORTED
```go
// v1 (current fog usage)
output.Keys = []string{"Name", "Email"}
output.AddContents(userData)
output.AddToBuffer()
output.Keys = []string{"ID", "Status"}
output.AddContents(statusData)
output.Write()

// v2 equivalent
doc := output.New().
    Table("Users", userData, output.WithKeys("Name", "Email")).
    Table("Status", statusData, output.WithKeys("ID", "Status")).
    Build()
```
**Status**: Fully supported with cleaner API

---

## Gap Resolution Status (v2.2.1)

### ✅ RESOLVED: Inline Color/Styling Methods (v2.2.1)

**Status**: FULLY RESOLVED

#### Current Fog Usage
```go
// Used 10+ times in drift.go
changetype = outputsettings.StringWarningInline(changetype)
properties = append(properties, outputsettings.StringWarningInline(fmt.Sprintf(...)))
properties = append(properties, outputsettings.StringPositiveInline(fmt.Sprintf(...)))
```

#### V2.2.1 Solution
The v2.2.1 release added stateless inline styling functions:

```go
// Direct replacements for v1 methods
changetype = output.StyleWarning(changetype)  // Red bold (replaces StringWarningInline)
text = output.StylePositive(text)             // Green bold (replaces StringPositiveInline)
text = output.StyleNegative(text)             // Red (replaces StringNegativeInline)
text = output.StyleInfo(text)                 // Blue
text = output.StyleBold(text)                 // Bold

// Conditional styling (apply only if condition is true)
text = output.StyleWarningIf(isDrifted, text)
text = output.StylePositiveIf(isHealthy, text)
```

#### Migration Example
```go
// v1 (fog current usage)
changetype = outputsettings.StringWarningInline(changetype)

// v2.2.1 (direct replacement)
changetype = output.StyleWarning(changetype)

// Or with conditional styling
changetype = output.StyleWarningIf(
    drift.StackResourceDriftStatus == types.StackResourceDriftStatusDeleted,
    changetype,
)
```

**Benefits**:
- ✅ No workarounds needed
- ✅ Stateless functions (thread-safe)
- ✅ Consistent ANSI color codes
- ✅ Conditional variants for cleaner code

---

### ✅ RESOLVED: TableMaxColumnWidth Configuration (v2.2.1)

**Status**: FULLY RESOLVED

#### Current Fog Usage
```go
// config/config.go
settings.TableMaxColumnWidth = config.GetInt("table.max-column-width")

// viper default in root.go
viper.SetDefault("table.max-column-width", 50)
```

#### V2.2.1 Solution
The v2.2.1 release added table format constructors with max column width:

```go
// Option 1: Max column width only
format := output.TableWithMaxColumnWidth(50)

// Option 2: Style + max column width
format := output.TableWithStyleAndMaxColumnWidth("Default", 50)

// Use in fog's config
func (config *Config) NewOutputSettings() output.Format {
    maxWidth := config.GetInt("table.max-column-width")
    if maxWidth > 0 {
        return output.TableWithMaxColumnWidth(maxWidth)
    }
    return output.Table
}
```

#### Migration Example
```go
// v1 (fog current usage)
settings := format.NewOutputSettings()
settings.TableMaxColumnWidth = 50

output := format.OutputArray{
    Settings: settings,
}

// v2.2.1 (direct replacement)
format := output.TableWithMaxColumnWidth(50)
out := output.NewOutput(
    output.WithFormat(format),
    output.WithWriter(output.NewStdoutWriter()),
)
```

**How it works**:
- Uses go-pretty's `SetColumnConfigs()` with `WidthMax`
- Automatically wraps text within cells when content exceeds width
- Works with all table styles

**Benefits**:
- ✅ Native support, no formatters needed
- ✅ Automatic text wrapping
- ✅ Compatible with table styles
- ✅ Simple configuration

---

### ✅ RESOLVED: Format-Aware Separator Handling (v2.2.1)

**Status**: FULLY RESOLVED via automatic array handling

#### Current Fog Usage
```go
// config/config.go
func (config *Config) GetSeparator() string {
    switch config.GetLCString("output") {
    case "table":
        return "\r\n"
    case "dot":
        return ","
    default:
        return ", "
    }
}

// Used in drift.go, exports.go, dependencies.go
ruledetails := fmt.Sprintf("Expected: %s%sActual: %s",
    expected, outputsettings.GetSeparator(), actual)
content["Imported By"] = strings.Join(resources, settings.GetSeparator())
```

#### V2.2.1 Solution
The v2.2.1 release added format-aware automatic array handling:

```go
// Instead of manually joining with separator
content["Imported By"] = strings.Join(resources, settings.GetSeparator())

// Just pass the array directly - v2 handles format-appropriate rendering
content["Imported By"] = resources  // []string

// v2 automatically renders:
// - Table format: newline-separated (\n)
// - CSV format: semicolon-separated (;)
// - JSON/YAML: native array structure
// - Markdown: HTML <br/> tags
```

#### Migration Strategy

**Option A: Use automatic array handling (RECOMMENDED)**
```go
// Simply pass arrays to v2
content["Properties"] = properties        // []string
content["Imported By"] = importedBy      // []string
content["Tags"] = tags                   // []string

// v2 automatically formats based on output type
```

**Option B: Keep GetSeparator() for backward compatibility**
```go
// Keep existing code, update GetSeparator() for v2 formats
func (config *Config) GetSeparator() string {
    switch config.GetLCString("output") {
    case "table", "markdown":
        return "\n"  // v2 uses \n instead of \r\n
    case "dot":
        return ","
    case "csv":
        return ";"
    default:
        return ", "
    }
}
// No other code changes needed
```

**Benefits**:
- ✅ Format-appropriate separators automatically applied
- ✅ Cleaner code (no manual joining)
- ✅ Better rendering in each format
- ✅ Native array support in JSON/YAML

---

### 🟢 Title Setting

**Impact**: VERY LOW
**Likelihood of Blocking**: NONE

#### Current Fog Usage
```go
output.Settings.Title = resultTitle
```

#### V2 Equivalent
```go
doc := output.New().
    Header(resultTitle).
    Table("", data).
    Build()
```

**Status**: Fully supported via `Header()` or table title parameter

---

## Features Fog Does NOT Use (Not Gaps)

These v2 features are available but fog doesn't currently need them:

- ❌ S3 output writers
- ❌ Mermaid diagram generation
- ❌ Draw.io CSV output
- ❌ HTML output format
- ❌ YAML output format
- ❌ Markdown output format
- ❌ Progress indicators (though v1 has support, fog doesn't use it)
- ❌ Collapsible content system (v2.1.0+)
- ❌ Data transformation pipelines (filter, aggregate)
- ❌ AWS Icons for Draw.io (v2.2.0+)
- ❌ Custom transformers beyond color/emoji/sort

---

## Migration Complexity Assessment

### Code Volume to Change
- **~64 OutputArray instantiations** → Builder pattern conversions
- **~20 OutputSettings configurations** → Functional option conversions
- **~10 inline styling calls** → Workaround implementation required
- **5 GetSeparator calls** → No change needed (keep in config)
- **Multiple imports** → Change from v1 to v2 path

### Estimated Migration Effort
- **Low complexity changes**: 60% (straightforward pattern replacements)
- **Medium complexity changes**: 30% (refactoring multi-table logic)
- **High complexity changes**: 10% (inline styling workarounds)

### Risk Assessment
- **Breaking changes**: HIGH (v2 is not backward compatible)
- **Testing required**: HIGH (all output formats need validation)
- **Rollback complexity**: MEDIUM (can keep v1 in parallel during migration)

---

## Recommendations

### Recommendation: ✅ PROCEED WITH MIGRATION

**All identified gaps have been resolved in v2.2.1. Migration can proceed with confidence.**

The migration to go-output v2.2.1 is now **straightforward** with direct replacements for all fog features:

1. ✅ Inline styling → `StyleWarning()`, `StylePositive()`, etc.
2. ✅ Column width → `TableWithMaxColumnWidth()`
3. ✅ Separators → Automatic array handling

**No workarounds or custom implementations needed.**

### Migration Checklist

#### 1. **Update Dependencies**
- [ ] Update go.mod to use go-output v2.2.1+
- [ ] Run `go mod tidy`
- [ ] Verify no conflicts with other dependencies

#### 2. **Update Inline Styling (Simple find-replace)**
- [ ] Replace `outputsettings.StringWarningInline(x)` → `output.StyleWarning(x)`
- [ ] Replace `outputsettings.StringPositiveInline(x)` → `output.StylePositive(x)`
- [ ] Replace `outputsettings.StringNegativeInline(x)` → `output.StyleNegative(x)`
- [ ] Update imports to include v2 package

#### 3. **Update Table Configuration**
- [ ] Replace `settings.TableMaxColumnWidth` with `TableWithMaxColumnWidth()`
- [ ] Update config.NewOutputSettings() to return Format instead of OutputSettings
- [ ] Test column wrapping with sample data

#### 4. **Optimize Array Handling (Optional)**
- [ ] Identify places using GetSeparator() + strings.Join()
- [ ] Consider replacing with direct array assignment
- [ ] Test output quality across all formats
- [ ] Keep GetSeparator() if backward compatibility preferred

### Streamlined Migration Approach

With all gaps resolved, the migration is now significantly simplified:

#### Phase 1: Setup and Simple Commands (1 day)
1. Create branch: `feature/go-output-v2-migration`
2. Update go.mod to v2.2.1+
3. Update config/config.go with new Format constructors
4. Migrate 2-3 simple commands (exports, dependencies, history)
5. Test all output formats

#### Phase 2: Core Commands (2-3 days)
1. Migrate deploy, describe, report commands
2. Update OutputArray → Builder pattern
3. Update OutputSettings → Functional options
4. Test with real CloudFormation stacks

#### Phase 3: Drift Command (2 days)
1. Migrate drift.go inline styling calls
2. Update GetSeparator() usage (or switch to arrays)
3. Test drift detection output thoroughly
4. Run integration tests

#### Phase 4: Final Testing and Release (1 day)
1. Full regression testing on all commands
2. Test on both macOS and Linux
3. Update documentation and CHANGELOG
4. Release as minor version bump (fog v2.x.0)

**Total Estimated Time: 1 week** (vs. original estimate of 2-3 weeks)

### Success Criteria

Migration is complete when:
- ✅ All commands use go-output v2.2.1
- ✅ All output formats work correctly (table, csv, json, dot)
- ✅ Inline styling preserved in drift detection
- ✅ Table column width limits work as expected
- ✅ File output functions properly
- ✅ All integration tests pass
- ✅ No v1 dependencies remain

---

## Benefits of Migration

### Technical Benefits
1. **Better Architecture**: Builder pattern eliminates global state
2. **Thread Safety**: Safe concurrent use for future parallelization
3. **Key Order Preservation**: Guaranteed column ordering
4. **Improved Error Handling**: Context-aware errors with wrapping
5. **Data Pipelines**: Future-proof for filtering/aggregation needs
6. **Collapsible Content**: Better UX for complex drift details (future)
7. **Additional Formats**: HTML, Markdown, YAML available if needed

### Maintenance Benefits
1. **Active Development**: v2 actively maintained, v1 may be deprecated
2. **Better Testing**: v2 has comprehensive test suite
3. **Clear Patterns**: Functional options more maintainable than settings objects
4. **Documentation**: Excellent API docs and migration guides

### User Benefits
1. **Consistent Output**: Better rendering across formats
2. **Future Features**: Access to collapsible content, data pipelines
3. **Better Error Messages**: More helpful error context
4. **Enhanced Formats**: Potential for Markdown reports, HTML dashboards

---

## Alternative: Stay on V1

### When to Consider
- If inline styling proves impossible to work around
- If migration effort exceeds available resources
- If v1 continues to receive security/bug fixes
- If fog is in maintenance-only mode

### Risks of Staying on V1
- **Deprecation Risk**: V1 may be deprecated as v2 matures
- **Feature Gap**: Missing out on pipelines, collapsibles, new formats
- **Bug Fixes**: Future fixes may only target v2
- **Community Shift**: Support questions may focus on v2

---

## Conclusion

**✅ Go-output v2.2.1 is fully ready for fog's migration - all gaps resolved!**

The v2.2.1 release provides **complete feature parity** with v1 and significant improvements. All three previously identified gaps have been addressed:

1. ✅ **Inline styling**: Direct replacements available - `StyleWarning()`, `StylePositive()`, etc.
2. ✅ **Column width**: Native support via `TableWithMaxColumnWidth()`
3. ✅ **Separators**: Automatic format-aware array handling

**Immediate next steps**:
1. ✅ Update go.mod to go-output v2.2.1+ (5 minutes)
2. ✅ Begin Phase 1 migration (simple commands) (1 day)
3. ✅ Continue with incremental migration (1 week total)

**Updated timeline**: **1 week for complete migration** (down from original 2-3 weeks estimate)

**Updated risk level**: **LOW** - straightforward migration with direct API replacements and no workarounds needed

---

## Version Requirement

**Minimum Required Version**: go-output v2.2.1

This version includes all critical features needed by fog:
- Inline styling functions (`StyleWarning`, `StylePositive`, etc.)
- Table max column width configuration
- Format-aware array handling

Earlier v2 versions (v2.0.0 - v2.2.0) lack these features and should not be used for migration.
