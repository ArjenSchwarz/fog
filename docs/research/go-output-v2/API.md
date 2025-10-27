# Go-Output v2 API Documentation

## Overview

Go-Output v2 is a complete redesign of the library providing thread-safe document generation with preserved key ordering and multiple output formats. This API documentation covers all public interfaces and methods.

**Version**: v2.4.0
**Go Version**: 1.24+
**Import Path**: `github.com/ArjenSchwarz/go-output/v2`

## Agent Implementation Guide

This documentation is optimized for AI agents implementing the library. Key patterns are highlighted with clear examples and common pitfalls are documented.

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Create a document using the builder pattern
    doc := output.New().
        Table("Users", []map[string]any{
            {"Name": "Alice", "Age": 30, "Status": "Active"},
            {"Name": "Bob", "Age": 25, "Status": "Inactive"},
        }, output.WithKeys("Name", "Age", "Status")).
        Text("This is additional text content").
        Build()

    // Create output with multiple formats
    out := output.NewOutput(
        output.WithFormats(output.Table, output.JSON),
        output.WithWriter(output.NewStdoutWriter()),
    )

    // Render the document
    if err := out.Render(context.Background(), doc); err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Agent Implementation Patterns

### Critical Implementation Rules

1. **ALWAYS preserve key order**: Use `WithKeys()` or `WithSchema()` - never rely on map iteration order
2. **NEVER modify global state**: All operations must use the Builder pattern
3. **ALWAYS check for errors**: Use `HasErrors()` and `Errors()` on Builder before rendering
4. **Thread safety is guaranteed**: All public methods are thread-safe

### Common Implementation Tasks

#### Task: Display tabular data with specific column order

```go
// CORRECT: Explicit key ordering
doc := output.New().
    Table("Results", data, output.WithKeys("ID", "Name", "Score", "Status")).
    Build()

// INCORRECT: Relies on map iteration (undefined order)
doc := output.New().
    Table("Results", data). // No WithKeys - order will be random!
    Build()
```

#### Task: Create expandable error details in tables

```go
// Use built-in error formatter for arrays of errors
schema := output.WithSchema(
    output.Field{
        Name: "file",
        Type: "string",
    },
    output.Field{
        Name: "errors",
        Type: "array",
        Formatter: output.ErrorListFormatter(
            output.WithCollapsibleExpanded(false), // Collapsed by default
            output.WithCodeFences("text"), // v2.1.1+: Add code highlighting
        ),
    },
)

doc := output.New().
    Table("Validation Results", errorData, schema).
    Build()
```

#### Task: Generate multiple output formats

```go
// Single document, multiple formats
doc := output.New().
    Table("Data", data, output.WithKeys("Name", "Value")).
    Build()

// Render to multiple formats and destinations
fileWriter, _ := output.NewFileWriter("./reports", "output.{format}")
out := output.NewOutput(
    output.WithFormats(output.JSON, output.CSV, output.Markdown),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithWriter(fileWriter),
)

err := out.Render(context.Background(), doc)
```

#### Task: Create hierarchical document structure

```go
doc := output.New().
    Header("Analysis Report").
    Section("Overview", func(b *output.Builder) {
        b.Text("System health: OK")
        b.Table("Metrics", metrics, output.WithKeys("Metric", "Value"))
    }).
    Section("Details", func(b *output.Builder) {
        b.Table("Performance", perfData, output.WithKeys("Component", "Time"))
        b.Table("Errors", errorData, output.WithKeys("Error", "Count"))
    }).
    Build()
```

### Error Handling Patterns

```go
// Always check builder errors before rendering
builder := output.New().
    Table("Data", invalidData, output.WithKeys("Key"))
    
if builder.HasErrors() {
    for _, err := range builder.Errors() {
        // Log or handle each error
        fmt.Printf("Build error: %v\n", err)
    }
    return // Don't attempt to render
}

doc := builder.Build()
```

### Memory-Efficient Patterns

```go
// For large datasets, use streaming-capable formats
largeData := generateLargeDataset() // Millions of rows

// Good: Streaming formats
out := output.NewOutput(
    output.WithFormats(output.CSV, output.JSON), // Both support streaming
    output.WithWriter(output.NewFileWriter(".", "large.{format}")),
)

// Avoid: Non-streaming formats for large data
// DON'T use Mermaid, DrawIO, or DOT for large datasets
```

## Core Concepts

### Document-Builder Pattern

The v2 API eliminates global state by using an immutable Document-Builder pattern:

- **Document**: Immutable container for content and metadata
- **Builder**: Fluent API for constructing documents with thread-safe operations
- **Content**: Interface implemented by all content types

### Key Order Preservation

A fundamental feature that preserves exact user-specified column ordering:

- Key order is **never** alphabetized or reordered
- Each table maintains its own independent key ordering
- Supports multiple tables with different key sets

## Public API Reference

### Core Types

#### Document

Represents an immutable collection of content to be rendered.

```go
type Document struct {
    // Internal fields (not exported)
}

// GetContents returns a copy of the document's contents
func (d *Document) GetContents() []Content

// GetMetadata returns a copy of the document's metadata
func (d *Document) GetMetadata() map[string]any
```

**Thread Safety**: All methods are thread-safe using RWMutex.

#### Builder

Constructs documents using a fluent API pattern.

```go
type Builder struct {
    // Internal fields (not exported)
}

// New creates a new document builder
func New() *Builder

// Build finalizes and returns the document
func (b *Builder) Build() *Document

// HasErrors returns true if any errors occurred during building
func (b *Builder) HasErrors() bool

// Errors returns all errors that occurred during building
func (b *Builder) Errors() []error

// SetMetadata sets a metadata key-value pair
func (b *Builder) SetMetadata(key string, value any) *Builder
```

**Thread Safety**: All methods are thread-safe using Mutex.

### Content Types

#### Content Interface

All content types implement this interface:

```go
type Content interface {
    // Type returns the content type
    Type() ContentType

    // ID returns a unique identifier for this content
    ID() string

    // Encoding interfaces for efficient serialization
    encoding.TextAppender
    encoding.BinaryAppender
}
```

#### ContentType

Defines the type of content:

```go
type ContentType int

const (
    ContentTypeTable   ContentType = iota // Tabular data
    ContentTypeText                       // Unstructured text
    ContentTypeRaw                        // Format-specific content
    ContentTypeSection                    // Grouped content
)

// String returns the string representation
func (ct ContentType) String() string
```

#### TableContent

Represents tabular data with preserved key ordering:

```go
// NewTableContent creates a new table content
func NewTableContent(title string, data any, opts ...TableOption) (*TableContent, error)

// Methods
func (t *TableContent) Type() ContentType
func (t *TableContent) ID() string
func (t *TableContent) Title() string
func (t *TableContent) Schema() *Schema
func (t *TableContent) Records() []Record
```

**Key Features**:
- Preserves exact key order as specified by user
- Supports various data types ([]map[string]any, []Record, etc.)
- Thread-safe read operations

#### TextContent

Represents unstructured text with styling:

```go
// NewTextContent creates a new text content
func NewTextContent(text string, opts ...TextOption) *TextContent

// Methods
func (t *TextContent) Type() ContentType
func (t *TextContent) ID() string
func (t *TextContent) Text() string
func (t *TextContent) Style() TextStyle
```

#### RawContent

Represents format-specific content:

```go
// NewRawContent creates a new raw content
func NewRawContent(format string, data []byte, opts ...RawOption) (*RawContent, error)

// Methods
func (r *RawContent) Type() ContentType
func (r *RawContent) ID() string
func (r *RawContent) Format() string
func (r *RawContent) Data() []byte
```

#### SectionContent

Represents grouped content with hierarchical structure:

```go
// NewSectionContent creates a new section content
func NewSectionContent(title string, opts ...SectionOption) *SectionContent

// Methods
func (s *SectionContent) Type() ContentType
func (s *SectionContent) ID() string
func (s *SectionContent) Title() string
func (s *SectionContent) Level() int
func (s *SectionContent) Contents() []Content
func (s *SectionContent) AddContent(content Content)
```

### Builder Methods

#### Table Creation

```go
// Table adds a table with preserved key ordering
func (b *Builder) Table(title string, data any, opts ...TableOption) *Builder
```

**Table Options**:
- `WithKeys(keys ...string)` - Explicit key ordering (recommended)
- `WithSchema(fields ...Field)` - Full schema with formatters
- `WithAutoSchema()` - Auto-detect schema from data

**Example**:
```go
doc := output.New().
    Table("Users", userData, output.WithKeys("Name", "Email", "Status")).
    Table("Orders", orderData, output.WithKeys("ID", "Date", "Amount")).
    Build()
```

#### Text Content

```go
// Text adds text content with optional styling
func (b *Builder) Text(text string, opts ...TextOption) *Builder

// Header adds a header text (v1 compatibility)
func (b *Builder) Header(text string) *Builder
```

**Text Options**:
- `WithBold(bold bool)` - Bold text
- `WithItalic(italic bool)` - Italic text
- `WithColor(color string)` - Text color
- `WithHeader(header bool)` - Header styling

#### Raw Content

```go
// Raw adds format-specific raw content
func (b *Builder) Raw(format string, data []byte, opts ...RawOption) *Builder
```

**Supported Formats**: `html`, `css`, `js`, `json`, `xml`, `yaml`, `markdown`, `text`, `csv`, `dot`, `mermaid`, `drawio`, `svg`

#### Section Grouping

```go
// Section groups content under a heading
func (b *Builder) Section(title string, fn func(*Builder), opts ...SectionOption) *Builder
```

**Example**:
```go
doc := output.New().
    Section("User Data", func(b *output.Builder) {
        b.Table("Active Users", activeUsers, output.WithKeys("Name", "Email"))
        b.Table("Inactive Users", inactiveUsers, output.WithKeys("Name", "LastLogin"))
    }).
    Build()
```

#### Graph and Chart Methods

```go
// Graph adds graph content with edges
func (b *Builder) Graph(title string, edges []Edge) *Builder

// Chart adds a generic chart content
func (b *Builder) Chart(title, chartType string, data any) *Builder

// GanttChart adds a Gantt chart with tasks
func (b *Builder) GanttChart(title string, tasks []GanttTask) *Builder

// PieChart adds a pie chart with slices
func (b *Builder) PieChart(title string, slices []PieSlice, showData bool) *Builder

// DrawIO adds Draw.io diagram content
func (b *Builder) DrawIO(title string, records []Record, header DrawIOHeader) *Builder
```

### Schema System

#### Schema

Defines table structure with key ordering:

```go
type Schema struct {
    // Internal fields (not exported)
}

// NewSchemaFromKeys creates a schema from key names
func NewSchemaFromKeys(keys []string) *Schema

// DetectSchemaFromData auto-detects schema from data
func DetectSchemaFromData(data any) *Schema

// Methods
func (s *Schema) GetKeyOrder() []string
func (s *Schema) SetKeyOrder(keys []string)
func (s *Schema) FindField(name string) *Field
func (s *Schema) AddField(field Field)
func (s *Schema) GetFields() []Field
```

#### Field

Defines individual table columns:

```go
type Field struct {
    Name      string                    // Field name
    Type      string                    // Data type hint
    Formatter func(any) any           // Custom formatter (can return CollapsibleValue)
    Hidden    bool                      // Hide from output
}
```

### Collapsible Content System (v2.1.0+)

The v2 library provides comprehensive support for collapsible content that adapts to each output format, enabling summary/detail views for complex data.

#### Code Fence Support (v2.1.1+)

**New Feature**: Wrap collapsible details in syntax-highlighted code blocks for better readability.

```go
// WithCodeFences adds language-specific syntax highlighting
func WithCodeFences(language string) CollapsibleOption

// WithoutCodeFences explicitly disables code fence wrapping  
func WithoutCodeFences() CollapsibleOption
```

**Usage Example**:
```go
// JSON configuration with syntax highlighting
configValue := output.NewCollapsibleValue(
    "Configuration",
    jsonConfig,
    output.WithCollapsibleExpanded(false),
    output.WithCodeFences("json"), // Syntax highlight as JSON
)

// Error logs with code highlighting
errorValue := output.NewCollapsibleValue(
    "Error Stack Trace",
    stackTrace,
    output.WithCodeFences("bash"), // Highlight as bash/terminal output
)

// API response with YAML highlighting
apiValue := output.NewCollapsibleValue(
    "API Response",
    yamlResponse,
    output.WithCodeFences("yaml"),
)
```

**Format Behavior**:
- **HTML**: Uses `<pre><code class="language-{lang}">` for GitHub/GitLab compatibility
- **Markdown**: Uses triple-backtick code fences with language identifier
- **Other formats**: Preserves content without HTML escaping in code blocks

#### Enhanced Markdown Escaping (v2.1.3)

**Improvements**: Better markdown table cell escaping to prevent formatting issues.

**Escaped Characters in Table Cells**:
- Pipes (`|`) → `\|` - Prevents breaking table structure
- Asterisks (`*`) → `\*` - Prevents unintended bold/italic
- Underscores (`_`) → `\_` - Prevents unintended emphasis
- Backticks (`` ` ``) → `\`` - Prevents code formatting
- Square brackets (`[`) → `\[` - Prevents link interpretation
- Newlines → `<br>` - Maintains table cell integrity

**Agent Implementation Note**: When generating markdown tables with user content, the library automatically handles escaping. Do not pre-escape content as it may result in double-escaping.

#### CollapsibleValue Interface

Core interface for creating expandable content in table cells:

```go
type CollapsibleValue interface {
    Summary() string                              // Collapsed view text
    Details() any                                 // Expanded content (any type)
    IsExpanded() bool                            // Default expansion state
    FormatHint(format string) map[string]any     // Format-specific rendering hints
}
```

**Usage**: Field formatters can return CollapsibleValue instances to create expandable content.

#### DefaultCollapsibleValue

Standard implementation with configuration options:

```go
type DefaultCollapsibleValue struct {
    // Internal fields (not exported)
}

// NewCollapsibleValue creates a collapsible value with options
func NewCollapsibleValue(summary string, details any, opts ...CollapsibleOption) *DefaultCollapsibleValue

// Configuration options
func WithCollapsibleExpanded(expanded bool) CollapsibleOption
func WithMaxLength(length int) CollapsibleOption
func WithFormatHint(format string, hints map[string]any) CollapsibleOption
```

**Example**:
```go
// Create collapsible error list
errorValue := output.NewCollapsibleValue(
    "3 errors found",
    []string{"Missing import", "Unused variable", "Type error"},
    output.WithCollapsibleExpanded(false),
    output.WithMaxLength(200),
)
```

#### Built-in Collapsible Formatters

Pre-built formatters for common patterns:

```go
// Error list formatter - collapses arrays of strings/errors
func ErrorListFormatter(opts ...CollapsibleOption) func(any) any

// File path formatter - shortens long paths with expandable details
func FilePathFormatter(maxLength int, opts ...CollapsibleOption) func(any) any

// JSON formatter - collapses large JSON objects
func JSONFormatter(maxLength int, opts ...CollapsibleOption) func(any) any

// Custom collapsible formatter
func CollapsibleFormatter(summaryTemplate string, detailFunc func(any) any, opts ...CollapsibleOption) func(any) any
```

**Usage Example**:
```go
schema := output.WithSchema(
    output.Field{
        Name: "errors",
        Type: "array",
        Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false)),
    },
    output.Field{
        Name: "path",
        Type: "string", 
        Formatter: output.FilePathFormatter(30),
    },
    output.Field{
        Name: "config",
        Type: "object",
        Formatter: output.JSONFormatter(100),
    },
)
```

#### CollapsibleSection Interface

Interface for section-level collapsible content:

```go
type CollapsibleSection interface {
    Title() string                               // Section title/summary
    Content() []Content                          // Nested content items
    IsExpanded() bool                           // Default expansion state
    Level() int                                 // Nesting level (0-3)
    FormatHint(format string) map[string]any    // Format-specific hints
}
```

**Usage**: Create collapsible sections containing entire tables or content blocks.

#### DefaultCollapsibleSection

Standard implementation for collapsible sections:

```go
type DefaultCollapsibleSection struct {
    // Internal fields (not exported)
}

// NewCollapsibleSection creates a collapsible section
func NewCollapsibleSection(title string, content []Content, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection

// Helper constructors
func NewCollapsibleTable(title string, table *TableContent, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection
func NewCollapsibleReport(title string, content []Content, opts ...CollapsibleSectionOption) *DefaultCollapsibleSection

// Configuration options
func WithSectionExpanded(expanded bool) CollapsibleSectionOption
func WithSectionLevel(level int) CollapsibleSectionOption
func WithSectionFormatHint(format string, hints map[string]any) CollapsibleSectionOption
```

**Example**:
```go
// Create collapsible table section
analysisTable := output.NewTableContent("Analysis Results", data)
section := output.NewCollapsibleTable(
    "Detailed Code Analysis",
    analysisTable,
    output.WithSectionExpanded(false),
)

// Create multi-content section
reportSection := output.NewCollapsibleReport(
    "Performance Report",
    []output.Content{
        output.NewTextContent("Analysis complete"),
        analysisTable,
        output.NewTextContent("All systems operational"),
    },
    output.WithSectionExpanded(true),
)
```

#### Cross-Format Rendering

Collapsible content adapts automatically to each output format:

| Format   | CollapsibleValue Rendering | CollapsibleSection Rendering |
|----------|----------------------------|------------------------------|
| Markdown | `<details><summary>` HTML elements | Nested `<details>` structure |
| JSON     | `{"type": "collapsible", "summary": "...", "details": [...]}` | Structured data with content array |
| YAML     | YAML mapping with summary/details fields | YAML structure with nested content |
| HTML     | Semantic `<details>` with CSS classes | Section elements with collapsible behavior |
| Table    | Summary + expansion indicator | Section headers with indented content |
| CSV      | Summary + automatic detail columns | Metadata comments with table data |

#### Renderer Configuration

Control collapsible behavior globally per renderer:

```go
type CollapsibleConfig struct {
    GlobalExpansion      bool              // Override all IsExpanded() settings
    MaxDetailLength      int               // Character limit for details (default: 500)
    TruncateIndicator    string            // Truncation suffix (default: "[...truncated]")
    TableHiddenIndicator string            // Table collapse indicator
    HTMLCSSClasses       map[string]string // Custom CSS classes for HTML
}

// Apply configuration to renderers
tableOutput := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        GlobalExpansion:      false,
        TableHiddenIndicator: "[click to expand]",
        MaxDetailLength:      200,
    }),
)

htmlOutput := output.NewOutput(
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

#### Complete Example

```go
package main

import (
    "context"
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Data with complex nested information
    analysisData := []map[string]any{
        {
            "file": "/very/long/path/to/project/src/components/UserProfile.tsx",
            "errors": []string{
                "Missing import for React",
                "Unused variable 'userData'",
                "Type annotation missing for 'props'",
            },
            "config": map[string]any{
                "eslint": true,
                "typescript": true,
                "prettier": false,
                "rules": []string{"no-unused-vars", "explicit-return-type"},
            },
        },
    }
    
    // Create table with collapsible formatters
    table := output.NewTableContent("Code Analysis", analysisData,
        output.WithSchema(
            output.Field{
                Name: "file",
                Type: "string",
                Formatter: output.FilePathFormatter(25), // Shorten long paths
            },
            output.Field{
                Name: "errors", 
                Type: "array",
                Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false)),
            },
            output.Field{
                Name: "config",
                Type: "object",
                Formatter: output.JSONFormatter(50, output.WithCollapsibleExpanded(false)),
            },
        ))
    
    // Wrap in collapsible section
    section := output.NewCollapsibleTable(
        "Detailed Analysis Results",
        table,
        output.WithSectionExpanded(false),
    )
    
    // Build document
    doc := output.New().
        Header("Project Analysis Report").
        Text("Analysis completed successfully. Click sections to expand details.").
        Add(section).
        Build()
    
    // Render with custom configuration
    out := output.NewOutput(
        output.WithFormats(output.Markdown, output.JSON, output.Table),
        output.WithWriter(output.NewStdoutWriter()),
        output.WithCollapsibleConfig(output.CollapsibleConfig{
            TableHiddenIndicator: "[expand for details]",
            MaxDetailLength:      100,
        }),
    )
    
    if err := out.Render(context.Background(), doc); err != nil {
        panic(err)
    }
}
```

### Output System

#### Output

Manages rendering and writing:

```go
type Output struct {
    // Internal fields (not exported)
}

// NewOutput creates a new Output instance
func NewOutput(opts ...OutputOption) *Output

// Render processes a document through all configured components
func (o *Output) Render(ctx context.Context, doc *Document) error

// RenderTo is a convenience method using background context
func (o *Output) RenderTo(doc *Document) error

// Close cleans up resources
func (o *Output) Close() error
```

#### Output Options

Configuration options for Output:

```go
// Format options
func WithFormat(format Format) OutputOption
func WithFormats(formats ...Format) OutputOption

// Writer options
func WithWriter(writer Writer) OutputOption
func WithWriters(writers ...Writer) OutputOption

// Transformer options
func WithTransformer(transformer Transformer) OutputOption
func WithTransformers(transformers ...Transformer) OutputOption

// Progress options
func WithProgress(progress Progress) OutputOption

// v1 compatibility options
func WithTableStyle(style string) OutputOption
func WithTOC(enabled bool) OutputOption
func WithFrontMatter(fm map[string]string) OutputOption
func WithMetadata(key string, value any) OutputOption
```

### Format System

#### Format

Represents an output format:

```go
type Format struct {
    Name     string           // Format name
    Renderer Renderer         // Renderer implementation
    Options  map[string]any   // Format-specific options
}
```

#### Built-in Formats

Pre-configured formats:

```go
var (
    JSON     Format  // JSON output
    YAML     Format  // YAML output
    CSV      Format  // CSV output
    HTML     Format  // HTML output
    Table    Format  // Table output
    Markdown Format  // Markdown output
    DOT      Format  // Graphviz DOT output
    Mermaid  Format  // Mermaid diagram output
    DrawIO   Format  // Draw.io CSV output
)

// Table style variants
var (
    TableDefault       Format
    TableBold          Format
    TableColoredBright Format
    TableLight         Format
    TableRounded       Format
)
```

#### Format Constructors

```go
// TableWithStyle creates a table format with specified style
func TableWithStyle(styleName string) Format

// TableWithMaxColumnWidth creates a table format with max column width limit
func TableWithMaxColumnWidth(maxColumnWidth int) Format

// TableWithStyleAndMaxColumnWidth creates a table format with both style and max column width
func TableWithStyleAndMaxColumnWidth(styleName string, maxColumnWidth int) Format

// MarkdownWithToC creates markdown with table of contents
func MarkdownWithToC(enabled bool) Format

// MarkdownWithFrontMatter creates markdown with front matter
func MarkdownWithFrontMatter(frontMatter map[string]string) Format

// MarkdownWithOptions creates markdown with ToC and front matter
func MarkdownWithOptions(includeToC bool, frontMatter map[string]string) Format
```

**Table Max Column Width**:

Control the maximum width of table columns to prevent excessively wide output. When content exceeds the specified width, the go-pretty library will automatically wrap text within cells.

```go
// Basic max width configuration
format := output.TableWithMaxColumnWidth(50)
out := output.NewOutput(
    output.WithFormat(format),
    output.WithWriter(output.NewStdoutWriter()),
)

// Combine custom style with max width
format := output.TableWithStyleAndMaxColumnWidth("Bold", 40)

// Create directly with renderer constructors
renderer := output.NewTableRendererWithStyleAndWidth("Default", 60)
```

**Usage Example**:

```go
data := []map[string]any{
    {
        "Name": "Alice",
        "Description": "This is a very long description that would normally make the table extremely wide and difficult to read in terminal output",
    },
    {
        "Name": "Bob",
        "Description": "Another long description with lots of text that needs to be wrapped",
    },
}

doc := output.New().
    Table("Users", data, output.WithKeys("Name", "Description")).
    Build()

// Limit column width to 30 characters
format := output.TableWithMaxColumnWidth(30)
out := output.NewOutput(
    output.WithFormat(format),
    output.WithWriter(output.NewStdoutWriter()),
)

err := out.Render(context.Background(), doc)
// Output will wrap long descriptions within 30-character column width
```

**Notes**:
- Uses go-pretty's `SetColumnConfigs()` with `WidthMax` setting
- Text wrapping is handled automatically by the table renderer
- Does not truncate text - content wraps to multiple lines within the cell
- Works with all table styles
- Particularly useful for terminal output where horizontal space is limited

### Renderer Interface

Custom renderers implement this interface:

```go
type Renderer interface {
    // Format returns the output format name
    Format() string

    // Render converts the document to bytes
    Render(ctx context.Context, doc *Document) ([]byte, error)

    // RenderTo streams output to a writer
    RenderTo(ctx context.Context, doc *Document, w io.Writer) error

    // SupportsStreaming indicates if streaming is supported
    SupportsStreaming() bool
}
```

### Writer System

#### Writer Interface

```go
type Writer interface {
    // Write outputs rendered data
    Write(ctx context.Context, format string, data []byte) error
}
```

#### Built-in Writers

Pre-implemented writers:

```go
// StdoutWriter writes to standard output
func NewStdoutWriter() Writer

// FileWriter writes to files with pattern support
func NewFileWriter(rootDir, pattern string) (Writer, error)

// S3Writer writes to AWS S3 (compatible with AWS SDK v2)
func NewS3Writer(client S3PutObjectAPI, bucket, keyPattern string) *S3Writer

// MultiWriter writes to multiple destinations
func NewMultiWriter(writers ...Writer) Writer
```

**File Pattern Examples**:
- `"report.{format}"` → `report.json`, `report.csv`
- `"output/{format}/data.{ext}"` → `output/json/data.json`

#### HTML Template System (v2.4.0+)

The HTML renderer can wrap content in complete HTML document templates with responsive CSS styling:

```go
// Use built-in responsive template
htmlFormat := output.HTML.WithOptions(
    output.WithHTMLTemplate(output.DefaultHTMLTemplate),
)

// Or create custom template
customTemplate := &output.HTMLTemplate{
    Title:       "My Report",
    Description: "Analysis Results",
    CSS:         output.DefaultResponsiveCSS,
    Author:      "AI Agent",
    Viewport:    "width=device-width, initial-scale=1.0",
    ThemeOverrides: map[string]string{
        "--primary-color": "#007bff",
        "--bg-color": "#f8f9fa",
    },
}

htmlFormat = output.HTML.WithOptions(
    output.WithHTMLTemplate(customTemplate),
)
```

**Built-in Templates**:
- `DefaultHTMLTemplate`: Modern responsive design with mobile-first CSS and WCAG AA colors
- `MinimalHTMLTemplate`: Clean HTML with no styling
- `MermaidHTMLTemplate`: Optimized for Mermaid diagram rendering

**Template Fields**:
```go
type HTMLTemplate struct {
    Title          string            // Page title
    Description    string            // Meta description
    Author         string            // Meta author
    Keywords       string            // Meta keywords
    Viewport       string            // Viewport meta tag
    Charset        string            // Character encoding (default: utf-8)
    CSS            string            // Embedded CSS styles
    ExternalCSS    []string          // External stylesheet URLs
    ThemeOverrides map[string]string // CSS custom property overrides
    HeadExtra      string            // Additional head content (unescaped)
    BodyClass      string            // Body element class
    BodyAttributes string            // Additional body attributes
    BodyExtra      string            // Content after main (unescaped)
}
```

**CSS Theming**:

The default template uses CSS custom properties for easy theming:

```go
template := &output.HTMLTemplate{
    Title: "Report",
    CSS:   output.DefaultResponsiveCSS,
    ThemeOverrides: map[string]string{
        "--primary-color":   "#0066cc",
        "--secondary-color": "#6c757d",
        "--bg-color":        "#ffffff",
        "--text-color":      "#212529",
        "--border-color":    "#dee2e6",
        "--font-family":     "Arial, sans-serif",
    },
}
```

**Responsive Features**:
- Mobile-first design with breakpoints at 480px and 768px
- Responsive table layout with mobile stacking
- System font stack for optimal performance
- WCAG AA compliant color contrast

**Fragment Mode**:

When using append mode, the HTML renderer automatically switches to fragment mode (no `<html>`, `<head>`, `<body>` tags) to avoid duplicate page structure.

#### Append Mode (v2.4.0+)

FileWriter and S3Writer support append mode to add content to existing files instead of replacing them:

```go
// Enable append mode with options
fw, err := output.NewFileWriterWithOptions(
    "./logs",
    "app.{ext}",
    output.WithAppendMode(),
)

// S3 append mode with conflict detection
sw := output.NewS3WriterWithOptions(
    s3Client,
    "my-bucket",
    "logs/app.{ext}",
    output.WithS3AppendMode(),
    output.WithMaxAppendSize(10*1024*1024), // 10MB limit
)
```

**FileWriter Options**:
```go
// Functional options for FileWriter configuration
func WithAppendMode() FileWriterOption
func WithPermissions(perms os.FileMode) FileWriterOption
func WithDisallowUnsafeAppend() FileWriterOption
```

**S3Writer Options**:
```go
// Functional options for S3Writer configuration
func WithS3AppendMode() S3WriterOption
func WithMaxAppendSize(size int64) S3WriterOption
```

**Format-Specific Behavior**:

| Format | Append Behavior | Notes |
|--------|-----------------|-------|
| JSON/YAML | Byte-level append | Creates NDJSON-style logging (newline-separated objects) |
| CSV | Header-aware append | Automatically skips headers from appended data |
| HTML | Marker-based insert | Inserts before `<!-- go-output-append -->` marker |
| Text/Table | Byte-level append | Simple file concatenation |

**HTML Append Marker**:
```go
const HTMLAppendMarker = "<!-- go-output-append -->"
```

When using append mode with HTML format, content is inserted before this marker comment. The marker must be present in the file for append operations to succeed.

**Thread Safety**: Append operations use `sync.Mutex` for safe concurrent writes within a single FileWriter instance.

**S3 Append**: Uses download-modify-upload pattern with ETag-based conflict detection. Not suitable for high-frequency concurrent writes.

**Examples**: See [v2/examples/append_mode/](../../examples/append_mode/) for practical usage patterns.

### Transformer System

#### Transformer Interface

```go
type Transformer interface {
    // Name returns the transformer name
    Name() string

    // Transform modifies the input bytes
    Transform(ctx context.Context, input []byte, format string) ([]byte, error)

    // CanTransform checks if this transformer applies
    CanTransform(format string) bool

    // Priority determines transform order (lower = earlier)
    Priority() int
}
```

#### Built-in Transformers

Pre-implemented transformers with two usage patterns:

**Direct Struct Instantiation** (for simple transformers):
```go
// Basic emoji conversion - no constructor needed
&EmojiTransformer{}

// Remove color codes - no constructor needed  
&RemoveColorsTransformer{}
```

**Constructor Functions** (for configurable transformers):
```go
// Color transformers
func NewColorTransformer() *ColorTransformer
func NewColorTransformerWithScheme(scheme ColorScheme) *ColorTransformer

// Sorting transformers
func NewSortTransformer(key string, ascending bool) *SortTransformer
func NewSortTransformerAscending(key string) *SortTransformer

// Line splitting transformers
func NewLineSplitTransformer(separator string) *LineSplitTransformer
func NewLineSplitTransformerDefault() *LineSplitTransformer

// Enhanced transformers with format awareness
func NewEnhancedEmojiTransformer() *EnhancedEmojiTransformer
func NewEnhancedColorTransformer() *EnhancedColorTransformer
func NewEnhancedSortTransformer(key string, ascending bool) *EnhancedSortTransformer

// Format-aware wrapper for existing transformers
func NewFormatAwareTransformer(transformer Transformer) *FormatAwareTransformer

// Transform pipeline for multiple transformers
func NewTransformPipeline() *TransformPipeline
```

**Struct Instantiation with Configuration** (alternative to constructors):
```go
// Configure transformers directly
&SortTransformer{Key: "Name", Ascending: true}
&LineSplitTransformer{Column: "Description", Separator: ","}
&ColorTransformer{Scheme: ColorScheme{Success: "green", Error: "red"}}
```

**ColorScheme Structure**:
```go
type ColorScheme struct {
    Success string // Color for positive/success values
    Warning string // Color for warning values
    Error   string // Color for error/failure values
    Info    string // Color for informational values
}
```

### Per-Content Transformations (v2.4.0+)

**Breaking Change**: The Pipeline API was removed in v2.4.0. Use per-content transformations instead.

Per-content transformations allow you to apply operations directly to individual content items at creation time, enabling different transformations for different tables in the same document.

#### Key Features

- **Flexible**: Different transformations on different content items
- **Type-Specific**: `WithTransformations()` for tables, `WithTextTransformations()` for text, `WithRawTransformations()` for raw content, `WithSectionTransformations()` for sections
- **Integrated**: Works seamlessly across all renderers
- **Thread-Safe**: All transformation operations are thread-safe
- **Immutable**: Transformations applied during rendering, preserving original document

#### Basic Usage

```go
// Different transformations for different tables
doc := output.New().
    Table("High Earners", employees,
        output.WithKeys("Name", "Department", "Salary"),
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["Salary"].(float64) > 100000
            }),
            output.NewSortOp(output.SortKey{Column: "Salary", Direction: output.Descending}),
        ),
    ).
    Table("Active Projects", projects,
        output.WithKeys("Project", "Status", "Priority"),
        output.WithTransformations(
            output.NewFilterOp(func(r output.Record) bool {
                return r["Status"] == "Active"
            }),
            output.NewLimitOp(10),
        ),
    ).
    Build()
```

#### Transformation Operations

**Filter Operation**:
```go
output.NewFilterOp(func(r output.Record) bool {
    return r["status"] == "active"
})
```

**Sort Operation**:
```go
// Single column sort
output.NewSortOp(output.SortKey{Column: "name", Direction: output.Ascending})

// Custom comparator
output.NewSortWithOp(func(a, b output.Record) int {
    // Custom comparison logic
    return 0
})
```

**Limit Operation**:
```go
output.NewLimitOp(10) // Get first 10 records
```

**Group By Operation**:
```go
output.NewGroupByOp(
    []string{"category", "status"},
    map[string]output.AggregateFunc{
        "count": output.CountAggregate,
        "total": output.SumAggregate("amount"),
    },
)
```

**Add Column Operation**:
```go
output.NewAddColumnOp("calculated_field", func(r output.Record) any {
    return r["value1"].(float64) + r["value2"].(float64)
})
```

#### Content-Specific Transformation Options

```go
// Tables
output.WithTransformations(ops...)

// Text content
output.WithTextTransformations(ops...)

// Raw content
output.WithRawTransformations(ops...)

// Sections
output.WithSectionTransformations(ops...)
```

#### Migration from Pipeline API

**Old (Pipeline API - Removed in v2.4.0)**:
```go
transformedDoc := doc.Pipeline().
    Filter(predicate).
    Sort(keys).
    Limit(10).
    Execute()
```

**New (Per-Content Transformations)**:
```go
doc := output.New().
    Table("Data", data,
        output.WithKeys("Name", "Value"),
        output.WithTransformations(
            output.NewFilterOp(predicate),
            output.NewSortOp(output.SortKey{Column: "Value", Direction: output.Descending}),
            output.NewLimitOp(10),
        ),
    ).
    Build()
```

**Benefits of Per-Content Transformations**:
- Each table can have different transformations
- More intuitive - transformations defined where content is created
- Better performance - only transforms what needs transforming
- Cleaner API - no intermediate transformed document

For detailed migration guidance, see [PIPELINE_MIGRATION.md](PIPELINE_MIGRATION.md).

### Data Transformation Pipeline System (REMOVED in v2.4.0)

**⚠️ Deprecated**: The Pipeline API was removed in v2.4.0. Use per-content transformations instead (see above).

The Pipeline API previously provided a fluent interface for performing data-level transformations on structured table content before rendering. This has been replaced with the more flexible per-content transformations system.

#### Key Features

- **Data-Level Operations**: Transform structured data before rendering
- **Fluent API**: Chain operations with method chaining
- **Format-Aware**: Operations can adapt behavior based on target output format
- **Performance Optimized**: Operations are reordered for optimal execution
- **Immutable**: Returns new transformed documents without modifying originals
- **Error Handling**: Fail-fast with detailed context information

#### Pipeline Interface

```go
// Create a pipeline from any document
pipeline := doc.Pipeline()

// Chain operations fluently
transformedDoc := doc.Pipeline().
    Filter(func(r Record) bool { return r["status"] == "active" }).
    Sort(SortKey{Column: "timestamp", Direction: Descending}).
    Limit(100).
    AddColumn("age_days", func(r Record) any {
        return time.Since(r["created"].(time.Time)).Hours() / 24
    }).
    Execute()
```

#### Core Operations

##### Filter Operation

Filters table records based on predicate functions:

```go
// Basic filtering
doc.Pipeline().
    Filter(func(r Record) bool {
        return r["status"] == "active"
    }).
    Execute()

// Complex filtering with type assertions
doc.Pipeline().
    Filter(func(r Record) bool {
        score, ok := r["score"].(float64)
        return ok && score > 85.0
    }).
    Execute()

// Multiple filters (combined with AND logic)
doc.Pipeline().
    Filter(func(r Record) bool { return r["category"] == "premium" }).
    Filter(func(r Record) bool { return r["verified"].(bool) }).
    Execute()
```

**Filter Function Signature**: `func(Record) bool`
- **Parameter**: `Record` (map[string]any) - Full record data
- **Returns**: `bool` - true to keep record, false to filter out
- **Type Assertions**: Use type assertions for type-safe access to record fields

##### Sort Operations

Sort table data by one or more columns:

```go
// Single column sort
doc.Pipeline().
    SortBy("name", Ascending).
    Execute()

// Multi-column sort with different directions
doc.Pipeline().
    Sort(
        SortKey{Column: "category", Direction: Ascending},
        SortKey{Column: "score", Direction: Descending},
        SortKey{Column: "name", Direction: Ascending},
    ).
    Execute()

// Custom comparator function
doc.Pipeline().
    SortWith(func(a, b Record) int {
        // Custom comparison logic
        aVal := a["priority"].(string)
        bVal := b["priority"].(string)
        priorities := map[string]int{"high": 3, "medium": 2, "low": 1}
        return priorities[bVal] - priorities[aVal] // Reverse order
    }).
    Execute()
```

**Sort Types**:
- `SortDirection`: `Ascending` or `Descending`
- `SortKey`: `{Column: string, Direction: SortDirection}`
- **Custom Comparator**: `func(a, b Record) int` - return -1, 0, or 1

##### Limit Operation

Restricts output to first N records:

```go
// Get top 10 records
doc.Pipeline().
    SortBy("score", Descending).
    Limit(10).
    Execute()

// Pagination-style limiting
doc.Pipeline().
    Filter(func(r Record) bool { return r["category"] == "premium" }).
    Limit(50).
    Execute()
```

##### GroupBy and Aggregation

Group records by columns and apply aggregate functions:

```go
// Basic grouping with count
doc.Pipeline().
    GroupBy(
        []string{"category", "status"},
        map[string]AggregateFunc{
            "count":       CountAggregate,
            "total_score": SumAggregate("score"),
            "avg_score":   AverageAggregate("score"),
            "max_score":   MaxAggregate("score"),
            "min_score":   MinAggregate("score"),
        },
    ).
    Execute()

// Custom aggregate function
customAggregate := func(records []Record) any {
    var uniqueUsers []string
    seen := make(map[string]bool)
    for _, r := range records {
        user := r["user"].(string)
        if !seen[user] {
            uniqueUsers = append(uniqueUsers, user)
            seen[user] = true
        }
    }
    return len(uniqueUsers)
}

doc.Pipeline().
    GroupBy(
        []string{"department"},
        map[string]AggregateFunc{
            "unique_users": customAggregate,
        },
    ).
    Execute()
```

**Built-in Aggregate Functions**:
- `CountAggregate`: Count records in group
- `SumAggregate(column)`: Sum numeric values
- `AverageAggregate(column)`: Average numeric values
- `MinAggregate(column)`: Minimum value
- `MaxAggregate(column)`: Maximum value
- **Custom Function**: `func([]Record) any`

##### AddColumn (Calculated Fields)

Add calculated columns based on existing data:

```go
// Simple calculated field
doc.Pipeline().
    AddColumn("full_name", func(r Record) any {
        return fmt.Sprintf("%s %s", r["first_name"], r["last_name"])
    }).
    Execute()

// Complex calculations with type assertions
doc.Pipeline().
    AddColumn("duration_hours", func(r Record) any {
        start := r["start_time"].(time.Time)
        end := r["end_time"].(time.Time)
        return end.Sub(start).Hours()
    }).
    AddColumn("status_icon", func(r Record) any {
        switch r["status"].(string) {
        case "completed":
            return "✅"
        case "failed":
            return "❌"
        case "pending":
            return "⏳"
        default:
            return "❓"
        }
    }).
    Execute()

// Add column at specific position
doc.Pipeline().
    AddColumnAt("id", func(r Record) any {
        return fmt.Sprintf("ID_%d", r["index"].(int))
    }, 0). // Insert at beginning
    Execute()
```

**Calculation Function**: `func(Record) any`
- **Parameter**: `Record` - Full record with all existing fields
- **Returns**: `any` - Calculated value for new column
- **Position**: Use `AddColumnAt()` to specify column position

#### Pipeline Options

Configure pipeline behavior and resource limits:

```go
// Custom pipeline options
options := PipelineOptions{
    MaxOperations:    50,              // Max operations allowed
    MaxExecutionTime: 10 * time.Second, // Execution timeout
}

doc.Pipeline().
    WithOptions(options).
    Filter(func(r Record) bool { return r["active"].(bool) }).
    Execute()

// Default options
// MaxOperations: 100
// MaxExecutionTime: 30 seconds
```

#### Format-Aware Transformations

Operations can adapt behavior based on target output format:

```go
// Execute with specific format context
transformedDoc := doc.Pipeline().
    Filter(func(r Record) bool { return r["visible"].(bool) }).
    ExecuteWithFormat(context.Background(), "json")

// Operations can check format and adapt behavior
// (Advanced usage - most operations work across all formats)
```

#### Error Handling

Pipeline operations use fail-fast error handling with detailed context:

```go
transformedDoc, err := doc.Pipeline().
    Filter(func(r Record) bool {
        // This could panic if "score" field doesn't exist
        return r["score"].(float64) > 50.0
    }).
    Execute()

if err != nil {
    var pipelineErr *PipelineError
    if errors.As(err, &pipelineErr) {
        fmt.Printf("Pipeline failed at operation: %s\n", pipelineErr.Operation)
        fmt.Printf("Stage: %d\n", pipelineErr.Stage)
        fmt.Printf("Cause: %v\n", pipelineErr.Cause)
        // Access additional context
        fmt.Printf("Context: %+v\n", pipelineErr.Context)
    }
}
```

**Error Types**:
- `PipelineError`: Detailed pipeline execution error
- `ValidationError`: Pre-execution validation error
- **Context Information**: Operation name, stage, input sample, context data

#### Performance Optimization

Pipeline automatically optimizes operation order:

```go
// User-defined order (potentially inefficient)
doc.Pipeline().
    Sort("name", Ascending).        // Expensive operation first
    Filter(func(r Record) bool {    // Filter after sort
        return r["active"].(bool)
    }).
    Limit(10)                       // Limit after operations

// Automatically optimized to:
// 1. Filter (reduces dataset size)
// 2. Sort (operates on smaller dataset)  
// 3. Limit (gets top N of final results)
```

**Optimization Strategy**:
1. **Filter**: Applied first to reduce data size
2. **AddColumn**: Added next (may be needed for sorting/grouping)
3. **GroupBy**: Applied to further reduce data
4. **Sort**: Applied to smaller datasets
5. **Limit**: Applied last to get top N results

#### Complete Pipeline Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    output "github.com/ArjenSchwarz/go-output/v2"
)

func main() {
    // Create sample data
    salesData := []map[string]any{
        {
            "salesperson": "Alice",
            "region":      "North",
            "amount":      15000.50,
            "date":        time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
            "status":      "completed",
        },
        {
            "salesperson": "Bob",
            "region":      "South",
            "amount":      22000.75,
            "date":        time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
            "status":      "completed",
        },
        {
            "salesperson": "Carol",
            "region":      "North",
            "amount":      8000.25,
            "date":        time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC),
            "status":      "pending",
        },
        // ... more data
    }

    // Create document
    doc := output.New().
        Table("Sales Data", salesData, 
            output.WithKeys("salesperson", "region", "amount", "date", "status")).
        Build()

    // Transform data with pipeline
    transformedDoc, err := doc.Pipeline().
        // Filter completed sales only
        Filter(func(r output.Record) bool {
            return r["status"] == "completed"
        }).
        // Add calculated commission field
        AddColumn("commission", func(r output.Record) any {
            amount := r["amount"].(float64)
            return amount * 0.05 // 5% commission
        }).
        // Add days since sale
        AddColumn("days_ago", func(r output.Record) any {
            saleDate := r["date"].(time.Time)
            return int(time.Since(saleDate).Hours() / 24)
        }).
        // Sort by amount (highest first)
        SortBy("amount", output.Descending).
        // Limit to top 10
        Limit(10).
        Execute()

    if err != nil {
        log.Fatal(err)
    }

    // Output results
    out := output.NewOutput(
        output.WithFormats(output.Table, output.JSON),
        output.WithWriter(output.NewStdoutWriter()),
    )

    if err := out.Render(context.Background(), transformedDoc); err != nil {
        log.Fatal(err)
    }

    // Access transformation statistics
    if stats := transformedDoc.GetTransformStats(); stats != nil {
        fmt.Printf("\nTransformation Stats:\n")
        fmt.Printf("Input Records: %d\n", stats.InputRecords)
        fmt.Printf("Output Records: %d\n", stats.OutputRecords)
        fmt.Printf("Filtered Count: %d\n", stats.FilteredCount)
        fmt.Printf("Duration: %v\n", stats.Duration)
        
        for _, opStat := range stats.Operations {
            fmt.Printf("Operation %s: %v (%d records)\n", 
                opStat.Name, opStat.Duration, opStat.RecordsProcessed)
        }
    }
}
```

#### Best Practices

1. **Type Safety**: Always use type assertions when accessing record fields
2. **Error Handling**: Handle pipeline errors with proper context checking
3. **Performance**: Let pipeline optimize operation order automatically
4. **Immutability**: Pipeline returns new documents, preserving originals
5. **Resource Limits**: Use appropriate pipeline options for large datasets
6. **Schema Preservation**: Pipeline maintains table schema and key ordering

#### Migration from Byte Transformers

**Old Approach** (byte transformers):
```go
// Post-rendering text manipulation
transformer := &output.SortTransformer{Key: "name", Ascending: true}
out := output.NewOutput(
    output.WithTransformer(transformer),
)
```

**New Approach** (data pipeline):
```go
// Pre-rendering data transformation
transformedDoc := doc.Pipeline().
    SortBy("name", output.Ascending).
    Execute()
```

**When to Use Each**:
- **Data Pipeline**: For data operations (filter, sort, aggregate, calculate fields)
- **Byte Transformers**: For presentation styling (colors, emoji, formatting)

### Progress System

#### Progress Interface

```go
type Progress interface {
    // Core progress methods
    SetTotal(total int)
    SetCurrent(current int)
    Increment(delta int)
    SetStatus(status string)
    Complete()
    Fail(err error)

    // v1 compatibility methods
    SetColor(color ProgressColor)
    IsActive() bool
    SetContext(ctx context.Context)

    // v2 enhancements
    Close() error
}
```

#### Progress Types

```go
// ProgressColor for visual feedback
type ProgressColor int

const (
    ProgressColorDefault ProgressColor = iota
    ProgressColorGreen   // Success state
    ProgressColorRed     // Error state
    ProgressColorYellow  // Warning state
    ProgressColorBlue    // Informational state
)
```

#### Progress Constructors

```go
// NewProgress creates format-aware progress
func NewProgress(format string, opts ...ProgressOption) Progress

// NewProgressForFormats creates progress for multiple formats
func NewProgressForFormats(formats []Format, opts ...ProgressOption) Progress

// NewNoOpProgress creates a no-operation progress indicator
func NewNoOpProgress() Progress
```

### Error Handling

#### Error Types

Structured error types for different failure modes:

```go
// RenderError indicates rendering failure
type RenderError struct {
    Format   string
    Renderer string
    Content  string
    Cause    error
}

// ValidationError indicates invalid input
type ValidationError struct {
    Field   string
    Value   any
    Message string
}

// TransformError indicates transformation failure
type TransformError struct {
    Transformer string
    Format      string
    Input       []byte
    Cause       error
}

// WriterError indicates writer failure
type WriterError struct {
    Writer    string
    Format    string
    Operation string
    Cause     error
}
```

All error types implement `error` and can be unwrapped using `errors.Unwrap()`.

### Utility Functions

#### Data Types

```go
// Record represents a table row
type Record map[string]any

// GenerateID creates unique content identifiers
func GenerateID() string
```

#### Helper Functions

```go
// Helper functions for testing and validation
func ValidateNonNil(name string, value any) error
func ValidateSliceNonEmpty(name string, slice any) error
func FailFast(validators ...error) error
```

#### Inline Styling Functions

The v2 library provides stateless inline styling functions for adding ANSI color codes to text. These functions enable consistent terminal coloring without global state, making them safe for concurrent use.

```go
// Basic styling functions
func StyleWarning(text string) string   // Red bold text
func StylePositive(text string) string  // Green bold text
func StyleNegative(text string) string  // Red text
func StyleInfo(text string) string      // Blue text
func StyleBold(text string) string      // Bold text

// Conditional styling functions (apply styling only if condition is true)
func StyleWarningIf(condition bool, text string) string
func StylePositiveIf(condition bool, text string) string
func StyleNegativeIf(condition bool, text string) string
func StyleInfoIf(condition bool, text string) string
func StyleBoldIf(condition bool, text string) string
```

**Usage Examples**:

```go
// Simple inline styling in table data
data := []map[string]any{
    {
        "Name":   "Server 1",
        "Status": output.StylePositive("Running"),
        "CPU":    output.StyleWarningIf(cpuUsage > 80, fmt.Sprintf("%d%%", cpuUsage)),
    },
    {
        "Name":   "Server 2",
        "Status": output.StyleNegative("Down"),
        "CPU":    fmt.Sprintf("%d%%", cpuUsage),
    },
}

doc := output.New().
    Table("Server Status", data, output.WithKeys("Name", "Status", "CPU")).
    Build()

// Text content styling
doc := output.New().
    Text(output.StyleBold("Important Notice:")).
    Text(output.StyleInfo("System maintenance scheduled for tonight")).
    Build()

// Conditional styling based on values
errorCount := 5
message := fmt.Sprintf("Found %d errors", errorCount)
styledMessage := output.StyleWarningIf(errorCount > 0, message)

doc := output.New().
    Text(styledMessage).
    Build()
```

**Notes**:
- Colors use the `fatih/color` library for ANSI terminal codes
- Colors are automatically enabled even in non-TTY environments
- Functions are stateless and thread-safe
- Works with all output formats (ANSI codes pass through to terminal formats)
- For non-terminal formats (HTML, Markdown), use ColorTransformer to strip ANSI codes

## Format-Specific Behavior

### Array Handling in Output Formats

The v2 library automatically handles arrays (slices) in table data with format-appropriate rendering:

#### Table Format Array Handling

Arrays are rendered as newline-separated values within table cells:

```go
data := []map[string]any{
    {
        "Name": "Alice",
        "Tags": []string{"admin", "developer", "reviewer"},
        "Roles": []string{"Owner", "Maintainer"},
    },
    {
        "Name": "Bob",
        "Tags": []string{"user"},
        "Roles": []string{"Contributor"},
    },
}

doc := output.New().
    Table("Users", data, output.WithKeys("Name", "Tags", "Roles")).
    Build()

out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithWriter(output.NewStdoutWriter()),
)

err := out.Render(context.Background(), doc)

// Output (in table format):
// ┌───────┬───────────┬──────────────┐
// │ NAME  │ TAGS      │ ROLES        │
// ├───────┼───────────┼──────────────┤
// │ Alice │ admin     │ Owner        │
// │       │ developer │ Maintainer   │
// │       │ reviewer  │              │
// ├───────┼───────────┼──────────────┤
// │ Bob   │ user      │ Contributor  │
// └───────┴───────────┴──────────────┘
```

#### Markdown Format Array Handling

Arrays are rendered as `<br/>`-separated values in markdown table cells:

```go
// Same data as above
out := output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithWriter(output.NewStdoutWriter()),
)

err := out.Render(context.Background(), doc)

// Output (in markdown):
// | Name  | Tags                              | Roles                  |
// |-------|-----------------------------------|------------------------|
// | Alice | admin<br/>developer<br/>reviewer  | Owner<br/>Maintainer   |
// | Bob   | user                              | Contributor            |
```

The `<br/>` tags render correctly in GitHub, GitLab, and other markdown viewers while maintaining table cell integrity.

#### JSON/YAML Format Array Handling

Arrays are preserved natively in structured formats:

```go
// Same data
out := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewStdoutWriter()),
)

// Output preserves arrays as JSON arrays:
// [
//   {
//     "Name": "Alice",
//     "Tags": ["admin", "developer", "reviewer"],
//     "Roles": ["Owner", "Maintainer"]
//   },
//   ...
// ]
```

#### Supported Array Types

The array handling supports:
- `[]string` - Most common case for string slices
- `[]any` - Generic slices with any element type
- Empty arrays render as empty strings in table/markdown formats

**Notes**:
- Array elements are automatically escaped in markdown format
- Table format uses newlines for vertical layout
- Markdown format uses HTML `<br/>` for compatibility
- JSON/YAML preserve native array structure
- CSV format joins arrays with semicolons (`;`) by default

## Common Usage Patterns

### Basic Table Output

```go
doc := output.New().
    Table("Results", []map[string]any{
        {"Name": "Alice", "Score": 95},
        {"Name": "Bob", "Score": 87},
    }, output.WithKeys("Name", "Score")).
    Build()

out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithWriter(output.NewStdoutWriter()),
)

err := out.Render(context.Background(), doc)
```

### Multiple Formats and Destinations

```go
doc := output.New().
    Table("Data", data, output.WithKeys("ID", "Name", "Status")).
    Text("Report generated on: " + time.Now().Format(time.RFC3339)).
    Build()

fileWriter, _ := output.NewFileWriter("./output", "report.{format}")

out := output.NewOutput(
    output.WithFormats(output.JSON, output.CSV, output.HTML),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithWriter(fileWriter),
)

err := out.Render(context.Background(), doc)
```

### Mixed Content Document

```go
doc := output.New().
    Header("System Report").
    Section("User Statistics", func(b *output.Builder) {
        b.Table("Active Users", activeUsers, output.WithKeys("Name", "LastLogin"))
        b.Table("User Roles", roles, output.WithKeys("Role", "Count"))
    }).
    Section("System Health", func(b *output.Builder) {
        b.Text("All systems operational").
        b.Table("Metrics", metrics, output.WithKeys("Metric", "Value", "Status"))
    }).
    Build()
```

### Transformer Usage Patterns

```go
// Pattern 1: Direct struct instantiation (simple transformers)
out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformer(&output.EmojiTransformer{}),
    output.WithTransformer(&output.RemoveColorsTransformer{}),
    output.WithWriter(output.NewStdoutWriter()),
)

// Pattern 2: Constructor functions (configurable transformers)
colorTransformer := output.NewColorTransformerWithScheme(output.ColorScheme{
    Success: "green",
    Info:    "blue", 
    Warning: "yellow",
    Error:   "red",
})
sortTransformer := output.NewSortTransformer("Name", true)

out = output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformers(colorTransformer, sortTransformer),
    output.WithWriter(output.NewStdoutWriter()),
)

// Pattern 3: Struct instantiation with configuration
out = output.NewOutput(
    output.WithFormat(output.Table),
    output.WithTransformer(&output.SortTransformer{Key: "Name", Ascending: true}),
    output.WithTransformer(&output.LineSplitTransformer{Column: "Description", Separator: ","}),
    output.WithWriter(output.NewStdoutWriter()),
)
```

### Progress Tracking

```go
progress := output.NewProgress(output.FormatTable,
    output.WithProgressColor(output.ProgressColorGreen),
    output.WithProgressStatus("Processing data"),
)

out := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithProgress(progress),
)
```

### Graph and Chart Generation

```go
// Flow chart
doc := output.New().
    Graph("Process Flow", []output.Edge{
        {From: "Start", To: "Process", Label: "begin"},
        {From: "Process", To: "End", Label: "complete"},
    }).
    Build()

// Gantt chart
tasks := []output.GanttTask{
    {ID: "task1", Title: "Design", StartDate: "2024-01-01", EndDate: "2024-01-15"},
    {ID: "task2", Title: "Development", StartDate: "2024-01-16", EndDate: "2024-02-15"},
}

doc = output.New().
    GanttChart("Project Timeline", tasks).
    Build()

out := output.NewOutput(
    output.WithFormat(output.Mermaid),
    output.WithWriter(output.NewStdoutWriter()),
)
```

### Collapsible Content Patterns

#### Simple Field Collapsible Content

```go
// Data with error arrays and long paths
data := []map[string]any{
    {
        "file": "/very/long/path/to/project/src/components/UserDashboard.tsx",
        "errors": []string{"Import missing", "Unused variable", "Type error"},
        "warnings": []string{"Deprecated API", "Performance concern"},
    },
}

// Create table with collapsible formatters
doc := output.New().
    Table("Analysis Results", data, output.WithSchema(
        output.Field{
            Name: "file",
            Type: "string",
            Formatter: output.FilePathFormatter(25), // Shorten paths > 25 chars
        },
        output.Field{
            Name: "errors",
            Type: "array", 
            Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false)),
        },
        output.Field{
            Name: "warnings",
            Type: "array",
            Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(true)),
        },
    )).
    Build()

// Render to GitHub-compatible markdown
out := output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithWriter(output.NewStdoutWriter()),
)
```

#### Custom Collapsible Formatter

```go
// Custom formatter for configuration objects
func configFormatter(val any) any {
    if config, ok := val.(map[string]any); ok {
        configJSON, _ := json.MarshalIndent(config, "", "  ")
        if len(configJSON) > 100 {
            return output.NewCollapsibleValue(
                fmt.Sprintf("Config (%d keys)", len(config)),
                string(configJSON),
                output.WithCollapsibleExpanded(false),
                output.WithMaxLength(200),
            )
        }
    }
    return val
}

schema := output.WithSchema(
    output.Field{Name: "name", Type: "string"},
    output.Field{
        Name: "config",
        Type: "object",
        Formatter: configFormatter,
    },
)
```

#### Collapsible Sections for Report Organization

```go
// Create detailed analysis tables
usersTable := output.NewTableContent("User Analysis", userData)
performanceTable := output.NewTableContent("Performance Metrics", perfData)
securityTable := output.NewTableContent("Security Issues", securityData)

// Wrap tables in collapsible sections
userSection := output.NewCollapsibleTable(
    "User Activity Analysis",
    usersTable,
    output.WithSectionExpanded(true), // Expanded by default
)

performanceSection := output.NewCollapsibleTable(
    "Performance Analysis", 
    performanceTable,
    output.WithSectionExpanded(false), // Collapsed by default
)

// Multi-content section
securitySection := output.NewCollapsibleReport(
    "Security Report",
    []output.Content{
        output.NewTextContent("Security scan completed with 5 issues found"),
        securityTable,
        output.NewTextContent("Immediate action required for critical issues"),
    },
    output.WithSectionExpanded(false),
)

// Build comprehensive document
doc := output.New().
    Header("System Analysis Report").
    Text("Generated on " + time.Now().Format("2006-01-02 15:04:05")).
    Add(userSection).
    Add(performanceSection).
    Add(securitySection).
    Build()
```

#### Nested Collapsible Sections

```go
// Create nested hierarchy (max 3 levels)
subSection1 := output.NewCollapsibleTable(
    "Database Performance",
    dbTable,
    output.WithSectionLevel(2),
    output.WithSectionExpanded(false),
)

subSection2 := output.NewCollapsibleTable(
    "API Response Times",
    apiTable,
    output.WithSectionLevel(2),
    output.WithSectionExpanded(false),
)

mainSection := output.NewCollapsibleReport(
    "Infrastructure Analysis",
    []output.Content{
        output.NewTextContent("Infrastructure health check results"),
        subSection1,
        subSection2,
    },
    output.WithSectionLevel(1),
    output.WithSectionExpanded(true),
)

doc := output.New().Add(mainSection).Build()
```

#### Cross-Format Collapsible Rendering

```go
// Same data rendered in multiple formats with different behaviors
data := []map[string]any{
    {"errors": []string{"Error 1", "Error 2", "Error 3"}},
}

table := output.NewTableContent("Issues", data, output.WithSchema(
    output.Field{
        Name: "errors",
        Type: "array",
        Formatter: output.ErrorListFormatter(output.WithCollapsibleExpanded(false)),
    },
))

doc := output.New().Add(table).Build()

// Markdown: GitHub <details> elements
markdownOut := output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithWriter(output.NewFileWriter(".", "report.md")),
)

// JSON: Structured collapsible data
jsonOut := output.NewOutput(
    output.WithFormat(output.JSON),
    output.WithWriter(output.NewFileWriter(".", "report.json")),
)

// Table: Terminal-friendly with expansion indicators
tableOut := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithWriter(output.NewStdoutWriter()),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        TableHiddenIndicator: "[expand to view all errors]",
    }),
)

// CSV: Automatic detail columns for spreadsheet analysis
csvOut := output.NewOutput(
    output.WithFormat(output.CSV),
    output.WithWriter(output.NewFileWriter(".", "report.csv")),
)

// Render all formats
ctx := context.Background()
markdownOut.Render(ctx, doc)  // Creates expandable <details> 
jsonOut.Render(ctx, doc)      // Creates {"type": "collapsible", ...}
tableOut.Render(ctx, doc)     // Shows: "3 errors [expand to view all errors]"
csvOut.Render(ctx, doc)       // Creates: errors, errors_details columns
```

#### Global Expansion Control

```go
// Development/debug mode: expand all content
debugOut := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        GlobalExpansion: true, // Override all IsExpanded() settings
    }),
    output.WithWriter(output.NewStdoutWriter()),
)

// Production mode: respect individual expansion settings
prodOut := output.NewOutput(
    output.WithFormat(output.Markdown),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        GlobalExpansion: false, // Use individual IsExpanded() values
        MaxDetailLength: 500,   // Limit detail length
        TruncateIndicator: "... (truncated)",
    }),
    output.WithWriter(output.NewStdoutWriter()),
)
```

#### Advanced Collapsible Configuration

```go
// Custom HTML output with branded styling
htmlOut := output.NewOutput(
    output.WithFormat(output.HTML),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        HTMLCSSClasses: map[string]string{
            "details": "company-collapsible",
            "summary": "company-summary",
            "content": "company-details",
        },
    }),
    output.WithWriter(output.NewFileWriter(".", "report.html")),
)

// Custom table output with branded indicators
tableOut := output.NewOutput(
    output.WithFormat(output.Table),
    output.WithCollapsibleConfig(output.CollapsibleConfig{
        TableHiddenIndicator: "📋 Click to expand detailed information",
        MaxDetailLength:      150,
        TruncateIndicator:    "... [see full details in expanded view]",
    }),
    output.WithWriter(output.NewStdoutWriter()),
)
```

## Thread Safety

All v2 components are designed to be thread-safe:

- **Document**: Immutable after Build(), safe for concurrent reads
- **Builder**: Thread-safe during construction using mutexes
- **Output**: Thread-safe configuration and rendering
- **Content**: Immutable after creation

## Performance Considerations

- **Memory Efficiency**: Uses encoding.TextAppender and encoding.BinaryAppender interfaces
- **Concurrent Rendering**: Processes multiple formats in parallel
- **Streaming Support**: Large datasets can be streamed to avoid memory issues
- **Key Ordering**: No performance penalty - preserves user order without sorting

## Migration from v1

For migration guidance, see:
- `MIGRATION.md` - Complete migration guide
- `MIGRATION_EXAMPLES.md` - Before/after examples
- `MIGRATION_QUICK_REFERENCE.md` - Quick lookup reference

## Error Handling Best Practices

1. **Context Cancellation**: Always pass context for cancellable operations
2. **Error Wrapping**: Use structured error types for detailed debugging
3. **Early Validation**: Builder validates inputs during construction
4. **Resource Cleanup**: Call `Close()` on Output when done

## Extension Points

The v2 API is designed for extensibility:

- **Custom Renderers**: Implement `Renderer` interface for new formats
- **Custom Transformers**: Implement `Transformer` interface for data processing
- **Custom Writers**: Implement `Writer` interface for new destinations
- **Custom Content**: Implement `Content` interface for specialized content types

## API Method Quick Reference

### Builder Methods
| Method | Purpose | Example |
|--------|---------|---------|
| `New()` | Create new builder | `output.New()` |
| `Build()` | Finalize document | `builder.Build()` |
| `Table(title, data, opts)` | Add table with key order | `Table("Users", data, WithKeys("Name", "Age"))` |
| `Text(text, opts)` | Add text content | `Text("Summary", WithBold(true))` |
| `Header(text)` | Add header text | `Header("Report Title")` |
| `Section(title, fn, opts)` | Group content | `Section("Details", func(b) {...})` |
| `Raw(format, data, opts)` | Add raw content | `Raw("html", htmlBytes)` |
| `Graph(title, edges)` | Add graph diagram | `Graph("Flow", edges)` |
| `GanttChart(title, tasks)` | Add Gantt chart | `GanttChart("Timeline", tasks)` |
| `PieChart(title, slices, show)` | Add pie chart | `PieChart("Stats", slices, true)` |
| `DrawIO(title, records, header)` | Add Draw.io diagram | `DrawIO("Architecture", records, header)` |
| `SetMetadata(key, value)` | Set metadata | `SetMetadata("author", "AI Agent")` |
| `HasErrors()` | Check for errors | `if builder.HasErrors() {...}` |
| `Errors()` | Get all errors | `for _, err := range builder.Errors()` |

### Table Options
| Option | Purpose | Example |
|--------|---------|---------|
| `WithKeys(keys...)` | Set column order | `WithKeys("ID", "Name", "Status")` |
| `WithSchema(fields...)` | Define full schema | `WithSchema(Field{Name: "id", Type: "int"})` |
| `WithAutoSchema()` | Auto-detect schema | `WithAutoSchema()` |

### Collapsible Options (v2.1.0+)
| Option | Purpose | Example |
|--------|---------|---------|
| `WithCollapsibleExpanded(bool)` | Set default state | `WithCollapsibleExpanded(false)` |
| `WithMaxLength(int)` | Limit detail length | `WithMaxLength(200)` |
| `WithCodeFences(lang)` | Add syntax highlighting (v2.1.1+) | `WithCodeFences("json")` |
| `WithoutCodeFences()` | Disable code fences | `WithoutCodeFences()` |
| `WithFormatHint(fmt, hints)` | Format-specific hints | `WithFormatHint("html", map[string]any{"class": "custom"})` |

### Built-in Formatters
| Formatter | Purpose | Example |
|-----------|---------|---------|
| `ErrorListFormatter(opts)` | Format error arrays | `ErrorListFormatter(WithCollapsibleExpanded(false))` |
| `FilePathFormatter(max, opts)` | Shorten long paths | `FilePathFormatter(30)` |
| `JSONFormatter(max, opts)` | Format JSON objects | `JSONFormatter(100, WithCodeFences("json"))` |
| `CollapsibleFormatter(tmpl, fn, opts)` | Custom collapsible | `CollapsibleFormatter("Summary", detailFunc)` |

### Output Configuration
| Method | Purpose | Example |
|--------|---------|---------|
| `NewOutput(opts...)` | Create output instance | `NewOutput(WithFormat(Table))` |
| `WithFormat(format)` | Set single format | `WithFormat(output.JSON)` |
| `WithFormats(formats...)` | Set multiple formats | `WithFormats(JSON, CSV, Table)` |
| `WithWriter(writer)` | Set output destination | `WithWriter(NewStdoutWriter())` |
| `WithWriters(writers...)` | Multiple destinations | `WithWriters(stdout, file)` |
| `WithTransformer(t)` | Add transformer | `WithTransformer(&EmojiTransformer{})` |
| `WithProgress(p)` | Add progress tracking | `WithProgress(progress)` |
| `WithCollapsibleConfig(cfg)` | Configure collapsibles | `WithCollapsibleConfig(config)` |

### Writer Constructors
| Constructor | Purpose | Example |
|-------------|---------|---------|
| `NewStdoutWriter()` | Write to console | `NewStdoutWriter()` |
| `NewFileWriter(dir, pattern)` | Write to files | `NewFileWriter("./out", "report.{format}")` |
| `NewS3Writer(client, bucket, key)` | Write to S3 (AWS SDK v2) | `NewS3Writer(s3Client, "bucket", "key.{format}")` |
| `NewMultiWriter(writers...)` | Multiple outputs | `NewMultiWriter(stdout, file)` |

### Built-in Formats
| Format | Constant | Streaming | Notes |
|--------|----------|-----------|-------|
| JSON | `output.JSON` | ✓ | Structured data |
| YAML | `output.YAML` | ✓ | Human-readable structured |
| CSV | `output.CSV` | ✓ | Spreadsheet compatible |
| HTML | `output.HTML` | ✓ | Web display |
| Table | `output.Table` | ✓ | Terminal display |
| Markdown | `output.Markdown` | ✓ | Documentation format |
| DOT | `output.DOT` | ✗ | Graphviz diagrams |
| Mermaid | `output.Mermaid` | ✗ | Mermaid diagrams |
| DrawIO | `output.DrawIO` | ✗ | Draw.io CSV format |

### AWS Icons Package (v2/icons)

The `v2/icons` package provides AWS service icons for Draw.io diagrams, enabling professional architecture diagrams with proper AWS branding.

#### Basic Icon Retrieval

```go
import "github.com/ArjenSchwarz/go-output/v2/icons"

// Get a specific AWS icon
style, err := icons.GetAWSShape("Compute", "EC2")
if err != nil {
    log.Fatal(err)
}

// Check if a shape exists
if icons.HasAWSShape("Compute", "Lambda") {
    style, _ := icons.GetAWSShape("Compute", "Lambda")
    // Use the style
}
```

#### Discovering Available Icons

```go
// List all service groups
groups := icons.AllAWSGroups()
// Returns: ["Analytics", "Compute", "Database", "Networking", "Storage", ...]

// List shapes in a specific group
shapes, err := icons.AWSShapesInGroup("Compute")
// Returns: ["EC2", "ECS", "EKS", "Lambda", ...]
```

#### Integration with Draw.io Diagrams

```go
// Prepare data with AWS service information
data := []map[string]any{
    {"Name": "API Gateway", "Type": "APIGateway", "Group": "Networking Content Delivery"},
    {"Name": "Lambda Function", "Type": "Lambda", "Group": "Compute"},
    {"Name": "DynamoDB Table", "Type": "DynamoDB", "Group": "Database"},
}

// Add AWS icon styles to each record
for _, record := range data {
    style, err := icons.GetAWSShape(record["Group"].(string), record["Type"].(string))
    if err == nil {
        record["IconStyle"] = style
    }
}

// Create Draw.io diagram with dynamic icons using placeholders
header := output.DrawIOHeader{
    Style: "%IconStyle%",  // Placeholder replaced per-record
    Label: "%Name%",
    Width: 78,
    Height: 78,
}

doc := output.New().
    DrawIO("AWS Architecture", convertToRecords(data), header).
    Build()

out := output.NewOutput(
    output.WithFormat(output.DrawIO),
    output.WithWriter(output.NewFileWriter(".", "architecture.csv")),
)

err := out.Render(context.Background(), doc)
```

#### Migration from v1

The v2 icons package replaces v1's `drawio.GetAWSShape()`:

```go
// v1
import "github.com/ArjenSchwarz/go-output/drawio"
style := drawio.GetAWSShape("Compute", "EC2") // returns empty string on error

// v2
import "github.com/ArjenSchwarz/go-output/v2/icons"
style, err := icons.GetAWSShape("Compute", "EC2") // returns error
if err != nil {
    // handle error
}
```

**Key Differences from v1:**
- Explicit error handling with descriptive error messages
- Same embedded aws.json dataset (600+ AWS services)
- Same case-sensitive matching behavior
- Thread-safe concurrent access

#### Common AWS Service Groups

| Group | Example Services |
|-------|------------------|
| Compute | EC2, ECS, EKS, Lambda, Batch |
| Storage | S3, EBS, EFS, FSx, Backup |
| Database | RDS, DynamoDB, ElastiCache, Neptune |
| Networking Content Delivery | VPC, CloudFront, Route 53, API Gateway |
| Security Identity Compliance | IAM, Cognito, Secrets Manager, KMS |
| Analytics | Athena, EMR, Kinesis, Glue, QuickSight |
| Management Governance | CloudWatch, CloudFormation, Systems Manager |
| Machine Learning | SageMaker, Rekognition, Comprehend |

Use `icons.AllAWSGroups()` for the complete list.

### Version History
| Version | Key Features |
|---------|--------------|
| v2.4.0 | **Breaking**: Pipeline API removed, per-content transformations, file/S3 append mode, HTML template system |
| v2.3.0 | AWS Icons package, inline styling functions, table max column width, array handling |
| v2.2.0 | Data transformation pipeline system, development tooling automation |
| v2.1.3 | Enhanced markdown table escaping for pipes, asterisks, underscores, backticks, brackets |
| v2.1.1 | Code fence support for collapsible fields with syntax highlighting |
| v2.1.0 | Complete collapsible content system with format-aware rendering |
| v2.0.0 | Complete redesign with Builder pattern, key order preservation, thread safety |

For more examples and advanced usage, see the `/examples` directory in the repository.