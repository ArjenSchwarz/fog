/*
Copyright © 2021 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package cmd implements the command-line interface for fog using Cobra.
//
// This package contains all CLI commands, flags, middleware, and services for managing
// AWS CloudFormation stacks. The architecture is organized into several sub-packages:
//
//   - commands/: Individual command implementations (deploy, report, drift, etc.)
//   - flags/: Modular flag system with validation groups
//   - middleware/: Request validation, error handling, and recovery
//   - registry/: Command registration and dependency injection
//   - services/: Business logic services (deployment, AWS operations)
//   - ui/: Output formatting and user interaction
//
// # Commands
//
// The CLI provides commands organized into logical groups:
//
//   - Stack operations: deploy, describe, drift, history, dependencies, report
//   - Changeset operations: create, execute, describe
//   - Resource operations: list resources and their details
//
// Many commonly-used subcommands have root-level aliases for convenience
// (e.g., 'fog deploy' is an alias for 'fog stack deploy').
//
// # Configuration
//
// The package uses Viper for configuration management, supporting:
//
//   - Config files: fog.yaml, fog.json, or fog.toml in current directory or $HOME
//   - Environment variables: All settings can be overridden via environment
//   - Command-line flags: Persistent and command-specific flags
//
// Global flags available to all commands:
//
//   - --config: Specify config file location
//   - --verbose/-v: Enable verbose output
//   - --output: Set output format (table, csv, json, yaml, markdown, html, dot)
//   - --file: Save output to a file
//   - --profile: Use specific AWS profile
//   - --region: Use specific AWS region
//   - --timezone: Set timezone for time display
//   - --debug: Enable debug mode
//
// # Error Handling
//
// Commands use a structured error system with FogError types that provide
// consistent error codes and categories. Errors are handled through middleware
// that formats them appropriately for CLI output.
//
// # Examples
//
// Deploy a stack:
//
//	fog deploy mystack --template template.yaml
//
// Check for drift:
//
//	fog drift mystack
//
// Generate a report:
//
//	fog report mystack --output json
package cmd

import (
	"os"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var settings = new(config.Config)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fog",
	Short: "Fog is a tool for managing your CloudFormation stacks",
	Long: `Fog is a tool for managing your CloudFormation stacks.

Its aim is to make your life easier by handling some of the annoyances from the CLI. Look at the specific commands to see what they can do.

The timezone parameter supports both the shortform of a timezone (e.g. AEST) or the region/cityname (e.g. Australia/Melbourne)
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	// Initialize command groups
	InitGroups()

	// Add aliases for commonly used commands at the root level
	rootCmd.AddCommand(NewCommandAlias("deploy", "stack deploy", "Alias for 'stack deploy'"))
	rootCmd.AddCommand(NewCommandAlias("drift", "stack drift", "Alias for 'stack drift'"))
	rootCmd.AddCommand(NewCommandAlias("describe", "stack describe", "Alias for 'stack describe'"))
	rootCmd.AddCommand(NewCommandAlias("history", "stack history", "Alias for 'stack history'"))
	rootCmd.AddCommand(NewCommandAlias("dependencies", "stack dependencies", "Alias for 'stack dependencies'"))
	rootCmd.AddCommand(NewCommandAlias("report", "stack report", "Alias for 'stack report'"))
	rootCmd.AddCommand(NewCommandAlias("resources", "resource list", "Alias for 'resource list'"))
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is fog.yaml in current directory, or $HOME/fog.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	rootCmd.PersistentFlags().String("output", "table", "Format for the output, currently supported are table, csv, json, yaml, markdown, html, and dot (for certain functions)")
	rootCmd.PersistentFlags().String("file", "", "Optional file to save the output to, in addition to stdout")
	rootCmd.PersistentFlags().String("file-format", "", "Optional format for the file, defaults to the same as output")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific AWS profile")
	rootCmd.PersistentFlags().String("region", "", "Use a specific AWS region")
	rootCmd.PersistentFlags().String("timezone", "", "Specify a timezone you want to use for any times shown in output. By default it uses your system's timezone")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode, mainly for development purposes")

	if err := viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("output-file", rootCmd.PersistentFlags().Lookup("file")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("output-file-format", rootCmd.PersistentFlags().Lookup("file-format")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("timezone", rootCmd.PersistentFlags().Lookup("timezone")); err != nil {
		cobra.CheckErr(err)
	}
	if err := viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug")); err != nil {
		cobra.CheckErr(err)
	}

	// Default table settings
	viper.SetDefault("table.style", "Default")
	viper.SetDefault("table.max-column-width", 50)
	viper.SetDefault("timezone", "Local")

	// Default file structure settings
	viper.SetDefault("templates.extensions", []string{"", ".yaml", ".yml", ".templ", ".tmpl", ".template", ".json"})
	viper.SetDefault("templates.directory", "templates")
	viper.SetDefault("tags.extensions", []string{"", ".json"})
	viper.SetDefault("tags.directory", "tags")
	viper.SetDefault("tags.default", map[string]string{})
	viper.SetDefault("parameters.extensions", []string{"", ".json"})
	viper.SetDefault("parameters.directory", "parameters")
	viper.SetDefault("deployments.extensions", []string{"", ".yaml", ".yml", ".json"})
	viper.SetDefault("deployments.directory", []string{"."})
	viper.SetDefault("parameters.directory", "parameters")
	viper.SetDefault("rootdir", ".")

	viper.SetDefault("changeset.name-format", "fog-$TIMESTAMP")

	viper.SetDefault("logging.enabled", true)
	viper.SetDefault("logging.filename", "fog-deployments.log")
	viper.SetDefault("logging.show-previous", true)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		// Default to local config file
		viper.AddConfigPath(".")
		// Search config in home directory with name ".fog" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("fog")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	// Silently ignore error if config file not found
	_ = viper.ReadInConfig()
}
