package config

import (
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/viper"
)

// Config holds the global configuration settings
type Config struct {
	SeparateTables bool
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

func (config *Config) GetSeparator() string {
	switch config.GetLCString("output") {
	case "table":
		return "\r\n"
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
