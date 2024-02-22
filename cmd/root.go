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
package cmd

import (
	"github.com/ArjenSchwarz/fog/config"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string
var settings = new(config.Config)
var outputsettings *format.OutputSettings

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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is fog.yaml in current directory, or $HOME/fog.yaml)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Give verbose output")
	rootCmd.PersistentFlags().String("output", "table", "Format for the output, currently supported are table, csv, json, and dot (for certain functions)")
	rootCmd.PersistentFlags().String("file", "", "Optional file to save the output to, in addition to stdout")
	rootCmd.PersistentFlags().String("file-format", "", "Optional format for the file, defaults to the same as output")
	rootCmd.PersistentFlags().String("profile", "", "Use a specific AWS profile")
	rootCmd.PersistentFlags().String("region", "", "Use a specific AWS region")
	rootCmd.PersistentFlags().String("timezone", "", "Specify a timezone you want to use for any times shown in output. By default it uses your system's timezone")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug mode, mainly for development purposes")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("output-file", rootCmd.PersistentFlags().Lookup("file"))
	viper.BindPFlag("output-file-format", rootCmd.PersistentFlags().Lookup("file-format"))
	viper.BindPFlag("profile", rootCmd.PersistentFlags().Lookup("profile"))
	viper.BindPFlag("region", rootCmd.PersistentFlags().Lookup("region"))
	viper.BindPFlag("timezone", rootCmd.PersistentFlags().Lookup("timezone"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))

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
		home, err := homedir.Dir()
		cobra.CheckErr(err)
		// Default to local config file
		viper.AddConfigPath(".")
		// Search config in home directory with name ".fog" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName("fog")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		// fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
