package config

import (
	"fmt"
	"strings"

	format "github.com/ArjenSchwarz/go-output"
	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
	SeparateTables bool
	DotColumns     *DotColumns
}

// DotColumns is used to set the From and To columns for the dot output format
type DotColumns struct {
	From string
	To   string
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

func (config *Config) PrintFailure(text interface{}) {
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	redbold.Println("")
	redbold.Printf("🚨 %v 🚨\r\n\r\n", text)
}

func (config *Config) PrintWarning(text string) {
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	redbold.Printf("%v\r\n", text)
}

func (config *Config) PrintInlineWarning(text string) {
	red := color.New(color.FgRed)
	redbold := red.Add(color.Bold)
	redbold.Printf("%v", text)
}

func (config *Config) PrintSuccess(text interface{}) {
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	greenbold.Println("")
	greenbold.Printf("✅ %v\r\n\r\n", text)
}

func (config *Config) PrintPositive(text string) {
	green := color.New(color.FgGreen)
	greenbold := green.Add(color.Bold)
	greenbold.Printf("%v\r\n", text)
}

func (config *Config) PrintInfo(text interface{}) {
	fmt.Println("")
	fmt.Printf("ℹ️  %v\r\n\r\n", text)
}

func (config *Config) PrintBold(text string) {
	bold := color.New(color.Bold)
	bold.Printf("%v\r\n", text)
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
	// settings.UseEmoji = config.GetBool("output.use-emoji")
	settings.SetOutputFormat(config.GetLCString("output"))
	// settings.OutputFile = config.GetLCString("output.file")
	// settings.ShouldAppend = config.GetBool("output.append")
	settings.TableStyle = format.TableStyles[config.GetString("table.style")]
	settings.TableMaxColumnWidth = config.GetInt("table.max-column-width")
	return settings
}
