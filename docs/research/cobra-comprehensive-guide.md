# Comprehensive Guide to Cobra Applications in Go

This guide compiles best practices for developing command-line applications in Go using the Cobra framework, based on industry standards and real-world implementations.

## Table of Contents

1. [Project Structure](#project-structure)
2. [Command Design](#command-design)
3. [Configuration Management](#configuration-management)
4. [Error Handling](#error-handling)
5. [Testing Best Practices](#testing-best-practices)
6. [Documentation](#documentation)
7. [Advanced Patterns](#advanced-patterns)
8. [User Experience Best Practices](#user-experience-best-practices)
9. [Integration with Other Libraries](#integration-with-other-libraries)
10. [Performance Optimization](#performance-optimization)
11. [Alternatives to Cobra](#alternatives-to-cobra)

## Project Structure

Organize your CLI tool using a modular layout inspired by large-scale projects like Hugo and kubectl:

```
myapp/
├── cmd/           // Command definitions
│   ├── root.go    // Root command
│   └── serve.go   // Subcommands
├── internal/      // Private application logic
├── pkg/           // Reusable components
└── main.go        // Entry point
```

**Key considerations:**
- This separation enforces boundaries between command execution (cmd), business logic (internal), and shared utilities (pkg)
- Keep the main.go file minimal, primarily calling into the cmd package
- Place command-specific code in individual files within the cmd directory

## Command Design

Implement a hierarchical command structure using Cobra's natural workflow:

```go
// cmd/root.go
var rootCmd = &cobra.Command{
    Use:   "app",
    Short: "Core application",
    PersistentPreRun: func(cmd *cobra.Command, args []string) {
        // Shared initialization logic
    },
}

// cmd/serve.go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Run: func(cmd *cobra.Command, args []string) {
        // Server implementation
    },
}

func init() {
    rootCmd.AddCommand(serveCmd)
}
```

**Best practices:**
- Commands remain decoupled, enabling easier testing and maintenance
- Use `init()` functions to register subcommands with their parent commands
- Implement `PersistentPreRun` hooks for shared initialization logic
- Extract business logic from command Run functions into separate packages

## Configuration Management

Combine Cobra with Viper for enhanced flag handling:

```go
// cmd/root.go
var (
    cfgFile string
    rootCmd = &cobra.Command{
        Use:   "app",
        Short: "Application description",
    }
)

func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        viper.AddConfigPath(".")
        viper.SetConfigName("config")
    }

    viper.AutomaticEnv()
    viper.ReadInConfig()
}

// cmd/serve.go
func init() {
    serveCmd.Flags().IntP("port", "p", 8080, "Server port")
    viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
}
```

**Key practices:**
- Use environment variable prefixes (`APP_PORT`) alongside flags for flexible configuration
- Leverage Viper's ability to read from multiple sources (files, env vars, flags)
- Set sensible defaults for all configuration options
- Use persistent flags for global options and local flags for command-specific options

## Error Handling

Implement structured error reporting:

```go
var serveCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        if err := validateConfig(); err != nil {
            return fmt.Errorf("config validation failed: %w", err)
        }
        // Execution logic
        return nil
    },
}
```

**Best practices:**
- Use `RunE` instead of `Run` to return errors rather than calling `os.Exit()`
- Wrap errors with context using `fmt.Errorf` and `%w` verb
- Create custom error types for specific error conditions
- Implement consistent error formatting across all commands
- Use the root command to handle and format errors appropriately

## Testing Best Practices

### Separation of Concerns
Design your application so that the core command logic is in independent, testable functions outside Cobra's `Run` or `RunE` blocks:

```go
// cmd/serve.go
func runServer(port int) error {
    // Server implementation
    return nil
}

var serveCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        port, _ := cmd.Flags().GetInt("port")
        return runServer(port)
    },
}
```

### Dependency Injection
Use dependency injection for any services or resources needed by your commands:

```go
type ServerConfig struct {
    Port int
    Logger Logger
    DB Database
}

func NewServerCommand(config ServerConfig) *cobra.Command {
    return &cobra.Command{
        Use: "serve",
        RunE: func(cmd *cobra.Command, args []string) error {
            return runServer(config)
        },
    }
}
```

### Testing Command Output
To test Cobra commands directly:

```go
func TestServeCommand(t *testing.T) {
    cmd := NewServerCommand(mockConfig)
    buffer := new(bytes.Buffer)
    cmd.SetOut(buffer)

    err := cmd.Execute()
    if err != nil {
        t.Errorf("Expected no error, got %v", err)
    }

    output := buffer.String()
    if !strings.Contains(output, "Server started") {
        t.Errorf("Expected output to contain 'Server started', got %s", output)
    }
}
```

### Integration Testing
Complement unit tests with integration tests that execute commands as a user would:

```go
func TestCommandIntegration(t *testing.T) {
    // Setup test environment

    cmd := exec.Command("./myapp", "serve", "--port", "9000")
    output, err := cmd.CombinedOutput()

    // Assert on output and behavior
}
```

## Documentation

### Leverage Built-In Cobra Features
Cobra encourages documentation via the `Short` and `Long` fields:

```go
var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the application server",
    Long: `Start the application server on the specified port.
This command initializes all required resources and begins
listening for incoming connections.

Example:
  myapp serve --port 8080`,
    Run: func(cmd *cobra.Command, args []string) {
        // Implementation
    },
}
```

### Auto-Generated Help
Use Cobra's automatic help and usage output:

```go
func init() {
    rootCmd.SetHelpCommand(customHelpCommand)
    rootCmd.SetHelpTemplate(customHelpTemplate)
    rootCmd.SetUsageTemplate(customUsageTemplate)
}
```

### Generate Documentation
Use Cobra's built-in documentation generation:

```go
func main() {
    // Generate man pages
    if genMan := os.Getenv("GEN_MAN"); genMan != "" {
        header := &doc.GenManHeader{
            Title:   "MYAPP",
            Section: "1",
        }
        doc.GenManTree(rootCmd, header, "./man")
        os.Exit(0)
    }

    // Execute command
    rootCmd.Execute()
}
```

## Advanced Patterns

### Subcommand-Based Architecture
Structure your CLI by separating logic into subcommands:

```go
// cmd/root.go
rootCmd.AddCommand(serveCmd)
rootCmd.AddCommand(versionCmd)
rootCmd.AddCommand(configCmd)

// cmd/config.go
configCmd.AddCommand(configGetCmd)
configCmd.AddCommand(configSetCmd)
```

### Command Aliases and Strategy Pattern
Use command aliases for backward compatibility:

```go
var serveCmd = &cobra.Command{
    Use:     "serve",
    Aliases: []string{"server", "start"},
    Run:     runServer,
}
```

### Custom Help and Error Handling
Override default help messages and usage examples:

```go
rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
    // Custom help implementation
    fmt.Fprintf(cmd.OutOrStdout(), "Custom help for %s\n", cmd.Name())
    // Show contextual examples based on previous commands
})

rootCmd.SetUsageFunc(func(cmd *cobra.Command) error {
    // Custom usage implementation
    return nil
})
```

### Shell Autocompletion
Generate shell completion scripts:

```go
func init() {
    rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate completion script",
    Run: func(cmd *cobra.Command, args []string) {
        switch args[0] {
        case "bash":
            cmd.Root().GenBashCompletion(os.Stdout)
        case "zsh":
            cmd.Root().GenZshCompletion(os.Stdout)
        // Handle other shells
        }
    },
}
```

## User Experience Best Practices

### Clear and Consistent Command Structure
- Use verbs for commands (e.g., `create`, `delete`)
- Group related subcommands
- Avoid ambiguous naming

### Descriptive Flags and Parameters
- Each flag should have a clear name
- Avoid unnecessary abbreviations
- Include descriptive help strings
- Prefer POSIX-compliant flags (short and long versions)

```go
cmd.Flags().StringP("output", "o", "json", "Output format (json, yaml, table)")
```

### Automatic Help and Suggestions
Enable automatic help output and typo suggestions:

```go
rootCmd.SuggestionsMinimumDistance = 2
```

### Consistent User Feedback
Provide clear output for both successful and erroneous operations:

```go
func runCommand(cmd *cobra.Command, args []string) error {
    // Implementation

    if success {
        cmd.Println("Operation completed successfully")
        cmd.Println("Next steps: ...")
    }

    return nil
}
```

## Integration with Other Libraries

### Cobra and Viper Integration

```go
// cmd/root.go
func init() {
    cobra.OnInitialize(initConfig)
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        // Search for config in predefined locations
        viper.AddConfigPath("$HOME/.myapp")
        viper.AddConfigPath(".")
        viper.SetConfigName("config")
    }

    // Read environment variables with prefix
    viper.SetEnvPrefix("MYAPP")
    viper.AutomaticEnv()

    // Read config file
    if err := viper.ReadInConfig(); err == nil {
        fmt.Println("Using config file:", viper.ConfigFileUsed())
    }
}

// cmd/serve.go
func init() {
    serveCmd.Flags().IntP("port", "p", 8080, "Server port")
    viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
}

func runServer(cmd *cobra.Command, args []string) {
    port := viper.GetInt("port")
    // Use port value
}
```

### Integration with Logging Libraries

```go
// cmd/root.go
var (
    logLevel string
    logger   *zap.Logger
)

func init() {
    rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
    cobra.OnInitialize(initLogger)
}

func initLogger() {
    // Configure logger based on log level
    config := zap.NewProductionConfig()

    switch strings.ToLower(logLevel) {
    case "debug":
        config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
    // Other cases
    }

    logger, _ = config.Build()
    zap.ReplaceGlobals(logger)
}
```

## Performance Optimization

Leverage Go's concurrency model for I/O-bound tasks:

```go
func fetchDataConcurrently(urls []string) {
    var wg sync.WaitGroup
    results := make(chan Result, len(urls))

    for _, url := range urls {
        wg.Add(1)
        go func(u string) {
            defer wg.Done()
            results <- fetch(u)
        }(url)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    // Process results
    for result := range results {
        // Handle result
    }
}
```

## Alternatives to Cobra

| Option          | Use Case                          | Tradeoffs                         |
|-----------------|-----------------------------------|-----------------------------------|
| Standard `flag` | Simple tools with few commands    | Lacks subcommand support          |
| urfave/cli      | Alternative opinionated framework | Different design philosophy       |
| Kong            | Complex configuration needs       | Additional DSL learning curve     |

## Implementation Checklist

1. **Project Structure**
   - [ ] Organize code into cmd, internal, and pkg directories
   - [ ] Keep main.go minimal
   - [ ] Separate commands into individual files

2. **Command Design**
   - [ ] Define root command with persistent flags
   - [ ] Implement subcommands for specific functionality
   - [ ] Use command hooks for pre/post-execution logic

3. **Configuration**
   - [ ] Integrate Viper for configuration management
   - [ ] Support config files, environment variables, and flags
   - [ ] Set sensible defaults

4. **Error Handling**
   - [ ] Use RunE for error propagation
   - [ ] Implement consistent error formatting
   - [ ] Create custom error types as needed

5. **Testing**
   - [ ] Extract business logic for testability
   - [ ] Use dependency injection
   - [ ] Test command output and execution
   - [ ] Implement integration tests

6. **Documentation**
   - [ ] Provide Short and Long descriptions
   - [ ] Include usage examples
   - [ ] Generate man pages and completion scripts

7. **User Experience**
   - [ ] Use consistent command naming
   - [ ] Provide descriptive flags
   - [ ] Enable suggestions for typos
   - [ ] Implement clear user feedback

8. **Integration**
   - [ ] Combine Cobra with Viper
   - [ ] Integrate logging libraries
   - [ ] Use other Go libraries as needed

9. **Performance**
   - [ ] Optimize long-running operations
   - [ ] Use concurrency for I/O-bound tasks

10. **Distribution**
    - [ ] Cross-compile for multiple platforms
    - [ ] Package for distribution (e.g., Homebrew, apt)
    - [ ] Implement versioning
