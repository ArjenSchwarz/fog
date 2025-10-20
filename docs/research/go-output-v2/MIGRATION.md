# Go-Output v2 Migration Guide

This guide provides comprehensive instructions for migrating from go-output v1 to v2. Version 2 is a complete redesign that eliminates global state, provides thread safety, and maintains exact key ordering while preserving all v1 functionality.

## Table of Contents

- [Overview](#overview)
- [Breaking Changes](#breaking-changes)
- [Automated Migration](#automated-migration)
- [Migration Patterns](#migration-patterns)
  - [Basic Output](#basic-output)
  - [Multiple Tables](#multiple-tables)
  - [Output Settings](#output-settings)
  - [Progress Indicators](#progress-indicators)
  - [File Output](#file-output)
  - [S3 Output](#s3-output)
  - [Chart and Diagram Output](#chart-and-diagram-output)
- [Feature-by-Feature Migration](#feature-by-feature-migration)
- [Common Issues](#common-issues)
- [Examples](#examples)

## Overview

Go-Output v2 introduces a clean, modern API while maintaining feature parity with v1. The main changes are:

- **No Global State**: All state is encapsulated in instances
- **Builder Pattern**: Fluent API for document construction
- **Functional Options**: Configuration through option functions
- **Key Order Preservation**: Exact user-specified key ordering is maintained
- **Thread Safety**: Safe for concurrent use

## Breaking Changes

### 1. Import Path
```go
// v1
import "github.com/ArjenSchwarz/go-output/format"

// v2
import "github.com/ArjenSchwarz/go-output/v2"
```

### 2. OutputArray Replaced with Builder Pattern
```go
// v1
output := &format.OutputArray{
    Keys: []string{"Name", "Age"},
}

// v2
doc := output.New().
    Table("", data, output.WithKeys("Name", "Age")).
    Build()
```

### 3. OutputSettings Replaced with Functional Options
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.UseEmoji = true

// v2
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformer(&output.EmojiTransformer{}),
)
```

### 4. Write() Method Requires Context
```go
// v1
output.Write()

// v2
output.NewOutput().Render(ctx, doc)
```

### 5. Keys Field Replaced with Schema Options
```go
// v1
output.Keys = []string{"Name", "Age", "Status"}

// v2
output.WithKeys("Name", "Age", "Status")
// or
output.WithSchema(
    output.Field{Name: "Name"},
    output.Field{Name: "Age"},
    output.Field{Name: "Status"},
)
```

## Automated Migration

Use the included migration tool to automatically convert most v1 code:

```bash
# Install the migration tool
go install github.com/ArjenSchwarz/go-output/v2/migrate/cmd/migrate@latest

# Migrate a single file
migrate -file main.go

# Migrate an entire directory
migrate -source ./myproject

# Dry run to see changes without applying them
migrate -source ./myproject -dry-run

# Verbose mode for detailed information
migrate -source ./myproject -verbose
```

The migration tool handles approximately 80% of common v1 usage patterns. Manual adjustments may be needed for complex scenarios.

## Migration Patterns

### Basic Output

#### Simple Table Output
```go
// v1
output := &format.OutputArray{}
output.AddContents(map[string]interface{}{
    "Name": "Alice",
    "Age":  30,
})
output.Write()

// v2
ctx := context.Background()
doc := output.New().
    Table("", []map[string]interface{}{
        {"Name": "Alice", "Age": 30},
    }).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
```

#### Table with Key Ordering
```go
// v1
output := &format.OutputArray{
    Keys: []string{"ID", "Name", "Status"},
}
output.AddContents(data)
output.Write()

// v2
ctx := context.Background()
doc := output.New().
    Table("", data, output.WithKeys("ID", "Name", "Status")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
```

### Multiple Tables

#### Multiple Tables with Different Keys
```go
// v1
output := &format.OutputArray{}

// First table
output.Keys = []string{"Name", "Email"}
output.AddContents(userData)
output.AddToBuffer()

// Second table
output.Keys = []string{"ID", "Status", "Time"}
output.AddContents(statusData)
output.AddToBuffer()

output.Write()

// v2
ctx := context.Background()
doc := output.New().
    Table("Users", userData, output.WithKeys("Name", "Email")).
    Table("Status", statusData, output.WithKeys("ID", "Status", "Time")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
```

#### Tables with Headers
```go
// v1
output := &format.OutputArray{}
output.AddHeader("User Report")
output.Keys = []string{"Name", "Role"}
output.AddContents(users)
output.Write()

// v2
ctx := context.Background()
doc := output.New().
    Header("User Report").
    Table("", users, output.WithKeys("Name", "Role")).
    Build()

output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
```

### Output Settings

#### Basic Settings Migration
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFormat = "json"
settings.OutputFile = "report.json"
settings.UseEmoji = true
settings.UseColors = true

output := &format.OutputArray{
    Settings: settings,
}

// v2
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.ColorTransformer{}),
)
```

#### Table Styling
```go
// v1
settings := format.NewOutputSettings()
settings.TableStyle = "ColoredBright"

// v2
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTableStyle("ColoredBright"),
)
```

#### Multiple Output Formats
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFormat = "table"
settings.OutputFile = "report.html"
settings.OutputFileFormat = "html"

// v2
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithFormat(output.HTML),
    output.WithWriter(&output.StdoutWriter{}),
    output.WithWriter(output.NewFileWriter(".", "report.html")),
)
```

### Progress Indicators

#### Basic Progress
```go
// v1
settings := format.NewOutputSettings()
p := format.NewProgress(settings)
p.SetTotal(100)
p.SetColor(format.ProgressColorGreen)

for i := 0; i < 100; i++ {
    p.Increment(1)
    p.SetStatus(fmt.Sprintf("Processing item %d", i))
}
p.Complete()

// v2
p := output.NewProgress(output.Table,
    output.WithProgressColor(output.ProgressColorGreen),
)
p.SetTotal(100)

for i := 0; i < 100; i++ {
    p.Increment(1)
    p.SetStatus(fmt.Sprintf("Processing item %d", i))
}
p.Complete()
```

#### Progress with Output
```go
// v1
settings := format.NewOutputSettings()
settings.SetOutputFormat("table")
settings.ProgressOptions = format.ProgressOptions{
    Color: format.ProgressColorBlue,
    Status: "Loading data",
}

// v2
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithProgress(output.NewProgress(output.Table,
        output.WithProgressColor(output.ProgressColorBlue),
        output.WithProgressStatus("Loading data"),
    )),
)
```

### File Output

#### Simple File Output
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFile = "report.csv"
settings.OutputFormat = "csv"

// v2
out := output.NewOutput(
    output.WithFormat(output.CSV),
    output.WithWriter(output.NewFileWriter(".", "report.csv")),
)
```

#### Multiple File Outputs
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFile = "report.json"
settings.OutputFileFormat = "json"
settings.OutputFormat = "table" // For stdout

// v2
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithFormat(output.JSON),
    output.WithWriter(&output.StdoutWriter{}),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
)
```

### S3 Output

v2 S3Writer is fully compatible with AWS SDK v2 and requires no adapter:

```go
// v1
settings := format.NewOutputSettings()
settings.OutputS3Bucket = "my-bucket"
settings.OutputS3Key = "reports/output.json"

// v2 - Works directly with AWS SDK v2
import (
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/ArjenSchwarz/go-output/v2"
)

// Load AWS config
cfg, err := config.LoadDefaultConfig(context.TODO())
if err != nil {
    log.Fatal(err)
}

// Create S3 client from AWS SDK v2
s3Client := s3.NewFromConfig(cfg)

// Use S3 client directly with go-output
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewS3Writer(s3Client, "my-bucket", "reports/output.json")),
)

// Render to S3
err = out.Render(context.Background(), doc)
```

**Key Points:**
- No adapter needed - S3Writer accepts AWS SDK v2 `*s3.Client` directly
- The S3Writer interface is compatible with `s3.Client.PutObject`
- Easy to mock for testing using the `S3PutObjectAPI` interface
- Supports key patterns like `"reports/{format}.{ext}"` for dynamic naming

### Chart and Diagram Output

#### DOT Format (Graphviz)
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFormat = "dot"
settings.DotFromColumn = "source"
settings.DotToColumn = "target"

// v2
doc := output.New().
    Graph("Network", data, 
        output.WithFromTo("source", "target"),
    ).
    Build()

out := output.NewOutput(output.WithFormat(output.DOT))
```

#### Mermaid Charts
```go
// v1
settings := format.NewOutputSettings()
settings.OutputFormat = "mermaid"
settings.MermaidSettings = &mermaid.Settings{
    ChartType: "gantt",
}

// v2
doc := output.New().
    Chart("Project Timeline", ganttData,
        output.WithChartType("gantt"),
    ).
    Build()

out := output.NewOutput(output.WithFormat(output.Mermaid))
```

#### Draw.io Diagrams
```go
// v1
drawio.SetHeaderValues(drawio.Header{
    Label: "%Name%",
    Style: "%Image%",
    Layout: "horizontalflow",
})

// v2
doc := output.New().
    DrawIO("Architecture", data,
        output.WithDrawIOLayout("horizontalflow"),
        output.WithDrawIOLabel("%Name%"),
        output.WithDrawIOStyle("%Image%"),
    ).
    Build()

out := output.NewOutput(output.WithFormat(output.DrawIO))
```

#### AWS Icons for Draw.io

The AWS icon functionality has moved from `drawio` package to the new `icons` package with improved error handling:

```go
// v1
import "github.com/ArjenSchwarz/go-output/drawio"

style := drawio.GetAWSShape("Compute", "EC2")
// Returns empty string if shape not found - no error indication

// v2
import "github.com/ArjenSchwarz/go-output/v2/icons"

style, err := icons.GetAWSShape("Compute", "EC2")
if err != nil {
    // Explicit error: "shape group not found" or "shape not found in group"
    log.Fatal(err)
}
```

**Discovery helpers** (new in v2):

```go
// List all AWS service groups
groups := icons.AllAWSGroups()

// List all shapes in a group
shapes, err := icons.AWSShapesInGroup("Compute")

// Check if shape exists
if icons.HasAWSShape("Compute", "Lambda") {
    // Shape is available
}
```

**Complete Draw.io + AWS Icons example:**

```go
// v1 approach
import (
    "github.com/ArjenSchwarz/go-output/drawio"
    "github.com/ArjenSchwarz/go-output/format"
)

output := &format.OutputArray{}
for _, item := range data {
    // Get icon style (no error checking possible)
    style := drawio.GetAWSShape(item.Group, item.Service)
    output.AddRow(map[string]any{
        "Name": item.Name,
        "Icon": style,
    })
}
drawio.SetHeaderValues(drawio.Header{Style: "%Icon%"})
output.OutputFormat = "drawio"
output.Write()

// v2 approach
import (
    "github.com/ArjenSchwarz/go-output/v2"
    "github.com/ArjenSchwarz/go-output/v2/icons"
)

records := []output.Record{}
for _, item := range data {
    // Get icon style with proper error handling
    style, err := icons.GetAWSShape(item.Group, item.Service)
    if err != nil {
        log.Printf("Warning: icon not found for %s/%s: %v", item.Group, item.Service, err)
        style = "" // Use default/fallback
    }
    records = append(records, output.Record{
        "Name": item.Name,
        "Icon": style,
    })
}

doc := output.New().
    DrawIO("AWS Architecture", records, output.DrawIOHeader{
        Style: "%Icon%",
        Label: "%Name%",
    }).
    Build()

out := output.NewOutput(
    output.WithFormat(output.DrawIO),
    output.WithWriter(output.NewStdoutWriter()),
)
out.Render(context.Background(), doc)
```

## Feature-by-Feature Migration

### Sorting
```go
// v1
settings.SortKey = "Name"

// v2
output.WithTransformer(&output.SortTransformer{
    Key: "Name",
    Ascending: true,
})
```

### Line Splitting
```go
// v1
settings.LineSplitColumn = "Description"
settings.LineSplitSeparator = ","

// v2
output.WithTransformer(&output.LineSplitTransformer{
    Column: "Description",
    Separator: ",",
})
```

### Table of Contents (Markdown)
```go
// v1
settings.HasTOC = true

// v2
output.WithTOC(true)
```

### Front Matter (Markdown)
```go
// v1
settings.FrontMatter = map[string]string{
    "title": "Report",
    "date": "2024-01-01",
}

// v2
output.WithFrontMatter(map[string]string{
    "title": "Report",
    "date": "2024-01-01",
})
```

### Inline Color and Styling (v2.3.0+)

v1 used global styling methods that relied on package-level state. v2 provides stateless inline styling functions that are thread-safe and can be used directly in data:

```go
// v1
import "github.com/ArjenSchwarz/go-output/format"

// Global styling methods
format.StyleRed("Error occurred")
format.StyleGreen("Success")
format.StyleBold("Important")

// v2
import output "github.com/ArjenSchwarz/go-output/v2"

// Stateless inline styling functions
output.StyleWarning("Error occurred")    // Red bold
output.StylePositive("Success")          // Green bold
output.StyleNegative("Failed")           // Red
output.StyleInfo("Information")          // Blue
output.StyleBold("Important")            // Bold

// Conditional styling (new in v2)
output.StyleWarningIf(hasErrors, "Status: Error")
output.StylePositiveIf(isSuccess, "Status: Success")

// Use directly in table data
data := []map[string]any{
    {
        "Server": "web-01",
        "Status": output.StylePositive("Running"),
        "CPU":    output.StyleWarningIf(cpu > 80, fmt.Sprintf("%d%%", cpu)),
    },
}
```

**Key Differences**:
- v2 functions are stateless and thread-safe
- Conditional styling variants (`*If` suffix) are new in v2
- v2 automatically enables colors even in non-TTY environments
- Use `RemoveColorsTransformer` to strip ANSI codes for non-terminal formats

### Table Max Column Width (v2.3.0+)

v1 supported table column width configuration through settings. v2 provides this through format constructors and renderer options:

```go
// v1
settings := format.NewOutputSettings()
settings.TableMaxColumnWidth = 40

output := &format.OutputArray{
    Settings: settings,
}

// v2 - Using format constructor
format := output.TableWithMaxColumnWidth(40)
out := output.NewOutput(
    output.WithFormat(format),
    output.WithWriter(output.NewStdoutWriter()),
)

// v2 - Using format constructor with style
format := output.TableWithStyleAndMaxColumnWidth("Bold", 40)

// v2 - Using renderer directly
renderer := output.NewTableRendererWithStyleAndWidth("Default", 40)
```

**Usage Example**:
```go
// Long descriptions that need wrapping
data := []map[string]any{
    {
        "Name": "Alice",
        "Description": "This is a very long description that would normally make the table extremely wide",
    },
}

doc := output.New().
    Table("Users", data, output.WithKeys("Name", "Description")).
    Build()

// Limit column width to 30 characters (text wraps automatically)
out := output.NewOutput(
    output.WithFormat(output.TableWithMaxColumnWidth(30)),
    output.WithWriter(output.NewStdoutWriter()),
)

err := out.Render(context.Background(), doc)
```

**Notes**:
- Text wraps within cells (does not truncate)
- Uses go-pretty's `WidthMax` configuration
- Works with all table styles
- Particularly useful for terminal output with limited horizontal space

### Array/Slice Handling (v2.3.0+)

v2 automatically handles arrays in table data with format-appropriate rendering:

```go
// v1 - Arrays required manual formatting
data := []map[string]any{
    {
        "Name": "Alice",
        "Tags": strings.Join([]string{"admin", "developer"}, ", "),
    },
}

// v2 - Arrays handled automatically
data := []map[string]any{
    {
        "Name": "Alice",
        "Tags": []string{"admin", "developer", "reviewer"},
        "Roles": []string{"Owner", "Maintainer"},
    },
}

doc := output.New().
    Table("Users", data, output.WithKeys("Name", "Tags", "Roles")).
    Build()

// Table format: Renders arrays as newline-separated values
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithWriter(output.NewStdoutWriter()),
)
// Output:
// ┌───────┬───────────┬──────────────┐
// │ NAME  │ TAGS      │ ROLES        │
// ├───────┼───────────┼──────────────┤
// │ Alice │ admin     │ Owner        │
// │       │ developer │ Maintainer   │
// │       │ reviewer  │              │
// └───────┴───────────┴──────────────┘

// Markdown format: Renders arrays as <br/>-separated values
out = output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithWriter(output.NewStdoutWriter()),
)
// Output:
// | Name  | Tags                              | Roles                  |
// |-------|-----------------------------------|------------------------|
// | Alice | admin<br/>developer<br/>reviewer  | Owner<br/>Maintainer   |

// JSON/YAML: Arrays preserved natively
out = output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewStdoutWriter()),
)
// Output: {"Name": "Alice", "Tags": ["admin", "developer", "reviewer"], ...}
```

**Supported Array Types**:
- `[]string` - String slices
- `[]any` - Generic slices with any element type
- Empty arrays render as empty strings in table/markdown

**Format-Specific Behavior**:
- **Table**: Newline-separated for vertical layout
- **Markdown**: `<br/>` tags for GitHub/GitLab compatibility
- **JSON/YAML**: Native array structure
- **CSV**: Semicolon-separated by default

## Common Issues

### 1. Context Required
v2 requires a context for all rendering operations:
```go
ctx := context.Background()
// or with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 2. Key Order Not Preserved
Always use `WithKeys()` or `WithSchema()` to ensure key order:
```go
// This may not preserve order
doc := output.New().Table("", data).Build()

// This will preserve order
doc := output.New().
    Table("", data, output.WithKeys("ID", "Name", "Status")).
    Build()
```

### 3. Multiple Outputs
v2 handles multiple outputs more elegantly:
```go
// Create once, render to multiple formats/destinations
doc := output.New().Table("", data).Build()

out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithFormat(output.CSV),
    output.WithWriter(&output.StdoutWriter{}),
    output.WithWriter(output.NewFileWriter(".", "report.csv")),
)

err := out.Render(ctx, doc)
```

### 4. Error Handling
v2 provides better error context:
```go
err := out.Render(ctx, doc)
if err != nil {
    var renderErr *output.RenderError
    if errors.As(err, &renderErr) {
        log.Printf("Failed to render %s: %v", renderErr.Format, renderErr.Cause)
    }
}
```

## Examples

### Complete Example: Report Generation
```go
package main

import (
    "context"
    "log"
    
    "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    ctx := context.Background()
    
    // Sample data
    users := []map[string]interface{}{
        {"ID": 1, "Name": "Alice", "Role": "Admin"},
        {"ID": 2, "Name": "Bob", "Role": "User"},
    }
    
    stats := []map[string]interface{}{
        {"Metric": "Total Users", "Value": 2},
        {"Metric": "Active Sessions", "Value": 5},
    }
    
    // Build document
    doc := output.New().
        Header("System Report").
        Table("Users", users, output.WithKeys("ID", "Name", "Role")).
        Table("Statistics", stats, output.WithKeys("Metric", "Value")).
        Build()
    
    // Configure output
    out := output.NewOutput(
        // Multiple formats
        output.WithFormat(output.Table),
        output.WithFormat(output.JSON),
        
        // Multiple destinations
        output.WithWriter(&output.StdoutWriter{}),
        output.WithWriter(output.NewFileWriter(".", "report.json")),
        
        // Transformers
        output.WithTransformer(&output.ColorTransformer{}),
        
        // Table styling
        output.WithTableStyle("ColoredBright"),
    )
    
    // Render
    if err := out.Render(ctx, doc); err != nil {
        log.Fatalf("Failed to render: %v", err)
    }
}
```

### Example: Progress with Data Processing
```go
package main

import (
    "context"
    "time"
    
    "github.com/ArjenSchwarz/go-output/v2"
)

func processData(ctx context.Context) {
    // Create progress indicator
    progress := output.NewProgress(output.Table,
        output.WithProgressColor(output.ProgressColorGreen),
        output.WithProgressStatus("Processing records"),
    )
    
    // Set total items
    progress.SetTotal(100)
    
    // Process data
    for i := 0; i < 100; i++ {
        // Simulate work
        time.Sleep(50 * time.Millisecond)
        
        // Update progress
        progress.Increment(1)
        progress.SetStatus(fmt.Sprintf("Processing record %d/100", i+1))
        
        // Check context cancellation
        select {
        case <-ctx.Done():
            progress.Fail(ctx.Err())
            return
        default:
        }
    }
    
    // Complete
    progress.Complete()
}
```

## Enhanced Field.Formatter and Collapsible Content

### Field.Formatter Signature Change

**v2 introduces a significant enhancement to the `Field.Formatter` function signature to support rich collapsible content across all output formats.**

#### Signature Change
```go
// v1 and early v2
type Field struct {
    Name      string
    Type      string
    Formatter func(any) string  // OLD: Returns only strings
    Hidden    bool
}

// v2 Enhanced (Current)
type Field struct {
    Name      string
    Type      string
    Formatter func(any) any     // NEW: Can return CollapsibleValue or strings
    Hidden    bool
}
```

#### Backward Compatibility

**All existing Field.Formatter functions continue to work without modification.** The change is fully backward compatible:

```go
// Existing v2 formatters continue to work unchanged
func upperFormatter(val any) any {
    return strings.ToUpper(fmt.Sprint(val))  // Still works!
}

// You can also return strings directly (backward compatible)
func oldStyleFormatter(val any) any {
    return fmt.Sprintf("Value: %v", val)     // Still works!
}
```

#### New Collapsible Content Support

**The enhanced signature enables CollapsibleValue returns for expandable content:**

```go
import "github.com/ArjenSchwarz/go-output/v2"

// New: Return CollapsibleValue for expandable content
func errorListFormatter(val any) any {
    if errors, ok := val.([]string); ok && len(errors) > 0 {
        return output.NewCollapsibleValue(
            fmt.Sprintf("%d errors", len(errors)),  // Summary view
            errors,                                  // Detailed content
            output.WithCollapsibleExpanded(false),             // Collapsed by default
        )
    }
    return val  // Return unchanged for non-arrays
}

// Use the formatter in a schema
schema := output.WithSchema(
    output.Field{
        Name: "errors",
        Type: "array", 
        Formatter: errorListFormatter,  // Enhanced formatter
    },
)
```

### Built-in Collapsible Formatters

**v2 provides pre-built formatters for common collapsible patterns:**

#### Error List Formatter
```go
// Automatically creates collapsible content for error arrays
doc := output.New().
    Table("Issues", data, output.WithSchema(
        output.Field{Name: "file", Type: "string"},
        output.Field{
            Name: "errors", 
            Type: "array",
            Formatter: output.ErrorListFormatter(),  // Built-in formatter
        },
    )).
    Build()
```

#### File Path Formatter
```go
// Shows abbreviated paths with full path in details
doc := output.New().
    Table("Files", data, output.WithSchema(
        output.Field{
            Name: "path",
            Type: "string", 
            Formatter: output.FilePathFormatter(30),  // Truncate at 30 chars
        },
    )).
    Build()
```

#### JSON Formatter
```go
// Shows compact summary for large JSON objects
doc := output.New().
    Table("Config", data, output.WithSchema(
        output.Field{
            Name: "settings",
            Type: "object",
            Formatter: output.JSONFormatter(100),  // Collapse if > 100 chars
        },
    )).
    Build()
```

### Cross-Format Rendering

**Collapsible content adapts to each output format:**

```go
data := []map[string]any{
    {
        "file": "/very/long/path/to/project/components/UserProfile.tsx",
        "errors": []string{
            "Missing import for React",
            "Unused variable 'userData'",
            "Type annotation missing",
        },
    },
}

table := output.NewTableContent("Issues", data, output.WithSchema(
    output.Field{Name: "file", Type: "string", Formatter: output.FilePathFormatter(25)},
    output.Field{Name: "errors", Type: "array", Formatter: output.ErrorListFormatter()},
))

doc := output.New().Add(table).Build()

// Markdown: Creates GitHub-compatible <details> elements
output.NewOutput(output.WithFormat(output.Markdown)).Render(ctx, doc)
// Output: <details><summary>3 errors</summary>Missing import...<br/>Unused variable...</details>

// JSON: Structured data with type indicators
output.NewOutput(output.WithFormat(output.JSON)).Render(ctx, doc)
// Output: {"type": "collapsible", "summary": "3 errors", "details": [...], "expanded": false}

// Table: Summary with expansion indicators
output.NewOutput(output.WithFormat(output.Table)).Render(ctx, doc)
// Output: 3 errors [details hidden - use --expand for full view]

// CSV: Automatic detail columns
output.NewOutput(output.WithFormat(output.CSV)).Render(ctx, doc)
// Creates: errors, errors_details columns
```

### Migration Steps for Field.Formatter

#### Step 1: Review Existing Formatters
**No immediate action required** - existing formatters continue to work:

```go
// This continues to work unchanged
func myFormatter(val any) any {
    return fmt.Sprintf("Custom: %v", val)
}
```

#### Step 2: Optional Enhancement
**Add collapsible support where beneficial:**

```go
// Before: Simple string formatter
func oldErrorFormatter(val any) any {
    if errors, ok := val.([]string); ok {
        return strings.Join(errors, ", ")  // Simple concatenation
    }
    return val
}

// After: Enhanced with collapsible support
func newErrorFormatter(val any) any {
    if errors, ok := val.([]string); ok && len(errors) > 0 {
        return output.NewCollapsibleValue(
            fmt.Sprintf("%d errors", len(errors)),
            errors,
            output.WithCollapsibleExpanded(false),
        )
    }
    return val
}
```

#### Step 3: Use Built-in Formatters
**Replace custom implementations with built-ins where applicable:**

```go
// Before: Custom implementation
func myPathFormatter(val any) any {
    path := fmt.Sprint(val)
    if len(path) > 50 {
        return "..." + path[len(path)-47:]
    }
    return path
}

// After: Use built-in with collapsible support
output.Field{
    Name: "path",
    Type: "string",
    Formatter: output.FilePathFormatter(50),  // Built-in with collapsible details
}
```

### Collapsible Sections

**v2 also supports section-level collapsible content:**

```go
// Create collapsible sections containing entire tables or content blocks
analysisSection := output.NewCollapsibleTable(
    "Detailed Analysis Results",
    tableContent,
    output.WithSectionExpanded(false),
)

reportSection := output.NewCollapsibleReport(
    "Performance Report", 
    []output.Content{
        output.NewTextContent("System analysis complete"),
        tableContent,
        output.NewTextContent("All systems operational"),
    },
    output.WithSectionExpanded(true),
)

doc := output.New().
    Add(analysisSection).
    Add(reportSection).
    Build()
```

### Configuration Options

**Control collapsible behavior globally:**

```go
// Table renderer with custom expansion settings
tableRenderer := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        GlobalExpansion:      false,
        TableHiddenIndicator: "[click to expand]",
        MaxDetailLength:      200,
        TruncateIndicator:    "...",
    }),
)

// HTML renderer with custom CSS classes
htmlRenderer := output.NewOutput(
    output.WithFormat(output.HTML),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        HTMLCSSClasses: map[string]string{
            "details": "my-collapsible",
            "summary": "my-summary",
            "content": "my-details",
        },
    }),
)
```

## Data Transformation Pipeline Migration

Version 2 introduces a powerful **Data Transformation Pipeline** system that operates on structured data before rendering, providing significant advantages over the traditional byte-based transformers. This section explains how to migrate from byte transformers to data pipelines and when to use each approach.

### Overview: Two Transformation Systems

V2 maintains **both** transformation systems for different use cases:

1. **Data Transformation Pipeline** (NEW): Operates on structured data before rendering
   - Use for: Filtering, sorting, aggregation, calculated fields
   - Benefits: Type-safe, format-agnostic, performance optimized
   - Stage: Pre-rendering data manipulation

2. **Byte Transformers** (EXISTING): Operates on rendered output
   - Use for: Text styling, colors, emoji, format-specific presentation
   - Benefits: Format-specific customization, backward compatibility
   - Stage: Post-rendering text manipulation

### When to Use Each System

#### Use Data Pipeline When:
- Filtering records based on data values
- Sorting by column values
- Performing aggregations (sum, count, average)
- Adding calculated fields
- Working with data structure and content

#### Use Byte Transformers When:
- Adding colors to output
- Converting text to emoji
- Format-specific styling (ANSI codes, HTML classes)
- Post-rendering text modifications

### Migration Examples

#### Example 1: Sorting Migration

**OLD: Byte Transformer Approach**
```go
// v1/v2 byte transformer - post-rendering text manipulation
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformer(&output.SortTransformer{
        Key:       "name",
        Ascending: true,
    }),
)

// Problems:
// - Parses rendered text to extract data
// - Format-dependent implementation
// - Cannot handle complex data types
// - Performance overhead from parse/render cycle
```

**NEW: Data Pipeline Approach**
```go
// v2 data pipeline - pre-rendering data transformation
doc := output.New().
    Table("Users", userData, output.WithKeys("name", "age", "status")).
    Build()

transformedDoc := doc.Pipeline().
    SortBy("name", output.Ascending).
    Execute()

// Benefits:
// - Works directly with structured data
// - Format-agnostic (works with JSON, CSV, HTML, etc.)
// - Type-safe operations
// - Better performance (no parse/render cycle)
```

#### Example 2: Filtering Migration

**OLD: Manual Pre-filtering**
```go
// v1 approach - manual data manipulation before output
var activeUsers []map[string]any
for _, user := range allUsers {
    if user["status"] == "active" {
        activeUsers = append(activeUsers, user)
    }
}

doc := output.New().
    Table("Active Users", activeUsers).
    Build()
```

**NEW: Pipeline Filtering**
```go
// v2 pipeline approach - integrated filtering
doc := output.New().
    Table("Users", allUsers, output.WithKeys("name", "status", "last_login")).
    Build()

transformedDoc := doc.Pipeline().
    Filter(func(r output.Record) bool {
        return r["status"] == "active"
    }).
    SortBy("last_login", output.Descending).
    Limit(50).
    Execute()

// Benefits:
// - Declarative filtering logic
// - Chainable with other operations
// - Automatic optimization (filter before sort)
// - Preserves original data for comparison
```

#### Example 3: Complex Transformation Migration

**OLD: Multi-step Manual Approach**
```go
// v1 approach - multiple manual steps
// Step 1: Filter data
var salesData []map[string]any
for _, record := range rawSales {
    if record["status"] == "completed" && record["amount"].(float64) > 1000 {
        salesData = append(salesData, record)
    }
}

// Step 2: Sort data
sort.Slice(salesData, func(i, j int) bool {
    return salesData[i]["amount"].(float64) > salesData[j]["amount"].(float64)
})

// Step 3: Limit results
if len(salesData) > 25 {
    salesData = salesData[:25]
}

// Step 4: Add calculated fields manually
for _, record := range salesData {
    amount := record["amount"].(float64)
    record["commission"] = amount * 0.05
}

doc := output.New().
    Table("Top Sales", salesData).
    Build()
```

**NEW: Integrated Pipeline Approach**
```go
// v2 pipeline approach - single fluent chain
doc := output.New().
    Table("Sales", rawSales, 
        output.WithKeys("salesperson", "region", "amount", "date", "status")).
    Build()

transformedDoc := doc.Pipeline().
    // Filter high-value completed sales
    Filter(func(r output.Record) bool {
        return r["status"] == "completed" && r["amount"].(float64) > 1000
    }).
    // Add calculated commission field
    AddColumn("commission", func(r output.Record) any {
        amount := r["amount"].(float64)
        return amount * 0.05
    }).
    // Sort by amount (highest first)
    SortBy("amount", output.Descending).
    // Get top 25 results
    Limit(25).
    Execute()

// Benefits:
// - Single fluent chain
// - Automatic optimization (filter → add column → sort → limit)
// - Immutable transformations
// - Built-in error handling with context
// - Performance tracking
```

### Advanced Migration Patterns

#### Pattern 1: Combining Both Systems

Use data pipeline for data operations and byte transformers for styling:

```go
// Step 1: Data transformations
doc := output.New().
    Table("Sales Report", salesData, output.WithKeys("rep", "region", "amount")).
    Build()

// Apply data transformations
transformedDoc := doc.Pipeline().
    Filter(func(r output.Record) bool {
        return r["amount"].(float64) > 10000
    }).
    AddColumn("performance", func(r output.Record) any {
        amount := r["amount"].(float64)
        if amount > 50000 {
            return "excellent"
        } else if amount > 25000 {
            return "good"
        }
        return "average"
    }).
    SortBy("amount", output.Descending).
    Execute()

// Step 2: Style with byte transformers
out := output.NewOutput(
    output.WithFormat(output.Table),
    // Add colors based on performance values
    output.WithTransformer(&output.ColorTransformer{
        Scheme: output.ColorScheme{
            Success: "excellent",
            Warning: "good",
            Info:    "average",
        },
    }),
    output.WithWriter(output.NewStdoutWriter()),
)

// This combines the best of both systems:
// - Structured data operations (pipeline)
// - Visual styling (byte transformers)
```

#### Pattern 2: Aggregation and Reporting

**OLD: Manual Aggregation**
```go
// v1 manual aggregation
regionSums := make(map[string]float64)
regionCounts := make(map[string]int)

for _, record := range salesData {
    region := record["region"].(string)
    amount := record["amount"].(float64)
    
    regionSums[region] += amount
    regionCounts[region]++
}

var aggregatedData []map[string]any
for region, sum := range regionSums {
    aggregatedData = append(aggregatedData, map[string]any{
        "region":      region,
        "total_sales": sum,
        "avg_sales":   sum / float64(regionCounts[region]),
        "count":       regionCounts[region],
    })
}
```

**NEW: Pipeline Aggregation**
```go
// v2 pipeline aggregation
doc := output.New().
    Table("Sales", salesData, output.WithKeys("salesperson", "region", "amount")).
    Build()

aggregatedDoc := doc.Pipeline().
    GroupBy(
        []string{"region"},
        map[string]output.AggregateFunc{
            "total_sales": output.SumAggregate("amount"),
            "avg_sales":   output.AverageAggregate("amount"),
            "count":       output.CountAggregate,
            "max_sale":    output.MaxAggregate("amount"),
        },
    ).
    SortBy("total_sales", output.Descending).
    Execute()

// Benefits:
// - Built-in aggregation functions
// - Automatic schema generation
// - Error handling for type mismatches
// - Performance optimized
```

### Performance Guidance

#### Pipeline Performance Characteristics

1. **Automatic Optimization**: Operations are reordered for optimal performance
   ```go
   // Written order (potentially inefficient)
   doc.Pipeline().
       Sort("name", output.Ascending).        // Expensive operation first
       Filter(func(r output.Record) bool {    // Filter after sort
           return r["active"].(bool)
       }).
       Limit(10)

   // Automatically optimized to:
   // 1. Filter first (reduce dataset)
   // 2. Sort smaller dataset  
   // 3. Limit final results
   ```

2. **Memory Efficiency**: Uses copy-on-write and efficient cloning
3. **Type Safety**: Runtime type checking with clear error messages
4. **Resource Limits**: Built-in limits prevent runaway operations

#### When to Use Byte Transformers vs Pipeline

| Use Case | Byte Transformer | Data Pipeline | Reason |
|----------|------------------|---------------|---------|
| Filter records | ❌ | ✅ | Data operation, not text styling |
| Sort by column | ❌ | ✅ | Data operation with complex types |
| Add calculated fields | ❌ | ✅ | Requires access to structured data |
| Add ANSI colors | ✅ | ❌ | Format-specific text styling |
| Convert to emoji | ✅ | ❌ | Text replacement operation |
| Aggregate data | ❌ | ✅ | Requires grouping and calculation |
| Format numbers | ✅ | ❌ | Presentation formatting |

### Migration Checklist

**Phase 1: Identify Transformation Types**
- [ ] List all current transformers in use
- [ ] Categorize as data operations vs presentation styling
- [ ] Identify complex manual data manipulation code

**Phase 2: Migrate Data Operations**
- [ ] Replace manual filtering with Pipeline.Filter()
- [ ] Replace manual sorting with Pipeline.Sort/SortBy()
- [ ] Replace manual aggregation with Pipeline.GroupBy()
- [ ] Replace calculated fields with Pipeline.AddColumn()

**Phase 3: Optimize and Test**
- [ ] Remove redundant manual data manipulation
- [ ] Test with various data sizes
- [ ] Verify key order preservation
- [ ] Check error handling

**Phase 4: Keep Presentation Styling**
- [ ] Keep byte transformers for colors, emoji, formatting
- [ ] Combine pipeline and byte transformers where needed
- [ ] Update to format-aware transformers if needed

### Common Migration Pitfalls

#### Pitfall 1: Converting Presentation Logic
```go
// WRONG: Don't convert text styling to data pipeline
// This belongs in byte transformers, not data pipeline
doc.Pipeline().
    AddColumn("status_styled", func(r output.Record) any {
        status := r["status"].(string)
        return fmt.Sprintf("🟢 %s", status) // This is presentation!
    })

// RIGHT: Keep styling in byte transformers
// Data pipeline for data, byte transformer for styling
transformedDoc := doc.Pipeline().
    Filter(func(r output.Record) bool { return r["status"] == "active" }).
    Execute()

// Then apply styling with byte transformer
out := output.NewOutput(
    output.WithTransformer(&output.EmojiTransformer{}),
)
```

#### Pitfall 2: Over-complicating Simple Cases
```go
// WRONG: Using pipeline for single simple operation
doc.Pipeline().
    Filter(func(r output.Record) bool { return r["active"].(bool) }).
    Execute()

// BETTER: Pre-filter data if it's simple and static
activeRecords := []map[string]any{}
for _, record := range allRecords {
    if record["active"].(bool) {
        activeRecords = append(activeRecords, record)
    }
}

doc := output.New().
    Table("Active", activeRecords).
    Build()
```

### Performance Comparison

| Operation | Manual Approach | Byte Transformer | Data Pipeline |
|-----------|-----------------|------------------|---------------|
| Filter 1000 records | ~0.1ms | ~10ms (parse overhead) | ~0.2ms |
| Sort 1000 records | ~1ms | ~15ms (parse overhead) | ~1.5ms |
| Add calculated field | ~0.5ms | Not applicable | ~0.8ms |
| Complex chain | ~2ms | ~50ms (multiple parses) | ~3ms |

**Key Insights**:
- Data pipeline has minimal overhead vs manual approach
- Byte transformers have significant parse/render overhead
- Complex operation chains benefit most from pipeline optimization

### Real-World Migration Example

**Complete Before/After for a Sales Reporting System**

**BEFORE (v1 + manual operations)**:
```go
// Step 1: Manual data filtering and sorting
var qualifiedSales []map[string]any
for _, sale := range allSales {
    if sale["status"] == "completed" && sale["amount"].(float64) > 5000 {
        // Add calculated fields manually
        amount := sale["amount"].(float64)
        sale["commission"] = amount * 0.03
        sale["tier"] = determineTier(amount)
        qualifiedSales = append(qualifiedSales, sale)
    }
}

// Step 2: Manual sorting
sort.Slice(qualifiedSales, func(i, j int) bool {
    return qualifiedSales[i]["amount"].(float64) > qualifiedSales[j]["amount"].(float64)
})

// Step 3: Truncate manually
if len(qualifiedSales) > 50 {
    qualifiedSales = qualifiedSales[:50]
}

// Step 4: Create output with transformers
output := &format.OutputArray{
    Keys: []string{"salesperson", "region", "amount", "commission", "tier"},
}
output.SetKeys(qualifiedSales)

// Add styling transformers
config := format.OutputConfig{
    Format: format.OutputTable,
    Transformers: []format.Transformer{
        &format.ColorTransformer{},
    },
}
```

**AFTER (v2 with data pipeline)**:
```go
// Single integrated pipeline with automatic optimization
doc := output.New().
    Table("Sales", allSales, 
        output.WithKeys("salesperson", "region", "amount", "date", "status")).
    Build()

finalDoc := doc.Pipeline().
    // Filter qualified sales
    Filter(func(r output.Record) bool {
        return r["status"] == "completed" && r["amount"].(float64) > 5000
    }).
    // Add calculated fields
    AddColumn("commission", func(r output.Record) any {
        return r["amount"].(float64) * 0.03
    }).
    AddColumn("tier", func(r output.Record) any {
        return determineTier(r["amount"].(float64))
    }).
    // Sort by amount (highest first)
    SortBy("amount", output.Descending).
    // Get top 50
    Limit(50).
    Execute()

// Output with styling
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformer(&output.ColorTransformer{}),
    output.WithWriter(output.NewStdoutWriter()),
)

// Benefits achieved:
// ✅ 50% less code
// ✅ Type-safe operations
// ✅ Automatic optimization
// ✅ Better error handling
// ✅ Built-in performance tracking
// ✅ Immutable transformations
// ✅ Format-agnostic operations
```

## Need Help?

- Check the [API documentation](https://pkg.go.dev/github.com/ArjenSchwarz/go-output/v2)
- Review the [examples](https://github.com/ArjenSchwarz/go-output/tree/main/v2/examples)
- See [collapsible examples](https://github.com/ArjenSchwarz/go-output/tree/main/v2/examples/collapsible_*)
- Report issues at [GitHub Issues](https://github.com/ArjenSchwarz/go-output/issues)

The migration tool can handle most common patterns automatically. For complex migrations, refer to this guide and the API documentation.