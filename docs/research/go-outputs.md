# go-output Library Documentation

A comprehensive Go library for outputting structured data in multiple formats. This library provides a unified interface to convert your data into JSON, YAML, CSV, HTML, tables, markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Supported Output Formats](#supported-output-formats)
- [Configuration Options](#configuration-options)
- [Advanced Features](#advanced-features)
- [Integration Examples](#integration-examples)
- [API Reference](#api-reference)

## Installation

```bash
go get github.com/ArjenSchwarz/go-output
```

## Quick Start

Here's a simple example to get you started:

```go
package main

import (
    "github.com/ArjenSchwarz/go-output"
)

func main() {
    // Create output settings
    settings := format.NewOutputSettings()
    settings.SetOutputFormat("json")

    // Create output array
    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"Name", "Age", "City"},
    }

    // Add data
    output.AddContents(map[string]interface{}{
        "Name": "Alice",
        "Age":  30,
        "City": "New York",
    })

    output.AddContents(map[string]interface{}{
        "Name": "Bob",
        "Age":  25,
        "City": "London",
    })

    // Output the data
    output.Write()
}
```

This will output:
```json
[{"Age":30,"City":"New York","Name":"Alice"},{"Age":25,"City":"London","Name":"Bob"}]
```

## Examples

For complete working examples demonstrating all features, see the [examples directory](examples/). It includes:

- Basic usage for all output formats
- Advanced configuration options
- Multi-section reports
- Mermaid diagram generation
- File output examples

Run the examples with:
```bash
cd examples
go run basic_usage.go
```

## Core Concepts

### OutputArray

The `OutputArray` is the primary structure that holds your data and configuration:

```go
type OutputArray struct {
    Settings *OutputSettings  // Configuration options
    Contents []OutputHolder   // Your data
    Keys     []string        // Column headers/field names
}
```

### OutputHolder

Individual data containers that hold key-value pairs:

```go
type OutputHolder struct {
    Contents map[string]interface{}
}
```

### OutputSettings

Configuration object that controls output behavior:

```go
settings := format.NewOutputSettings()
settings.SetOutputFormat("table")
settings.Title = "My Data Report"
settings.UseColors = true
```

## Supported Output Formats

### JSON Format

Default format that outputs standard JSON.

```go
settings.SetOutputFormat("json")
```

**Example Output:**
```json
[{"Name":"Alice","Age":30},{"Name":"Bob","Age":25}]
```

### YAML Format

YAML formatted output for configuration files and documentation.

```go
settings.SetOutputFormat("yaml")
```

**Example Output:**
```yaml
- Age: 30
  Name: Alice
- Age: 25
  Name: Bob
```

### CSV Format

Comma-separated values for spreadsheet applications.

```go
settings.SetOutputFormat("csv")
```

**Example Output:**
```
Name,Age
Alice,30
Bob,25
```

### HTML Format

Full HTML pages with built-in styling and optional table of contents.

```go
settings.SetOutputFormat("html")
settings.Title = "Employee Report"
settings.HasTOC = true
```

**Features:**
- Responsive table styling
- Optional table of contents
- Custom titles
- Section headers
- File output support

### Table Format

Console-friendly table output with various styling options.

```go
settings.SetOutputFormat("table")
settings.TableStyle = format.TableStyles["ColoredBright"]
settings.TableMaxColumnWidth = 30
```

**Available Styles:**
- `Default`
- `Bold`
- `ColoredBright`
- `ColoredDark`
- Various themed color combinations

### Markdown Format

GitHub-flavored markdown with table support.

```go
settings.SetOutputFormat("markdown")
settings.Title = "Data Report"
settings.HasTOC = true
settings.FrontMatter = map[string]string{
    "title": "My Report",
    "date": "2024-01-01",
}
```

**Features:**
- Front matter support
- Table of contents generation
- Section headers
- Proper markdown table formatting

### DOT Format

GraphViz DOT format for creating graphs and diagrams.

```go
settings.SetOutputFormat("dot")
settings.AddFromToColumns("Source", "Target")
```

**Requirements:**
- Must specify source and target columns using `AddFromToColumns()`
- Data should represent relationships/connections

**Example Usage:**
```go
output.AddContents(map[string]interface{}{
    "Source": "NodeA",
    "Target": "NodeB",
})
```

### Mermaid Format

Create Mermaid diagrams including flowcharts, pie charts, and Gantt charts.

#### Flowcharts

```go
settings.SetOutputFormat("mermaid")
settings.MermaidSettings.ChartType = "flowchart"
settings.AddFromToColumns("From", "To")
```

#### Pie Charts

```go
settings.SetOutputFormat("mermaid")
settings.MermaidSettings.ChartType = "piechart"
settings.AddFromToColumns("Label", "Value")

output.AddContents(map[string]interface{}{
    "Label": "Apples",
    "Value": 42.5,
})
```

#### Gantt Charts

```go
settings.SetOutputFormat("mermaid")
settings.MermaidSettings.ChartType = "ganttchart"
settings.MermaidSettings.GanttSettings = &mermaid.GanttSettings{
    LabelColumn:     "Task",
    StartDateColumn: "Start",
    DurationColumn:  "Duration",
    StatusColumn:    "Status",
}

output.AddContents(map[string]interface{}{
    "Task":     "Design Phase",
    "Start":    "2024-01-01",
    "Duration": "5d",
    "Status":   "done",
})
```

### Draw.io Format

Export data as CSV files that can be imported into Draw.io/Diagrams.net.

```go
settings.SetOutputFormat("drawio")
settings.DrawIOHeader = drawio.Header{
    // Configure Draw.io import settings
}
settings.OutputFile = "diagram_data.csv"
```

## Configuration Options

### OutputSettings Fields

```go
type OutputSettings struct {
    // Display options
    Title          string            // Title for the output
    UseColors      bool             // Enable colored output
    UseEmoji       bool             // Use emoji for boolean values
    HasTOC         bool             // Generate table of contents

    // File output
    OutputFile       string          // Output file path
    OutputFileFormat string          // Format for file output
    ShouldAppend     bool           // Append to existing file

    // Table configuration
    TableStyle            table.Style  // Table styling
    TableMaxColumnWidth   int         // Maximum column width
    SeparateTables       bool         // Add spacing between tables

    // Data processing
    SortKey        string            // Field to sort by
    SplitLines     bool             // Split multi-value fields

    // Format-specific settings
    FromToColumns    *FromToColumns   // For graph formats
    MermaidSettings  *mermaid.Settings // For Mermaid diagrams
    DrawIOHeader     drawio.Header    // For Draw.io export
    FrontMatter      map[string]string // For Markdown

    // Cloud storage
    S3Bucket         S3Output         // S3 bucket configuration
}
```

### Common Configuration Patterns

#### Basic Setup
```go
settings := format.NewOutputSettings()
settings.SetOutputFormat("table")
settings.Title = "My Report"
settings.UseColors = true
```

#### File Output
```go
settings.OutputFile = "report.html"
settings.OutputFileFormat = "html" // Can differ from display format
```

#### Sorting and Processing
```go
settings.SortKey = "Name"          // Sort by Name field
settings.SplitLines = true         // Split comma-separated values
```

#### S3 Output
```go
s3Client := // ... initialize AWS S3 client
settings.SetS3Bucket(s3Client, "my-bucket", "reports/output.json")
```

## Advanced Features

### Multiple Sections with Headers

Create reports with multiple sections:

```go
// First section
output.AddHeader("Active Users")
output.AddContents(map[string]interface{}{
    "Name": "Alice",
    "Status": "Active",
})
output.AddToBuffer()

// Second section
output.AddHeader("Inactive Users")
output.AddContents(map[string]interface{}{
    "Name": "Bob",
    "Status": "Inactive",
})
output.AddToBuffer()

// Generate final output
output.Write()
```

### Custom Data Conversion

The library automatically converts data types:

```go
// Boolean values
output.AddContents(map[string]interface{}{
    "Name": "Alice",
    "Active": true,        // Becomes "Yes" or "✅"
    "Verified": false,     // Becomes "No" or "❌"
})

// Arrays
output.AddContents(map[string]interface{}{
    "Name": "Bob",
    "Skills": []string{"Go", "Python", "Docker"}, // Joins with separator
})

// Numbers
output.AddContents(map[string]interface{}{
    "Name": "Charlie",
    "Age": 30,            // Converted to string
    "Score": 95.5,        // Converted to string
})
```

### Working with Raw Data

Get data in different formats:

```go
// Get as string maps (converted)
stringMaps := output.GetContentsMap()

// Get as interface maps (raw)
rawMaps := output.GetContentsMapRaw()
```

### Color and Emoji Support

```go
settings.UseColors = true  // Enable terminal colors
settings.UseEmoji = true   // Use ✅/❌ instead of Yes/No
```

## Integration Examples

### CLI Application Integration

```go
package main

import (
    "flag"
    "github.com/ArjenSchwarz/go-output"
)

func main() {
    var outputFormat = flag.String("output", "table", "Output format")
    var outputFile = flag.String("file", "", "Output file")
    flag.Parse()

    // Setup
    settings := format.NewOutputSettings()
    settings.SetOutputFormat(*outputFormat)
    if *outputFile != "" {
        settings.OutputFile = *outputFile
    }

    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"ID", "Name", "Status"},
    }

    // Your data collection logic
    data := collectData()
    for _, item := range data {
        output.AddContents(map[string]interface{}{
            "ID":     item.ID,
            "Name":   item.Name,
            "Status": item.Status,
        })
    }

    output.Write()
}
```

### Web Service Integration

```go
func generateReport(w http.ResponseWriter, r *http.Request) {
    format := r.URL.Query().Get("format")
    if format == "" {
        format = "json"
    }

    settings := format.NewOutputSettings()
    settings.SetOutputFormat(format)

    output := format.OutputArray{
        Settings: settings,
        Keys:     []string{"ID", "Name", "Email"},
    }

    // Add your data...

    // For web output, capture the bytes
    var buf bytes.Buffer
    // Temporarily redirect output
    originalStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    output.Write()

    w.Close()
    os.Stdout = originalStdout
    io.Copy(&buf, r)

    // Set appropriate content type
    switch format {
    case "json":
        w.Header().Set("Content-Type", "application/json")
    case "html":
        w.Header().Set("Content-Type", "text/html")
    case "csv":
        w.Header().Set("Content-Type", "text/csv")
    }

    w.Write(buf.Bytes())
}
```

### Database Query Results

```go
func outputQueryResults(rows *sql.Rows, format string) {
    columns, _ := rows.Columns()

    settings := format.NewOutputSettings()
    settings.SetOutputFormat(format)
    settings.SortKey = columns[0] // Sort by first column

    output := format.OutputArray{
        Settings: settings,
        Keys:     columns,
    }

    for rows.Next() {
        values := make([]interface{}, len(columns))
        valuePtrs := make([]interface{}, len(columns))
        for i := range columns {
            valuePtrs[i] = &values[i]
        }

        rows.Scan(valuePtrs...)

        record := make(map[string]interface{})
        for i, col := range columns {
            record[col] = values[i]
        }

        output.AddContents(record)
    }

    output.Write()
}
```

## API Reference

### OutputArray Methods

#### `AddHolder(holder OutputHolder)`
Adds an OutputHolder to the array with automatic sorting if SortKey is set.

#### `AddContents(contents map[string]interface{})`
Convenience method to add data as a map.

#### `AddHeader(header string)`
Adds a section header (format-dependent styling).

#### `AddToBuffer()`
Adds current contents to internal buffer for multi-section output.

#### `Write()`
Outputs the final result to stdout or configured file/S3.

#### `GetContentsMap() []map[string]string`
Returns data as string maps (all values converted to strings).

#### `GetContentsMapRaw() []map[string]interface{}`
Returns raw data without type conversion.

#### `HtmlTableOnly() []byte`
Returns just the HTML table portion without full page structure.

### OutputSettings Methods

#### `NewOutputSettings() *OutputSettings`
Creates a new OutputSettings with sensible defaults.

#### `SetOutputFormat(format string)`
Sets the output format (case-insensitive).

#### `AddFromToColumns(from, to string)`
Sets source and target columns for graph formats.

#### `AddFromToColumnsWithLabel(from, to, label string)`
Sets source, target, and label columns for graph formats.

#### `SetS3Bucket(client *s3.Client, bucket, path string)`
Configures S3 output destination.

#### `GetDefaultExtension() string`
Returns appropriate file extension for the current format.

#### `NeedsFromToColumns() bool`
Returns true if the format requires from/to column configuration.

### Utility Functions

#### `PrintByteSlice(contents []byte, outputFile string, targetBucket S3Output) error`
Low-level function to output byte data to file, stdout, or S3.

### Error Handling

The library uses `log.Fatal()` for critical errors such as:
- Missing required configuration for specific formats
- File I/O errors
- S3 upload failures

For production use, consider wrapping library calls in error recovery:

```go
func safeOutput(output format.OutputArray) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Output error: %v", r)
            // Handle error appropriately
        }
    }()

    output.Write()
}
```

## Best Practices

1. **Always set Keys**: Define your column headers/field names explicitly
2. **Use consistent data types**: Keep field types consistent across records
3. **Configure before adding data**: Set up OutputSettings before adding content
4. **Handle file permissions**: Ensure write permissions for file output
5. **Test different formats**: Verify output in your target formats during development
6. **Use sorting**: Set SortKey for consistent output ordering
7. **Validate graph data**: Ensure from/to columns exist for graph formats
8. **Consider memory usage**: For large datasets, process in chunks

## Dependencies

The library depends on several external packages:
- `github.com/jedib0t/go-pretty/v6` - Table formatting
- `github.com/emicklei/dot` - DOT graph generation
- `github.com/aws/aws-sdk-go-v2/service/s3` - S3 integration
- `gopkg.in/yaml.v3` - YAML output
- `github.com/gosimple/slug` - URL-safe string generation
- `github.com/fatih/color` - Terminal colors

## License

This library is provided as-is. Check the repository for specific license terms.

---

This documentation covers the complete functionality of the go-output library. For additional examples or specific use cases, refer to the test files in the repository or create issues for clarification.
