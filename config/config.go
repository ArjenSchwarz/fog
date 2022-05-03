package config

import (
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
}

func (config *Config) GetLCString(setting string) string {
	if viper.IsSet(setting) {
		return strings.ToLower(viper.GetString(setting))
	}
	return ""
}

func (config *Config) GetString(setting string) string {
	if viper.IsSet(setting) {
		return viper.GetString(setting)
	}
	return ""
}

func (config *Config) GetBool(setting string) bool {
	return viper.GetBool(setting)
}

func (config *Config) GetInt(setting string) int {
	if viper.IsSet(setting) {
		return viper.GetInt(setting)
	}
	return 0
}

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

func (config *Config) NewOutputSettings() *format.OutputSettings {
	settings := format.NewOutputSettings()
	settings.UseEmoji = true
	settings.UseColors = true
	settings.SetOutputFormat(config.GetLCString("output"))
	// settings.OutputFile = config.GetLCString("output.file")
	// settings.ShouldAppend = config.GetBool("output.append")
	settings.TableStyle = format.TableStyles[config.GetString("table.style")]
	settings.TableMaxColumnWidth = config.GetInt("table.max-column-width")
	return settings
}
