// Package config provides configuration management for the fog CLI tool.
//
// This package handles all configuration-related functionality including reading
// configuration files, managing AWS settings, and providing a unified interface
// for accessing configuration values throughout the application.
//
// # Configuration Sources
//
// The package supports multiple configuration sources with the following precedence
// (highest to lowest):
//   - Command-line flags
//   - Environment variables
//   - Configuration files (fog.yaml, fog.json, fog.toml)
//   - Default values
//
// # Configuration Files
//
// Fog looks for configuration files in the following locations:
//   - Current directory: ./fog.yaml, ./fog.json, ./fog.toml
//   - Home directory: ~/fog.yaml, ~/fog.json, ~/fog.toml
//   - Custom location: via --config flag
//
// The configuration file can contain settings for:
//   - AWS profile and region selection
//   - Output formatting preferences
//   - Template file locations and extensions
//   - Parameter and tag file locations
//   - Deployment file settings
//   - Logging configuration
//   - Table display options
//
// # AWS Configuration
//
// AWS-specific settings are managed through the AWSConfig type and include:
//   - Profile selection for AWS credentials
//   - Region override for API calls
//   - Retry and timeout configurations
//
// # Configuration Values
//
// The Config type provides methods to retrieve configuration values with
// type safety and default handling:
//   - GetString/GetLCString: String values (lowercase variant available)
//   - GetBool: Boolean values
//   - GetInt: Integer values
//   - GetStringSlice: Array values
//   - GetStringMap/GetStringMapString: Map values
//
// # Examples
//
// Reading a string configuration value:
//
//	cfg := &config.Config{}
//	outputFormat := cfg.GetString("output")
//
// Getting AWS configuration:
//
//	cfg := &config.Config{}
//	awsCfg, err := config.DefaultAwsConfig(*cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Checking a boolean flag:
//
//	if cfg.GetBool("verbose") {
//	    // Enable verbose output
//	}
package config

import (
	"log"
	"path/filepath"
	"strings"
	"time"

	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
}

// GetLCString gets a string configuration value and converts it to lowercase
func (config *Config) GetLCString(setting string) string {
	if viper.IsSet(setting) {
		return strings.ToLower(viper.GetString(setting))
	}
	return ""
}

// GetString gets a string configuration value
func (config *Config) GetString(setting string) string {
	if viper.IsSet(setting) {
		return viper.GetString(setting)
	}
	return ""
}

// GetStringSlice gets a string slice configuration value
func (config *Config) GetStringSlice(setting string) []string {
	if viper.IsSet(setting) {
		return viper.GetStringSlice(setting)
	}
	return []string{}
}

// GetBool gets a boolean configuration value
func (config *Config) GetBool(setting string) bool {
	return viper.GetBool(setting)
}

// GetInt gets an integer configuration value
func (config *Config) GetInt(setting string) int {
	if viper.IsSet(setting) {
		return viper.GetInt(setting)
	}
	return 0
}

// GetFieldOrEmptyValue returns the value if not empty, otherwise returns an appropriate empty value based on output format
func (config *Config) GetFieldOrEmptyValue(value string) string {
	if value != "" {
		return value
	}
	switch config.GetLCString("output") {
	case "json":
		return ""
	default:
		return "-"
	}
}

// GetTimezoneLocation gets the location object you can use in a time object
// based on the timezone specified in your settings.
func (config *Config) GetTimezoneLocation() *time.Location {
	location, err := time.LoadLocation(config.GetString("timezone"))
	if err != nil {
		panic(err)
	}
	return location
}

// GetTableFormat creates a v2 Format object for table output with configured style and max column width
func (config *Config) GetTableFormat() output.Format {
	styleName := config.GetString("table.style")
	maxWidth := config.GetInt("table.max-column-width")

	// v2 API accepts style name directly as string
	return output.TableWithStyleAndMaxColumnWidth(styleName, maxWidth)
}

// getFormatForOutput maps format name to v2 Format object
func (config *Config) getFormatForOutput(formatName string) output.Format {
	switch formatName {
	case "csv":
		return output.CSV()
	case "json":
		return output.JSON()
	case "dot":
		return output.DOT()
	case "markdown":
		return output.Markdown()
	case "html":
		return output.HTML()
	case "yaml":
		return output.YAML()
	default:
		return config.GetTableFormat()
	}
}

// GetOutputOptions creates v2 functional options from config settings.
// When console and file formats are the same, returns options for a single Output.
// When formats differ, only returns console options - use GetFileOutputOptions for file.
func (config *Config) GetOutputOptions() []output.OutputOption {
	opts := []output.OutputOption{}

	// Console output format and writer
	consoleFormatName := config.GetLCString("output")
	consoleFormat := config.getFormatForOutput(consoleFormatName)
	opts = append(opts, output.WithFormat(consoleFormat))
	opts = append(opts, output.WithWriter(output.NewStdoutWriter()))

	// File output - only add to same Output if formats match
	if outputFile := config.GetLCString("output-file"); outputFile != "" {
		fileFormatName := config.GetLCString("output-file-format")
		if fileFormatName == "" {
			fileFormatName = consoleFormatName
		}

		// Only combine when formats are the same
		// When formats differ, file output needs separate Output instance
		if fileFormatName == consoleFormatName {
			dir, pattern := filepath.Split(outputFile)
			if dir == "" {
				dir = "."
			}
			fileWriter, err := output.NewFileWriter(dir, pattern)
			if err != nil {
				log.Printf("Warning: Failed to create file writer for %s: %v", outputFile, err)
			} else {
				opts = append(opts, output.WithWriter(fileWriter))
			}
		}
	}

	// Transformers
	if config.GetBool("use-emoji") {
		opts = append(opts, output.WithTransformer(&output.EmojiTransformer{}))
	}
	if config.GetBool("use-colors") {
		opts = append(opts, output.WithTransformer(output.NewEnhancedColorTransformer()))
	}

	return opts
}

// GetFileOutputOptions returns output options for file output when file format
// differs from console format. Returns nil if no separate file output is needed.
func (config *Config) GetFileOutputOptions() []output.OutputOption {
	outputFile := config.GetLCString("output-file")
	if outputFile == "" {
		return nil
	}

	consoleFormatName := config.GetLCString("output")
	fileFormatName := config.GetLCString("output-file-format")
	if fileFormatName == "" {
		fileFormatName = consoleFormatName
	}

	// Only return separate options if formats differ
	if fileFormatName == consoleFormatName {
		return nil
	}

	opts := []output.OutputOption{}
	fileFormat := config.getFormatForOutput(fileFormatName)
	opts = append(opts, output.WithFormat(fileFormat))

	dir, pattern := filepath.Split(outputFile)
	if dir == "" {
		dir = "."
	}
	fileWriter, err := output.NewFileWriter(dir, pattern)
	if err != nil {
		log.Printf("Warning: Failed to create file writer for %s: %v", outputFile, err)
		return nil
	}
	opts = append(opts, output.WithWriter(fileWriter))

	// Transformers - file output typically doesn't need colors
	if config.GetBool("use-emoji") {
		opts = append(opts, output.WithTransformer(&output.EmojiTransformer{}))
	}

	return opts
}
