package config

import (
	"strings"
	"time"

	format "github.com/ArjenSchwarz/go-output/v2"
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

// GetSeparator returns the appropriate separator string based on the output format
func (config *Config) GetSeparator() string {
	switch config.GetLCString("output") {
	case "table":
		return "\r\n"
	case "dot":
		return ","
	default:
		return ", "
	}
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

// NewOutputSettings creates a new OutputSettings object with configuration values applied
func (config *Config) NewOutputSettings() *format.OutputSettings {
	settings := format.NewOutputSettings()
	settings.UseEmoji = true
	settings.UseColors = true
	settings.SetOutputFormat(config.GetLCString("output"))
	settings.OutputFile = config.GetLCString("output-file")
	settings.OutputFileFormat = config.GetLCString("output-file-format")
	// settings.ShouldAppend = config.GetBool("output.append")
	settings.TableStyle = format.TableStyles[config.GetString("table.style")]
	settings.TableMaxColumnWidth = config.GetInt("table.max-column-width")
	return settings
}
