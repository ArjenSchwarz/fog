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
	"fmt"
	"strings"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// exportsCmd represents the exports command
var exportsCmd = &cobra.Command{
	Use:   "exports",
	Short: "Get a list of all CloudFormation exports",
	Long: `Provides an overview of all CloudFormation exports

By default the function will return all exports in the region for the account.
Using the stackname argument you can limit this to a specific stack using the stack's name or ID. If you provide a wildcard filter such as "*dev*" it will match all stacks that match that pattern.

Examples:

$ fog exports
$ fog exports --stackname my-awesome-stack
$ fog exports --stackname "*awesome*"
$ fog exports --export "*myproject*"
`,
	Run: listExports,
}

var exports_stackName *string
var export_exportName *string

func init() {
	resourceGroupCmd.AddCommand(exportsCmd)
	exports_stackName = exportsCmd.Flags().StringP("stackname", "n", "", "Name, ID, or wildcard filter for the stack (optional)")
	export_exportName = exportsCmd.Flags().StringP("export", "e", "", "Filter for the export name")
}

func listExports(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	exports := lib.GetExports(exports_stackName, export_exportName, awsConfig.CloudformationClient())
	keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
	if settings.GetBool("verbose") {
		keys = append(keys, "Imported By")
	}
	subtitle := "All exports"
	if *exports_stackName != "" {
		subtitle = fmt.Sprintf("Exports for %v", *exports_stackName)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = title
	output.Settings.SortKey = "Export"
	for _, resource := range exports {
		content := make(map[string]interface{})
		content["Export"] = resource.ExportName
		content["Value"] = resource.OutputValue
		content["Description"] = resource.Description
		content["Stack"] = resource.StackName
		if resource.Imported {
			content["Imported"] = "Yes"
		} else {
			content["Imported"] = "No"
		}
		if settings.GetBool("verbose") {
			content["Imported By"] = strings.Join(resource.ImportedBy, settings.GetSeparator())
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()

}
