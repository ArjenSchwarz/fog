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
		return output.CSV
	case "json":
		return output.JSON
	case "dot":
		return output.DOT
	case "markdown":
		return output.Markdown
	case "html":
		return output.HTML
	case "yaml":
		return output.YAML
	default:
		return config.GetTableFormat()
	}
}

// GetOutputOptions creates v2 functional options from config settings
func (config *Config) GetOutputOptions() []output.OutputOption {
	opts := []output.OutputOption{}
	formats := []output.Format{}

	// Console output format
	consoleFormatName := config.GetLCString("output")
	consoleFormat := config.getFormatForOutput(consoleFormatName)
	formats = append(formats, consoleFormat)

	// Console writer
	opts = append(opts, output.WithWriter(output.NewStdoutWriter()))

	// File output if configured
	if outputFile := config.GetLCString("output-file"); outputFile != "" {
		fileFormatName := config.GetLCString("output-file-format")
		// If file format not specified, use console format name
		if fileFormatName == "" {
			fileFormatName = consoleFormatName
		}
		fileFormat := config.getFormatForOutput(fileFormatName)

		// Only add file format if different from console format
		addFileFormat := true
		if fileFormatName == consoleFormatName {
			addFileFormat = false
		}

		dir, pattern := filepath.Split(outputFile)
		// If no directory specified, default to current directory
		if dir == "" {
			dir = "."
		}
		fileWriter, err := output.NewFileWriter(dir, pattern)
		if err != nil {
			// Log warning message with file path and error details
			// Continue with console output even if file writer fails
			log.Printf("Warning: Failed to create file writer for %s: %v", outputFile, err)
		} else {
			if addFileFormat {
				formats = append(formats, fileFormat)
			}
			opts = append(opts, output.WithWriter(fileWriter))
		}
	}

	// Add all formats at once
	opts = append(opts, output.WithFormats(formats...))

	// Transformers
	if config.GetBool("use-emoji") {
		opts = append(opts, output.WithTransformer(&output.EmojiTransformer{}))
	}
	if config.GetBool("use-colors") {
		opts = append(opts, output.WithTransformer(&output.ColorTransformer{}))
	}

	return opts
}
