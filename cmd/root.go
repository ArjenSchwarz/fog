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
//     variables named after the uppercased key with dots and dashes replaced
//     by underscores (e.g. table.max-column-width -> TABLE_MAX_COLUMN_WIDTH)
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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

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
	// The README and user guide document `fog demo ...`, so keep it reachable at the root
	rootCmd.AddCommand(NewCommandAlias("demo", "util demo", "Alias for 'util demo'"))

	// Enables the conventional `fog --version` flag alongside `fog util version`.
	// Version is read here (not at declaration) so build-time ldflags values are used.
	rootCmd.Version = Version

	// Must run after the root-level commands above are registered, because
	// Cobra only creates the help command for a root that has subcommands.
	initGeneratedCommands(rootCmd)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is fog.yaml in current directory, or $HOME/fog.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	rootCmd.PersistentFlags().String("output", "table", "Format for the output, currently supported are table, csv, json, yaml, markdown, html, and dot (for certain functions)")
	rootCmd.PersistentFlags().String("file", "", "Optional file to save the output to, in addition to stdout")
	rootCmd.PersistentFlags().String("file-format", "", "Optional format for the file, defaults to the same as output")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific AWS profile")
	rootCmd.PersistentFlags().String("region", "", "Use a specific AWS region")
	rootCmd.PersistentFlags().String("timezone", "", "Specify a timezone you want to use for any times shown in output. By default it uses your system's timezone")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode, mainly for development purposes")

	bindPersistentFlags()

	// Default table settings
	viper.SetDefault("table.style", "Default")
	viper.SetDefault("table.max-column-width", 50)
	viper.SetDefault("timezone", "Local")

	// Default file structure settings
	viper.SetDefault("templates.extensions", []string{"", extYAML, ".yml", ".templ", ".tmpl", ".template", extJSON})
	viper.SetDefault("templates.directory", "templates")
	viper.SetDefault("tags.extensions", []string{"", extJSON})
	viper.SetDefault("tags.directory", "tags")
	viper.SetDefault("tags.default", map[string]string{})
	viper.SetDefault("parameters.extensions", []string{"", extJSON})
	viper.SetDefault("parameters.directory", "parameters")
	viper.SetDefault("deployments.extensions", []string{"", extYAML, ".yml", extJSON})
	viper.SetDefault("deployments.directory", ".")
	viper.SetDefault("rootdir", ".")

	viper.SetDefault("changeset.name-format", "fog-$TIMESTAMP")

	viper.SetDefault("logging.enabled", true)
	viper.SetDefault("logging.filename", "fog-deployments.log")
	viper.SetDefault("logging.show-previous", true)
}

// bindPersistentFlags binds the root persistent flags to their config keys so
// an explicitly passed flag takes precedence over environment variables and
// config file values.
func bindPersistentFlags() {
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
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cobra.CheckErr(loadConfigFile(cfgFile))
}

// loadConfigFile configures viper sources and reads the config file. An explicit
// --config path (cfgFile != "") must exist and be readable, so its read errors
// are returned. For the implicit default lookup a missing file is tolerated.
func loadConfigFile(cfgFile string) error {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		// Default to local config file
		viper.AddConfigPath(".")
		// Search config in home directory with name ".fog" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("fog")
	}

	// Config keys contain dots (nesting) and dashes, neither of which can be
	// used in shell variable names. Map both to underscores so that, for
	// example, table.max-column-width can be set via TABLE_MAX_COLUMN_WIDTH.
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		// Tolerate a missing file only for the implicit default lookup.
		if cfgFile == "" && errors.As(err, &notFound) {
			return validateDefaultTags()
		}
		return err
	}
	return validateDefaultTags()
}

// validateDefaultTags rejects a tags.default value that Viper cannot turn into
// a map. An environment variable (TAGS_DEFAULT) always arrives as a string and
// replaces the configured map entirely; Viper only parses it when it is a JSON
// object and otherwise returns an empty map, which would silently drop every
// configured default tag. Values must be JSON strings too: the deploy
// consumer (GetStringMapString) turns numbers, booleans and null into empty
// tag values. Failing here surfaces the mistake instead.
func validateDefaultTags() error {
	raw, ok := viper.Get("tags.default").(string)
	if !ok {
		return nil
	}
	const hint = `must be a JSON object with string values such as {"Environment":"production","CostCenter":"123"}`
	var tags map[string]any
	if err := json.Unmarshal([]byte(raw), &tags); err != nil || tags == nil {
		return fmt.Errorf("invalid tags.default value %q (from TAGS_DEFAULT or the config file): %s", raw, hint)
	}
	for _, key := range slices.Sorted(maps.Keys(tags)) {
		if _, isString := tags[key].(string); !isString {
			return fmt.Errorf("invalid tags.default value for tag %q (from TAGS_DEFAULT or the config file): got %s, %s", key, jsonTypeName(tags[key]), hint)
		}
	}
	return nil
}

// jsonTypeName names the JSON type of a decoded value for error messages.
func jsonTypeName(value any) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", value)
	}
}
