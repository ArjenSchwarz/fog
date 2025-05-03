# Best Practices for Go Applications Using the Cobra Framework

This document outlines best practices for developing Go CLI applications using the [Cobra](https://github.com/spf13/cobra) framework. These recommendations are based on industry standards, the official Cobra documentation, and patterns observed in successful Cobra-based applications.

## Table of Contents

1. [Project Structure](#project-structure)
2. [Command Organization](#command-organization)
3. [Flag Management](#flag-management)
4. [Configuration with Viper](#configuration-with-viper)
5. [Error Handling](#error-handling)
6. [Testing](#testing)
7. [Documentation](#documentation)
8. [Performance Considerations](#performance-considerations)
9. [Examples and Patterns](#examples-and-patterns)

## Project Structure

### Recommended Directory Layout

```
myapp/
├── cmd/                    # Command implementations
│   ├── root.go             # Root command definition
│   ├── command1.go         # Command implementation
│   └── command2.go         # Command implementation
├── pkg/                    # Reusable packages
│   ├── feature1/           # Feature-specific code
│   └── feature2/           # Feature-specific code
├── internal/               # Private application code
├── main.go                 # Application entry point
├── go.mod                  # Go modules definition
└── go.sum                  # Go modules checksums
```

### Best Practices

- **Separate Commands from Business Logic**: Keep command implementations in the `cmd/` directory and business logic in `pkg/` or `internal/`.
- **One File per Command**: Each command should have its own file, named after the command.
- **Minimal `main.go`**: The `main.go` file should be minimal, typically just calling the root command's `Execute()` method.

```go
// main.go
package main

import "myapp/cmd"

func main() {
    cmd.Execute()
}
```

## Command Organization

### Command Hierarchy

- **Root Command**: Define a single root command that represents your application.
- **Subcommands**: Group related functionality into subcommands.
- **Command Nesting**: Limit nesting to 2-3 levels for usability.

### Command Definition Best Practices

- **Descriptive Use Field**: The `Use` field should clearly indicate command usage.
- **Concise Short Description**: The `Short` field should be a single line.
- **Comprehensive Long Description**: The `Long` field should provide detailed information.
- **Examples**: Include practical examples in the `Example` field.

```go
var exampleCmd = &cobra.Command{
    Use:     "example [arg]",
    Short:   "A brief description of the command",
    Long:    `A longer description that explains the command in detail
              and can span multiple lines.`,
    Example: `  # Example 1
  myapp example foo
  # Example 2
  myapp example bar --flag`,
    Run: func(cmd *cobra.Command, args []string) {
        // Command implementation
    },
}
```

### Command Registration

- **Initialize in `init()`**: Register commands in the `init()` function.
- **Consistent Pattern**: Use a consistent pattern for adding commands.

```go
func init() {
    rootCmd.AddCommand(exampleCmd)
}
```

## Flag Management

### Flag Types

- **Persistent Flags**: Use for flags that apply to all subcommands.
- **Local Flags**: Use for flags specific to a command.
- **Required Flags**: Mark flags as required when necessary.

### Flag Best Practices

- **Short and Long Forms**: Provide both short (single character) and long forms for common flags.
- **Default Values**: Set sensible default values for flags.
- **Flag Variables**: Define flag variables at the package level for accessibility.
- **Descriptive Help Text**: Provide clear help text for each flag.

```go
var (
    verbose bool
    outputFile string
)

func init() {
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
    exampleCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (required)")
    exampleCmd.MarkFlagRequired("output")
}
```

### Flag Groups

- **Group Related Flags**: Use flag groups for related options.
- **Mutually Exclusive Flags**: Document when flags are mutually exclusive.

## Configuration with Viper

### Integration with Cobra

- **Bind Flags to Viper**: Bind Cobra flags to Viper for configuration file support.
- **Configuration Precedence**: Establish a clear precedence: command-line flags > environment variables > config file > defaults.

```go
func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        // Search for config in default locations
        viper.AddConfigPath(".")
        viper.AddConfigPath("$HOME")
        viper.SetConfigName(".myapp")
    }

    viper.AutomaticEnv()
    viper.ReadInConfig()
}
```

### Configuration Best Practices

- **Multiple Config Formats**: Support multiple configuration formats (YAML, JSON, TOML).
- **Environment Variables**: Use environment variables for configuration.
- **Sensible Defaults**: Set sensible defaults for all configuration options.
- **Configuration Validation**: Validate configuration values.

## Error Handling

### Error Reporting

- **Consistent Error Format**: Use a consistent format for error messages.
- **Appropriate Error Level**: Use appropriate error levels (debug, info, warning, error).
- **User-Friendly Messages**: Provide user-friendly error messages.

```go
if err != nil {
    fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    os.Exit(1)
}
```

### Error Handling Patterns

- **Early Returns**: Use early returns for error conditions.
- **Error Wrapping**: Wrap errors to provide context.
- **Custom Error Types**: Define custom error types for specific error conditions.

```go
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

## Testing

### Command Testing

- **Table-Driven Tests**: Use table-driven tests for commands.
- **Mock Dependencies**: Mock external dependencies for testing.
- **Test Output**: Test command output and error conditions.

```go
func TestExampleCommand(t *testing.T) {
    tests := []struct {
        name     string
        args     []string
        wantErr  bool
        expected string
    }{
        {"basic", []string{"arg"}, false, "expected output"},
        {"error", []string{}, true, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewExampleCommand()
            output := &bytes.Buffer{}
            cmd.SetOut(output)
            cmd.SetArgs(tt.args)

            err := cmd.Execute()
            if (err != nil) != tt.wantErr {
                t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && output.String() != tt.expected {
                t.Errorf("Output = %v, want %v", output.String(), tt.expected)
            }
        })
    }
}
```

### Integration Testing

- **End-to-End Tests**: Write end-to-end tests for critical paths.
- **Test Fixtures**: Use test fixtures for complex test scenarios.

## Documentation

### Command Documentation

- **Help Text**: Provide comprehensive help text for all commands.
- **Man Pages**: Generate man pages for your application.
- **Markdown Documentation**: Generate markdown documentation.

```go
// Generate documentation
func main() {
    cmd := NewRootCommand()

    // Generate markdown docs
    if err := doc.GenMarkdownTree(cmd, "./docs"); err != nil {
        log.Fatal(err)
    }

    // Generate man pages
    header := &doc.GenManHeader{
        Title:   "MYAPP",
        Section: "1",
    }
    if err := doc.GenManTree(cmd, header, "./man"); err != nil {
        log.Fatal(err)
    }

    cmd.Execute()
}
```

### Self-Documentation

- **Command Discoverability**: Make commands and flags discoverable.
- **Consistent Help Format**: Use a consistent format for help text.
- **Examples**: Include examples in help text.

## Performance Considerations

### Startup Time

- **Lazy Loading**: Lazy-load expensive resources.
- **Minimize Dependencies**: Minimize external dependencies.
- **Efficient Initialization**: Optimize initialization code.

### Resource Usage

- **Resource Cleanup**: Clean up resources properly.
- **Memory Efficiency**: Be mindful of memory usage.
- **Concurrency**: Use concurrency appropriately.

## Examples and Patterns

### Command Factory Pattern

```go
// Command factory pattern
func NewExampleCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "example",
        Short: "Example command",
        Run:   runExample,
    }

    cmd.Flags().StringP("flag", "f", "", "Example flag")
    return cmd
}

func runExample(cmd *cobra.Command, args []string) {
    // Command implementation
}
```

### Middleware Pattern

```go
// Middleware pattern
func withLogging(fn func(*cobra.Command, []string)) func(*cobra.Command, []string) {
    return func(cmd *cobra.Command, args []string) {
        log.Printf("Running command: %s", cmd.Name())
        fn(cmd, args)
        log.Printf("Finished command: %s", cmd.Name())
    }
}

var exampleCmd = &cobra.Command{
    Use:   "example",
    Short: "Example command",
    Run:   withLogging(runExample),
}
```

### Command Composition

```go
// Command composition
func addSharedFlags(cmd *cobra.Command) {
    cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
    cmd.Flags().StringP("output", "o", "", "Output file")
}

func init() {
    addSharedFlags(cmd1)
    addSharedFlags(cmd2)
}
```

## Conclusion

Following these best practices will help you create well-structured, maintainable, and user-friendly CLI applications using the Cobra framework. Remember that these are guidelines, and you should adapt them to your specific project requirements.

## References

- [Official Cobra Documentation](https://github.com/spf13/cobra)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
