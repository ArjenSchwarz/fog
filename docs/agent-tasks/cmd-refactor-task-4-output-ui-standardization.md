# Task 4: Output and UI Standardization

## Objective

Create consistent user interface patterns across all commands by standardizing output formatting, progress indicators, confirmation prompts, and establishing reusable UI components.

## Current State

### Problems
- Inconsistent output formatting across commands
- Mixed use of different output libraries and patterns
- Hardcoded UI strings scattered throughout command files
- No standardized progress indicators or user feedback
- Inconsistent error message formatting and presentation
- Mixed confirmation prompt patterns

### Current UI Implementation
- `helpers.go` - Basic confirmation prompt function
- Individual commands handle their own output formatting
- Direct use of `fmt.Print*` functions throughout codebase
- `go-output` library used inconsistently
- Global `outputsettings` variable mixed with command-specific formatting
- Color output handled inconsistently

### Problematic Patterns
```go
// Current: Inconsistent output patterns
fmt.Printf("Deploying new stack '%v' to region %v\n", stackName, region)
fmt.Print(outputsettings.StringSuccess("Deployment successful"))
color.Red("Error: %s", err.Error())
```

## Target State

### Goals
- Consistent output formatting across all commands
- Reusable UI components for common interactions
- Standardized progress indicators and status messages
- Centralized management of output themes and styling
- Context-aware output (supports different formats: table, JSON, etc.)
- Accessible and user-friendly interface patterns

### UI Architecture
```
cmd/
├── ui/
│   ├── interfaces.go           # UI component interfaces
│   ├── console/
│   │   ├── output.go           # Console output handler
│   │   ├── progress.go         # Progress indicators
│   │   ├── prompts.go          # User prompts and confirmations
│   │   ├── tables.go           # Table formatting
│   │   └── colors.go           # Color and styling
│   ├── formatters/
│   │   ├── deployment.go       # Deployment-specific formatting
│   │   ├── drift.go           # Drift detection formatting
│   │   ├── changeset.go       # Changeset formatting
│   │   └── common.go          # Common format patterns
│   ├── components/
│   │   ├── status.go          # Status messages
│   │   ├── spinner.go         # Loading spinners
│   │   ├── confirmation.go    # Confirmation dialogs
│   │   └── validation.go      # Validation message display
│   └── themes/
│       ├── default.go         # Default theme
│       ├── minimal.go         # Minimal output theme
│       └── json.go           # JSON output theme
```

## Prerequisites

- Task 1: Command Structure Reorganization (provides handler framework)
- Task 5: Error Handling (provides structured errors)

## Step-by-Step Implementation

### Step 1: Define UI Interfaces

**File**: `cmd/ui/interfaces.go`

```go
package ui

import (
    "context"
    "io"
    "time"
    format "github.com/ArjenSchwarz/go-output"
)

// OutputHandler manages all output operations
type OutputHandler interface {
    // Basic output methods
    Success(message string)
    Info(message string)
    Warning(message string)
    Error(message string)
    Debug(message string)

    // Formatted output
    Table(data interface{}, options TableOptions) error
    JSON(data interface{}) error

    // Progress and status
    StartProgress(message string) ProgressIndicator
    SetStatus(message string)

    // User interaction
    Confirm(message string) bool
    ConfirmWithDefault(message string, defaultValue bool) bool

    // Output settings
    SetVerbose(verbose bool)
    SetQuiet(quiet bool)
    SetOutputFormat(format OutputFormat)

    // Writers
    GetWriter() io.Writer
    GetErrorWriter() io.Writer
}

// ProgressIndicator represents a progress indicator
type ProgressIndicator interface {
    Update(message string)
    Success(message string)
    Error(message string)
    Stop()
}

// TableOptions configures table output
type TableOptions struct {
    Title       string
    Headers     []string
    MaxWidth    int
    Style       TableStyle
    SortBy      string
    ShowIndex   bool
}

// TableStyle defines table visual style
type TableStyle int

const (
    TableStyleDefault TableStyle = iota
    TableStyleMinimal
    TableStyleBordered
    TableStyleCompact
)

// OutputFormat defines the output format
type OutputFormat int

const (
    FormatTable OutputFormat = iota
    FormatJSON
    FormatCSV
    FormatYAML
    FormatText
)

// Theme defines UI styling and colors
type Theme interface {
    Success() string
    Info() string
    Warning() string
    Error() string
    Debug() string
    Emphasis() string
    Muted() string

    // Progress colors
    ProgressSpinner() string
    ProgressSuccess() string
    ProgressError() string

    // Table styling
    TableHeader() string
    TableBorder() string
    TableData() string
}

// Formatter handles specific output formatting
type Formatter interface {
    FormatDeploymentInfo(info DeploymentInfo) string
    FormatChangeset(changeset ChangesetInfo) string
    FormatDriftResult(result DriftResult) string
    FormatStackInfo(stack StackInfo) string
}

// ValidationDisplayer handles validation message display
type ValidationDisplayer interface {
    ShowValidationErrors(errors []ValidationError)
    ShowValidationWarnings(warnings []ValidationWarning)
    ShowValidationSummary(summary ValidationSummary)
}

// Data structures for formatting
type DeploymentInfo struct {
    StackName    string
    Region       string
    Account      string
    IsNew        bool
    DryRun       bool
    TemplateInfo TemplateInfo
}

type TemplateInfo struct {
    Path     string
    Size     int64
    S3URL    string
    Hash     string
}

type ChangesetInfo struct {
    Name         string
    Status       string
    Changes      []ChangeInfo
    Summary      ChangeSummary
    DangerLevel  DangerLevel
}

type ChangeInfo struct {
    Action       string
    ResourceType string
    LogicalID    string
    PhysicalID   string
    Replacement  string
    Details      []ChangeDetail
}

type ChangeDetail struct {
    Property    string
    OldValue    interface{}
    NewValue    interface{}
    ChangeType  string
}

type ChangeSummary struct {
    TotalChanges int
    Additions    int
    Modifications int
    Deletions    int
    Replacements int
}

type DangerLevel int

const (
    DangerLow DangerLevel = iota
    DangerMedium
    DangerHigh
    DangerCritical
)

type DriftResult struct {
    StackName       string
    DriftStatus     string
    TotalResources  int
    DriftedCount    int
    Resources       []DriftedResource
}

type DriftedResource struct {
    LogicalID    string
    PhysicalID   string
    ResourceType string
    DriftStatus  string
    Properties   []PropertyDrift
}

type PropertyDrift struct {
    PropertyPath string
    Expected     interface{}
    Actual       interface{}
    DriftType    string
}

type StackInfo struct {
    Name         string
    Status       string
    CreatedAt    time.Time
    UpdatedAt    *time.Time
    Parameters   []Parameter
    Outputs      []Output
    Tags         []Tag
}

type Parameter struct {
    Key   string
    Value string
}

type Output struct {
    Key         string
    Value       string
    Description string
    ExportName  string
}

type Tag struct {
    Key   string
    Value string
}

type ValidationError struct {
    Field   string
    Message string
    Code    string
}

type ValidationWarning struct {
    Field   string
    Message string
    Code    string
}

type ValidationSummary struct {
    ErrorCount   int
    WarningCount int
    InfoCount    int
}
```

### Step 2: Implement Console Output Handler

**File**: `cmd/ui/console/output.go`

```go
package console

import (
    "context"
    "fmt"
    "io"
    "os"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/fatih/color"
    format "github.com/ArjenSchwarz/go-output"
)

// OutputHandler implements console-based output
type OutputHandler struct {
    theme         ui.Theme
    format        ui.OutputFormat
    verbose       bool
    quiet         bool
    writer        io.Writer
    errorWriter   io.Writer
    outputSettings *format.OutputSettings
}

// NewOutputHandler creates a new console output handler
func NewOutputHandler(theme ui.Theme, outputSettings *format.OutputSettings) *OutputHandler {
    return &OutputHandler{
        theme:          theme,
        format:         ui.FormatTable,
        verbose:        false,
        quiet:          false,
        writer:         os.Stdout,
        errorWriter:    os.Stderr,
        outputSettings: outputSettings,
    }
}

// Success displays a success message
func (h *OutputHandler) Success(message string) {
    if h.quiet {
        return
    }

    formatted := h.theme.Success() + "✅ " + message + color.ResetString
    fmt.Fprintln(h.writer, formatted)
}

// Info displays an info message
func (h *OutputHandler) Info(message string) {
    if h.quiet {
        return
    }

    formatted := h.theme.Info() + "ℹ️  " + message + color.ResetString
    fmt.Fprintln(h.writer, formatted)
}

// Warning displays a warning message
func (h *OutputHandler) Warning(message string) {
    formatted := h.theme.Warning() + "⚠️  " + message + color.ResetString
    fmt.Fprintln(h.writer, formatted)
}

// Error displays an error message
func (h *OutputHandler) Error(message string) {
    formatted := h.theme.Error() + "❌ " + message + color.ResetString
    fmt.Fprintln(h.errorWriter, formatted)
}

// Debug displays a debug message
func (h *OutputHandler) Debug(message string) {
    if !h.verbose {
        return
    }

    formatted := h.theme.Debug() + "🔍 " + message + color.ResetString
    fmt.Fprintln(h.writer, formatted)
}

// Table displays data in table format
func (h *OutputHandler) Table(data interface{}, options ui.TableOptions) error {
    switch h.format {
    case ui.FormatTable:
        return h.renderTable(data, options)
    case ui.FormatJSON:
        return h.JSON(data)
    case ui.FormatCSV:
        return h.renderCSV(data, options)
    default:
        return h.renderTable(data, options)
    }
}

// JSON displays data in JSON format
func (h *OutputHandler) JSON(data interface{}) error {
    // Implementation using the existing go-output library
    output := format.OutputArray{Settings: h.outputSettings}
    // Convert data to OutputArray format
    // This would need to be implemented based on the data structure
    output.Write()
    return nil
}

// StartProgress starts a progress indicator
func (h *OutputHandler) StartProgress(message string) ui.ProgressIndicator {
    if h.quiet {
        return &NoOpProgressIndicator{}
    }

    return NewSpinner(message, h.theme, h.writer)
}

// SetStatus sets the current status message
func (h *OutputHandler) SetStatus(message string) {
    if h.quiet {
        return
    }

    formatted := h.theme.Muted() + "⏳ " + message + color.ResetString
    fmt.Fprintln(h.writer, formatted)
}

// Confirm prompts for user confirmation
func (h *OutputHandler) Confirm(message string) bool {
    return h.ConfirmWithDefault(message, false)
}

// ConfirmWithDefault prompts for user confirmation with a default value
func (h *OutputHandler) ConfirmWithDefault(message string, defaultValue bool) bool {
    if h.quiet {
        return defaultValue
    }

    prompt := NewConfirmationPrompt(message, defaultValue, h.theme, h.writer)
    return prompt.Ask()
}

// SetVerbose enables/disables verbose output
func (h *OutputHandler) SetVerbose(verbose bool) {
    h.verbose = verbose
}

// SetQuiet enables/disables quiet mode
func (h *OutputHandler) SetQuiet(quiet bool) {
    h.quiet = quiet
}

// SetOutputFormat sets the output format
func (h *OutputHandler) SetOutputFormat(format ui.OutputFormat) {
    h.format = format
}

// GetWriter returns the output writer
func (h *OutputHandler) GetWriter() io.Writer {
    return h.writer
}

// GetErrorWriter returns the error writer
func (h *OutputHandler) GetErrorWriter() io.Writer {
    return h.errorWriter
}

// Private helper methods

func (h *OutputHandler) renderTable(data interface{}, options ui.TableOptions) error {
    // Implementation using the existing go-output library
    // This would convert the data to the format expected by go-output
    return nil
}

func (h *OutputHandler) renderCSV(data interface{}, options ui.TableOptions) error {
    // CSV rendering implementation
    return nil
}

// NoOpProgressIndicator is a no-op implementation for quiet mode
type NoOpProgressIndicator struct{}

func (n *NoOpProgressIndicator) Update(message string)   {}
func (n *NoOpProgressIndicator) Success(message string)  {}
func (n *NoOpProgressIndicator) Error(message string)    {}
func (n *NoOpProgressIndicator) Stop()                   {}
```

### Step 3: Implement Progress Indicators

**File**: `cmd/ui/console/progress.go`

```go
package console

import (
    "fmt"
    "io"
    "sync"
    "time"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/fatih/color"
)

// Spinner implements a console spinner progress indicator
type Spinner struct {
    message     string
    theme       ui.Theme
    writer      io.Writer
    active      bool
    mutex       sync.Mutex
    stopCh      chan bool
    frames      []string
    frameIndex  int
}

// NewSpinner creates a new spinner progress indicator
func NewSpinner(message string, theme ui.Theme, writer io.Writer) *Spinner {
    return &Spinner{
        message: message,
        theme:   theme,
        writer:  writer,
        active:  false,
        stopCh:  make(chan bool),
        frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
    }
}

// Start begins the spinner animation
func (s *Spinner) Start() {
    s.mutex.Lock()
    if s.active {
        s.mutex.Unlock()
        return
    }
    s.active = true
    s.mutex.Unlock()

    go s.spin()
}

// Update changes the spinner message
func (s *Spinner) Update(message string) {
    s.mutex.Lock()
    s.message = message
    s.mutex.Unlock()
}

// Success stops the spinner with a success message
func (s *Spinner) Success(message string) {
    s.Stop()
    successMsg := s.theme.ProgressSuccess() + "✅ " + message + color.ResetString
    fmt.Fprintln(s.writer, successMsg)
}

// Error stops the spinner with an error message
func (s *Spinner) Error(message string) {
    s.Stop()
    errorMsg := s.theme.ProgressError() + "❌ " + message + color.ResetString
    fmt.Fprintln(s.writer, errorMsg)
}

// Stop stops the spinner
func (s *Spinner) Stop() {
    s.mutex.Lock()
    if !s.active {
        s.mutex.Unlock()
        return
    }
    s.active = false
    s.mutex.Unlock()

    s.stopCh <- true

    // Clear the current line
    fmt.Fprint(s.writer, "\r\033[K")
}

// spin runs the spinner animation
func (s *Spinner) spin() {
    ticker := time.NewTicker(100 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-s.stopCh:
            return
        case <-ticker.C:
            s.mutex.Lock()
            if !s.active {
                s.mutex.Unlock()
                return
            }

            frame := s.frames[s.frameIndex]
            spinnerColor := s.theme.ProgressSpinner()
            messageColor := s.theme.Info()

            output := fmt.Sprintf("\r%s%s%s %s%s%s",
                spinnerColor, frame, color.ResetString,
                messageColor, s.message, color.ResetString)

            fmt.Fprint(s.writer, output)

            s.frameIndex = (s.frameIndex + 1) % len(s.frames)
            s.mutex.Unlock()
        }
    }
}

// ProgressBar implements a progress bar indicator
type ProgressBar struct {
    total       int
    current     int
    width       int
    message     string
    theme       ui.Theme
    writer      io.Writer
    mutex       sync.Mutex
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int, message string, theme ui.Theme, writer io.Writer) *ProgressBar {
    return &ProgressBar{
        total:   total,
        current: 0,
        width:   50,
        message: message,
        theme:   theme,
        writer:  writer,
    }
}

// Update updates the progress bar
func (p *ProgressBar) Update(message string) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    p.current++
    p.message = message
    p.render()
}

// Success completes the progress bar with success
func (p *ProgressBar) Success(message string) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    p.current = p.total
    p.message = message
    p.render()
    fmt.Fprintln(p.writer)

    successMsg := p.theme.ProgressSuccess() + "✅ " + message + color.ResetString
    fmt.Fprintln(p.writer, successMsg)
}

// Error stops the progress bar with error
func (p *ProgressBar) Error(message string) {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    fmt.Fprintln(p.writer)
    errorMsg := p.theme.ProgressError() + "❌ " + message + color.ResetString
    fmt.Fprintln(p.writer, errorMsg)
}

// Stop stops the progress bar
func (p *ProgressBar) Stop() {
    p.mutex.Lock()
    defer p.mutex.Unlock()

    fmt.Fprintln(p.writer)
}

// render draws the progress bar
func (p *ProgressBar) render() {
    percentage := float64(p.current) / float64(p.total)
    filledWidth := int(percentage * float64(p.width))

    bar := "["
    for i := 0; i < p.width; i++ {
        if i < filledWidth {
            bar += "="
        } else if i == filledWidth {
            bar += ">"
        } else {
            bar += " "
        }
    }
    bar += "]"

    output := fmt.Sprintf("\r%s%s%s %3.0f%% %s%s%s",
        p.theme.ProgressSpinner(), bar, color.ResetString,
        percentage*100,
        p.theme.Info(), p.message, color.ResetString)

    fmt.Fprint(p.writer, output)
}
```

### Step 4: Implement User Prompts

**File**: `cmd/ui/console/prompts.go`

```go
package console

import (
    "bufio"
    "fmt"
    "io"
    "strings"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/fatih/color"
)

// ConfirmationPrompt handles user confirmation prompts
type ConfirmationPrompt struct {
    message      string
    defaultValue bool
    theme        ui.Theme
    writer       io.Writer
    reader       *bufio.Reader
}

// NewConfirmationPrompt creates a new confirmation prompt
func NewConfirmationPrompt(message string, defaultValue bool, theme ui.Theme, writer io.Writer) *ConfirmationPrompt {
    return &ConfirmationPrompt{
        message:      message,
        defaultValue: defaultValue,
        theme:        theme,
        writer:       writer,
        reader:       bufio.NewReader(os.Stdin),
    }
}

// Ask prompts the user for confirmation
func (c *ConfirmationPrompt) Ask() bool {
    defaultStr := "y/N"
    if c.defaultValue {
        defaultStr = "Y/n"
    }

    promptColor := c.theme.Info()
    emphasisColor := c.theme.Emphasis()

    prompt := fmt.Sprintf("%s🔔 %s%s [%s%s%s]: ",
        promptColor, c.message, color.ResetString,
        emphasisColor, defaultStr, color.ResetString)

    for {
        fmt.Fprint(c.writer, prompt)

        response, err := c.reader.ReadString('\n')
        if err != nil {
            return c.defaultValue
        }

        response = strings.ToLower(strings.TrimSpace(response))

        switch response {
        case "y", "yes":
            return true
        case "n", "no":
            return false
        case "":
            return c.defaultValue
        default:
            warningColor := c.theme.Warning()
            fmt.Fprintf(c.writer, "%sPlease enter 'y' for yes or 'n' for no.%s\n",
                warningColor, color.ResetString)
        }
    }
}

// SelectPrompt handles selection from multiple options
type SelectPrompt struct {
    message     string
    options     []string
    defaultIdx  int
    theme       ui.Theme
    writer      io.Writer
    reader      *bufio.Reader
}

// NewSelectPrompt creates a new selection prompt
func NewSelectPrompt(message string, options []string, defaultIdx int, theme ui.Theme, writer io.Writer) *SelectPrompt {
    return &SelectPrompt{
        message:    message,
        options:    options,
        defaultIdx: defaultIdx,
        theme:      theme,
        writer:     writer,
        reader:     bufio.NewReader(os.Stdin),
    }
}

// Ask prompts the user to select from options
func (s *SelectPrompt) Ask() (int, string) {
    promptColor := s.theme.Info()
    emphasisColor := s.theme.Emphasis()
    mutedColor := s.theme.Muted()

    fmt.Fprintf(s.writer, "%s%s%s\n", promptColor, s.message, color.ResetString)

    for i, option := range s.options {
        prefix := fmt.Sprintf("%s%d)%s", mutedColor, i+1, color.ResetString)
        if i == s.defaultIdx {
            fmt.Fprintf(s.writer, "%s %s%s%s (default)\n",
                prefix, emphasisColor, option, color.ResetString)
        } else {
            fmt.Fprintf(s.writer, "%s %s\n", prefix, option)
        }
    }

    prompt := fmt.Sprintf("%sSelect option [%s%d%s]: ",
        promptColor, emphasisColor, s.defaultIdx+1, color.ResetString)

    for {
        fmt.Fprint(s.writer, prompt)

        response, err := s.reader.ReadString('\n')
        if err != nil {
            return s.defaultIdx, s.options[s.defaultIdx]
        }

        response = strings.TrimSpace(response)

        if response == "" {
            return s.defaultIdx, s.options[s.defaultIdx]
        }

        // Parse selection
        var selectedIdx int
        if _, err := fmt.Sscanf(response, "%d", &selectedIdx); err == nil {
            selectedIdx-- // Convert to 0-based index
            if selectedIdx >= 0 && selectedIdx < len(s.options) {
                return selectedIdx, s.options[selectedIdx]
            }
        }

        warningColor := s.theme.Warning()
        fmt.Fprintf(s.writer, "%sPlease enter a number between 1 and %d.%s\n",
            warningColor, len(s.options), color.ResetString)
    }
}

// InputPrompt handles text input prompts
type InputPrompt struct {
    message      string
    defaultValue string
    validator    func(string) error
    theme        ui.Theme
    writer       io.Writer
    reader       *bufio.Reader
}

// NewInputPrompt creates a new input prompt
func NewInputPrompt(message string, defaultValue string, validator func(string) error, theme ui.Theme, writer io.Writer) *InputPrompt {
    return &InputPrompt{
        message:      message,
        defaultValue: defaultValue,
        validator:    validator,
        theme:        theme,
        writer:       writer,
        reader:       bufio.NewReader(os.Stdin),
    }
}

// Ask prompts the user for text input
func (i *InputPrompt) Ask() string {
    promptColor := i.theme.Info()
    emphasisColor := i.theme.Emphasis()

    var prompt string
    if i.defaultValue != "" {
        prompt = fmt.Sprintf("%s%s [%s%s%s]: ",
            promptColor, i.message, emphasisColor, i.defaultValue, color.ResetString)
    } else {
        prompt = fmt.Sprintf("%s%s: %s", promptColor, i.message, color.ResetString)
    }

    for {
        fmt.Fprint(i.writer, prompt)

        response, err := i.reader.ReadString('\n')
        if err != nil {
            return i.defaultValue
        }

        response = strings.TrimSpace(response)

        if response == "" {
            response = i.defaultValue
        }

        // Validate input
        if i.validator != nil {
            if err := i.validator(response); err != nil {
                warningColor := i.theme.Warning()
                fmt.Fprintf(i.writer, "%sInvalid input: %s%s\n",
                    warningColor, err.Error(), color.ResetString)
                continue
            }
        }

        return response
    }
}
```

### Step 5: Implement Default Theme

**File**: `cmd/ui/themes/default.go`

```go
package themes

import (
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/fatih/color"
)

// DefaultTheme implements the default UI theme
type DefaultTheme struct{}

// NewDefaultTheme creates a new default theme
func NewDefaultTheme() ui.Theme {
    return &DefaultTheme{}
}

// Success returns the success color
func (t *DefaultTheme) Success() string {
    return color.GreenString("")
}

// Info returns the info color
func (t *DefaultTheme) Info() string {
    return color.CyanString("")
}

// Warning returns the warning color
func (t *DefaultTheme) Warning() string {
    return color.YellowString("")
}

// Error returns the error color
func (t *DefaultTheme) Error() string {
    return color.RedString("")
}

// Debug returns the debug color
func (t *DefaultTheme) Debug() string {
    return color.MagentaString("")
}

// Emphasis returns the emphasis color
func (t *DefaultTheme) Emphasis() string {
    return color.New(color.Bold).SprintFunc()("")
}

// Muted returns the muted color
func (t *DefaultTheme) Muted() string {
    return color.New(color.Faint).SprintFunc()("")
}

// ProgressSpinner returns the progress spinner color
func (t *DefaultTheme) ProgressSpinner() string {
    return color.BlueString("")
}

// ProgressSuccess returns the progress success color
func (t *DefaultTheme) ProgressSuccess() string {
    return color.GreenString("")
}

// ProgressError returns the progress error color
func (t *DefaultTheme) ProgressError() string {
    return color.RedString("")
}

// TableHeader returns the table header color
func (t *DefaultTheme) TableHeader() string {
    return color.New(color.Bold, color.FgCyan).SprintFunc()("")
}

// TableBorder returns the table border color
func (t *DefaultTheme) TableBorder() string {
    return color.New(color.Faint).SprintFunc()("")
}

// TableData returns the table data color
func (t *DefaultTheme) TableData() string {
    return color.WhiteString("")
}

// MinimalTheme implements a minimal UI theme
type MinimalTheme struct{}

// NewMinimalTheme creates a new minimal theme
func NewMinimalTheme() ui.Theme {
    return &MinimalTheme{}
}

// All methods return empty strings for minimal theme
func (t *MinimalTheme) Success() string         { return "" }
func (t *MinimalTheme) Info() string            { return "" }
func (t *MinimalTheme) Warning() string         { return "" }
func (t *MinimalTheme) Error() string           { return "" }
func (t *MinimalTheme) Debug() string           { return "" }
func (t *MinimalTheme) Emphasis() string        { return "" }
func (t *MinimalTheme) Muted() string           { return "" }
func (t *MinimalTheme) ProgressSpinner() string { return "" }
func (t *MinimalTheme) ProgressSuccess() string { return "" }
func (t *MinimalTheme) ProgressError() string   { return "" }
func (t *MinimalTheme) TableHeader() string     { return "" }
func (t *MinimalTheme) TableBorder() string     { return "" }
func (t *MinimalTheme) TableData() string       { return "" }
```

### Step 6: Implement Deployment Formatter

**File**: `cmd/ui/formatters/deployment.go`

```go
package formatters

import (
    "fmt"
    "strings"
    "time"
    "github.com/ArjenSchwarz/fog/cmd/ui"
)

// DeploymentFormatter formats deployment-related output
type DeploymentFormatter struct {
    theme ui.Theme
}

// NewDeploymentFormatter creates a new deployment formatter
func NewDeploymentFormatter(theme ui.Theme) *DeploymentFormatter {
    return &DeploymentFormatter{
        theme: theme,
    }
}

// FormatDeploymentInfo formats deployment information
func (f *DeploymentFormatter) FormatDeploymentInfo(info ui.DeploymentInfo) string {
    var builder strings.Builder

    // Header
    if info.IsNew {
        method := "Deploying"
        if info.DryRun {
            method = f.theme.Warning() + "Dry run for deploying" + f.theme.Info()
        }
        builder.WriteString(fmt.Sprintf("%s%s%s new stack '%s%s%s' to region %s%s%s of account %s%s%s\n",
            f.theme.Info(), method, f.theme.Emphasis(),
            f.theme.Emphasis(), info.StackName, f.theme.Info(),
            f.theme.Emphasis(), info.Region, f.theme.Info(),
            f.theme.Emphasis(), info.Account, f.theme.Info()))
    } else {
        method := "Updating"
        if info.DryRun {
            method = f.theme.Warning() + "Dry run for updating" + f.theme.Info()
        }
        builder.WriteString(fmt.Sprintf("%s%s%s stack '%s%s%s' in region %s%s%s of account %s%s%s\n",
            f.theme.Info(), method, f.theme.Emphasis(),
            f.theme.Emphasis(), info.StackName, f.theme.Info(),
            f.theme.Emphasis(), info.Region, f.theme.Info(),
            f.theme.Emphasis(), info.Account, f.theme.Info()))
    }

    // Template information
    builder.WriteString(f.formatTemplateInfo(info.TemplateInfo))

    return builder.String()
}

// FormatChangeset formats changeset information
func (f *DeploymentFormatter) FormatChangeset(changeset ui.ChangesetInfo) string {
    var builder strings.Builder

    // Header
    builder.WriteString(fmt.Sprintf("%sChangeset: %s%s%s (%s%s%s)\n",
        f.theme.Info(),
        f.theme.Emphasis(), changeset.Name, f.theme.Info(),
        f.theme.Muted(), changeset.Status, f.theme.Info()))

    // Summary
    builder.WriteString(f.formatChangeSummary(changeset.Summary))

    // Danger level warning
    if changeset.DangerLevel >= ui.DangerMedium {
        builder.WriteString(f.formatDangerWarning(changeset.DangerLevel))
    }

    // Changes list
    if len(changeset.Changes) > 0 {
        builder.WriteString(f.formatChangesList(changeset.Changes))
    }

    return builder.String()
}

// FormatDriftResult formats drift detection results
func (f *DeploymentFormatter) FormatDriftResult(result ui.DriftResult) string {
    var builder strings.Builder

    // Header
    builder.WriteString(fmt.Sprintf("%sDrift Detection Results for %s%s%s\n",
        f.theme.Info(),
        f.theme.Emphasis(), result.StackName, f.theme.Info()))

    // Summary
    builder.WriteString(fmt.Sprintf("%sStatus: %s%s%s\n",
        f.theme.Info(),
        f.getDriftStatusColor(result.DriftStatus), result.DriftStatus, f.theme.Info()))

    builder.WriteString(fmt.Sprintf("%sTotal Resources: %s%d%s, Drifted: %s%d%s\n",
        f.theme.Info(),
        f.theme.Emphasis(), result.TotalResources, f.theme.Info(),
        f.theme.Emphasis(), result.DriftedCount, f.theme.Info()))

    // Drifted resources
    if len(result.Resources) > 0 {
        builder.WriteString(f.formatDriftedResources(result.Resources))
    }

    return builder.String()
}

// FormatStackInfo formats stack information
func (f *DeploymentFormatter) FormatStackInfo(stack ui.StackInfo) string {
    var builder strings.Builder

    // Header
    builder.WriteString(fmt.Sprintf("%sStack: %s%s%s\n",
        f.theme.Info(),
        f.theme.Emphasis(), stack.Name, f.theme.Info()))

    // Status and timing
    builder.WriteString(fmt.Sprintf("%sStatus: %s%s%s\n",
        f.theme.Info(),
        f.getStackStatusColor(stack.Status), stack.Status, f.theme.Info()))

    builder.WriteString(fmt.Sprintf("%sCreated: %s%s%s\n",
        f.theme.Info(),
        f.theme.Muted(), stack.CreatedAt.Format(time.RFC3339), f.theme.Info()))

    if stack.UpdatedAt != nil {
        builder.WriteString(fmt.Sprintf("%sUpdated: %s%s%s\n",
            f.theme.Info(),
            f.theme.Muted(), stack.UpdatedAt.Format(time.RFC3339), f.theme.Info()))
    }

    return builder.String()
}

// Private helper methods

func (f *DeploymentFormatter) formatTemplateInfo(info ui.TemplateInfo) string {
    var builder strings.Builder

    builder.WriteString(fmt.Sprintf("%sTemplate: %s%s%s",
        f.theme.Info(),
        f.theme.Emphasis(), info.Path, f.theme.Info()))

    if info.Size > 0 {
        builder.WriteString(fmt.Sprintf(" (%s%s%s)",
            f.theme.Muted(), formatFileSize(info.Size), f.theme.Info()))
    }

    if info.S3URL != "" {
        builder.WriteString(fmt.Sprintf("\n%sUploaded to: %s%s%s",
            f.theme.Info(),
            f.theme.Muted(), info.S3URL, f.theme.Info()))
    }

    builder.WriteString("\n")
    return builder.String()
}

func (f *DeploymentFormatter) formatChangeSummary(summary ui.ChangeSummary) string {
    return fmt.Sprintf("%sChanges: %s%d total%s (%s%d additions%s, %s%d modifications%s, %s%d deletions%s, %s%d replacements%s)\n",
        f.theme.Info(),
        f.theme.Emphasis(), summary.TotalChanges, f.theme.Info(),
        f.theme.Success(), summary.Additions, f.theme.Info(),
        f.theme.Warning(), summary.Modifications, f.theme.Info(),
        f.theme.Error(), summary.Deletions, f.theme.Info(),
        f.theme.Error(), summary.Replacements, f.theme.Info())
}

func (f *DeploymentFormatter) formatDangerWarning(level ui.DangerLevel) string {
    var message string
    var color string

    switch level {
    case ui.DangerMedium:
        message = "⚠️  This changeset contains potentially destructive changes"
        color = f.theme.Warning()
    case ui.DangerHigh:
        message = "🚨 This changeset contains destructive changes"
        color = f.theme.Error()
    case ui.DangerCritical:
        message = "💥 This changeset contains highly destructive changes"
        color = f.theme.Error()
    default:
        return ""
    }

    return fmt.Sprintf("%s%s%s\n", color, message, f.theme.Info())
}

func (f *DeploymentFormatter) formatChangesList(changes []ui.ChangeInfo) string {
    var builder strings.Builder

    builder.WriteString(fmt.Sprintf("%s\nChanges:\n", f.theme.Info()))

    for _, change := range changes {
        actionColor := f.getActionColor(change.Action)
        builder.WriteString(fmt.Sprintf("  %s%s%s %s%s%s (%s%s%s)\n",
            actionColor, change.Action, f.theme.Info(),
            f.theme.Emphasis(), change.LogicalID, f.theme.Info(),
            f.theme.Muted(), change.ResourceType, f.theme.Info()))

        if change.Replacement != "" && change.Replacement != "False" {
            builder.WriteString(fmt.Sprintf("    %s⚠️  Replacement: %s%s\n",
                f.theme.Warning(), change.Replacement, f.theme.Info()))
        }
    }

    return builder.String()
}

func (f *DeploymentFormatter) formatDriftedResources(resources []ui.DriftedResource) string {
    var builder strings.Builder

    builder.WriteString(fmt.Sprintf("%s\nDrifted Resources:\n", f.theme.Info()))

    for _, resource := range resources {
        statusColor := f.getDriftStatusColor(resource.DriftStatus)
        builder.WriteString(fmt.Sprintf("  %s%s%s %s%s%s (%s%s%s)\n",
            statusColor, resource.DriftStatus, f.theme.Info(),
            f.theme.Emphasis(), resource.LogicalID, f.theme.Info(),
            f.theme.Muted(), resource.ResourceType, f.theme.Info()))

        for _, prop := range resource.Properties {
            builder.WriteString(fmt.Sprintf("    %s%s%s: %s%v%s → %s%v%s\n",
                f.theme.Muted(), prop.PropertyPath, f.theme.Info(),
                f.theme.Error(), prop.Expected, f.theme.Info(),
                f.theme.Warning(), prop.Actual, f.theme.Info()))
        }
    }

    return builder.String()
}

func (f *DeploymentFormatter) getActionColor(action string) string {
    switch action {
    case "Add":
        return f.theme.Success()
    case "Modify":
        return f.theme.Warning()
    case "Remove", "Delete":
        return f.theme.Error()
    default:
        return f.theme.Info()
    }
}

func (f *DeploymentFormatter) getDriftStatusColor(status string) string {
    switch status {
    case "IN_SYNC":
        return f.theme.Success()
    case "MODIFIED":
        return f.theme.Warning()
    case "DELETED":
        return f.theme.Error()
    default:
        return f.theme.Info()
    }
}

func (f *DeploymentFormatter) getStackStatusColor(status string) string {
    if strings.Contains(status, "COMPLETE") {
        return f.theme.Success()
    } else if strings.Contains(status, "FAILED") || strings.Contains(status, "ROLLBACK") {
        return f.theme.Error()
    } else if strings.Contains(status, "PROGRESS") {
        return f.theme.Warning()
    }
    return f.theme.Info()
}

func formatFileSize(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

### Step 7: Update Deploy Command to Use UI Components

**File**: `cmd/commands/deploy/handler.go` (update from previous tasks)

```go
package deploy

import (
    "context"
    "fmt"
    "github.com/ArjenSchwarz/fog/cmd/services"
    "github.com/ArjenSchwarz/fog/cmd/ui"
    "github.com/ArjenSchwarz/fog/cmd/ui/formatters"
    "github.com/ArjenSchwarz/fog/config"
)

// Handler implements the deploy command logic with UI components
type Handler struct {
    flags             *groups.DeploymentFlags
    deploymentService services.DeploymentService
    config            *config.Config
    ui                ui.OutputHandler
    formatter         *formatters.DeploymentFormatter
}

// NewHandler creates a new deploy command handler with UI components
func NewHandler(flags *groups.DeploymentFlags, deploymentService services.DeploymentService, config *config.Config, ui ui.OutputHandler) *Handler {
    return &Handler{
        flags:             flags,
        deploymentService: deploymentService,
        config:            config,
        ui:                ui,
        formatter:         formatters.NewDeploymentFormatter(ui.GetTheme()),
    }
}

// Execute runs the deploy command with enhanced UI
func (h *Handler) Execute(ctx context.Context) error {
    // Convert flags to deployment options
    opts := services.DeploymentOptions{
        StackName:      h.flags.StackName,
        TemplateSource: h.flags.Template,
        ParameterFiles: parseCommaSeparated(h.flags.Parameters),
        TagFiles:       parseCommaSeparated(h.flags.Tags),
        DefaultTags:    h.flags.DefaultTags,
        Bucket:         h.flags.Bucket,
        ChangesetName:  h.flags.ChangesetName,
        DeploymentFile: h.flags.DeploymentFile,
        DryRun:         h.flags.Dryrun,
        NonInteractive: h.flags.NonInteractive,
        CreateOnly:     h.flags.CreateChangeset,
        DeployOnly:     h.flags.DeployChangeset,
    }

    // Show deployment info
    h.ui.Info("Starting deployment preparation...")

    // Prepare deployment with progress indicator
    progress := h.ui.StartProgress("Preparing deployment...")
    plan, err := h.deploymentService.PrepareDeployment(ctx, opts)
    if err != nil {
        progress.Error("Failed to prepare deployment")
        return fmt.Errorf("failed to prepare deployment: %w", err)
    }
    progress.Success("Deployment prepared successfully")

    // Show deployment information
    deployInfo := ui.DeploymentInfo{
        StackName: plan.StackName,
        Region:    "us-east-1", // Get from config
        Account:   "123456789", // Get from config
        IsNew:     plan.IsNewStack,
        DryRun:    opts.DryRun,
        TemplateInfo: ui.TemplateInfo{
            Path:  plan.Template.LocalPath,
            Size:  plan.Template.Size,
            S3URL: plan.Template.S3URL,
            Hash:  plan.Template.Hash,
        },
    }
    h.ui.Info(h.formatter.FormatDeploymentInfo(deployInfo))

    // Validate deployment
    progress = h.ui.StartProgress("Validating deployment...")
    if err := h.deploymentService.ValidateDeployment(ctx, plan); err != nil {
        progress.Error("Deployment validation failed")
        return fmt.Errorf("deployment validation failed: %w", err)
    }
    progress.Success("Deployment validation completed")

    // Create changeset
    progress = h.ui.StartProgress("Creating changeset...")
    changeset, err := h.deploymentService.CreateChangeset(ctx, plan)
    if err != nil {
        progress.Error("Failed to create changeset")
        return fmt.Errorf("failed to create changeset: %w", err)
    }
    progress.Success("Changeset created successfully")

    // Show changeset information
    changesetInfo := ui.ChangesetInfo{
        Name:   changeset.Name,
        Status: string(changeset.Status),
        // Convert changeset data to UI format
        Changes:     convertChangesToUI(changeset.Changes),
        Summary:     convertSummaryToUI(changeset),
        DangerLevel: assessDangerLevel(changeset.Changes),
    }
    h.ui.Info(h.formatter.FormatChangeset(changesetInfo))

    // Handle dry run
    if opts.DryRun {
        h.ui.Success("Dry run completed successfully")
        return nil
    }

    // Handle create-only mode
    if opts.CreateOnly {
        h.ui.Success("Changeset created successfully")
        h.ui.Info("Only created the changeset, will now terminate")
        return nil
    }

    // Get deployment confirmation
    if !opts.NonInteractive {
        if !h.ui.Confirm("Do you want to deploy this changeset?") {
            h.ui.Info("Deployment cancelled by user")
            return nil
        }
    }

    // Execute deployment
    progress = h.ui.StartProgress("Deploying changeset...")
    result, err := h.deploymentService.ExecuteDeployment(ctx, plan, changeset)
    if err != nil {
        progress.Error("Deployment failed")
        return fmt.Errorf("deployment failed: %w", err)
    }

    if result.Success {
        progress.Success("Deployment completed successfully")
        h.ui.Success("Stack deployed successfully!")

        // Show outputs if any
        if len(result.Outputs) > 0 {
            h.showOutputs(result.Outputs)
        }
    } else {
        progress.Error("Deployment completed with errors")
        return fmt.Errorf("deployment completed with errors: %s", result.ErrorMessage)
    }

    return nil
}

// ValidateFlags validates the command flags
func (h *Handler) ValidateFlags() error {
    return h.flags.Validate(context.Background())
}

// Private helper methods

func (h *Handler) showOutputs(outputs []types.Output) {
    h.ui.Info("\nStack Outputs:")

    tableData := make([]map[string]interface{}, len(outputs))
    for i, output := range outputs {
        tableData[i] = map[string]interface{}{
            "Key":         *output.OutputKey,
            "Value":       *output.OutputValue,
            "Description": getStringValue(output.Description),
            "ExportName":  getStringValue(output.ExportName),
        }
    }

    h.ui.Table(tableData, ui.TableOptions{
        Title:   "Stack Outputs",
        Headers: []string{"Key", "Value", "Description", "ExportName"},
        Style:   ui.TableStyleDefault,
    })
}

func convertChangesToUI(changes []types.Change) []ui.ChangeInfo {
    // Implementation to convert AWS SDK types to UI types
    result := make([]ui.ChangeInfo, len(changes))
    // ... conversion logic
    return result
}

func convertSummaryToUI(changeset *services.ChangesetResult) ui.ChangeSummary {
    // Implementation to create summary from changeset
    return ui.ChangeSummary{
        TotalChanges:  len(changeset.Changes),
        // ... calculate other values
    }
}

func assessDangerLevel(changes []types.Change) ui.DangerLevel {
    // Implementation to assess danger level based on changes
    return ui.DangerLow
}

func getStringValue(ptr *string) string {
    if ptr == nil {
        return ""
    }
    return *ptr
}

func parseCommaSeparated(input string) []string {
    if input == "" {
        return nil
    }
    // Implementation to split by comma and trim whitespace
    return []string{input} // Placeholder
}
```

## Files to Create/Modify

### New Files
- `cmd/ui/interfaces.go`
- `cmd/ui/console/output.go`
- `cmd/ui/console/progress.go`
- `cmd/ui/console/prompts.go`
- `cmd/ui/console/tables.go`
- `cmd/ui/themes/default.go`
- `cmd/ui/themes/minimal.go`
- `cmd/ui/formatters/deployment.go`
- `cmd/ui/formatters/drift.go`
- `cmd/ui/formatters/changeset.go`
- `cmd/ui/formatters/common.go`
- `cmd/ui/components/status.go`
- `cmd/ui/components/validation.go`

### Modified Files
- `cmd/commands/deploy/handler.go` - Use UI components
- `cmd/commands/deploy/command.go` - Inject UI handler
- `cmd/helpers.go` - Deprecate existing confirmation function
- `cmd/root.go` - Create UI handler factory

## Testing Strategy

### Unit Tests
- Test UI component interfaces and implementations
- Test theme color output
- Test progress indicator behavior
- Test prompt functionality with mock input
- Test formatter output formatting

### Integration Tests
- Test complete UI flow with real commands
- Test different output formats (table, JSON, CSV)
- Test theme switching
- Test quiet and verbose modes

### Test Files to Create
- `cmd/ui/console/output_test.go`
- `cmd/ui/console/progress_test.go`
- `cmd/ui/console/prompts_test.go`
- `cmd/ui/formatters/deployment_test.go`
- `cmd/ui/themes/default_test.go`

## Success Criteria

### Functional Requirements
- [ ] Consistent output formatting across all commands
- [ ] Working progress indicators and user prompts
- [ ] Theme support with color and minimal themes
- [ ] Table, JSON, and CSV output format support
- [ ] Enhanced deployment information display

### Quality Requirements
- [ ] Unit tests cover >85% of UI component code
- [ ] User-friendly and accessible interface patterns
- [ ] Performance impact is minimal
- [ ] Consistent behavior across different terminals

### User Experience Requirements
- [ ] Clear and informative progress indicators
- [ ] Intuitive confirmation prompts
- [ ] Well-formatted table and text output
- [ ] Appropriate use of colors and styling
- [ ] Support for different accessibility needs

## Migration Timeline

### Phase 1: Foundation
- Create UI interfaces and basic components
- Implement console output handler
- Create default theme

### Phase 2: Core Components
- Implement progress indicators and prompts
- Create deployment formatter
- Integrate with deploy command

### Phase 3: Enhancement
- Add additional formatters for other commands
- Implement additional themes
- Add advanced table formatting

## Dependencies

### Upstream Dependencies
- Task 1: Command Structure Reorganization (provides handler framework)
- Task 5: Error Handling (provides structured errors for formatting)

### Downstream Dependencies
None - this task enhances user experience without breaking existing functionality.

## Risk Mitigation

### Potential Issues
- Terminal compatibility issues with colors and unicode
- Performance overhead from formatted output
- Complexity in maintaining consistent formatting

### Mitigation Strategies
- Graceful fallback for terminals without color support
- Performance testing for output formatting overhead
- Clear style guide and formatting standards
- Theme system for different output preferences
