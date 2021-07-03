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

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/format"
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
$ fog exports --stackname *awesome*
`,
	Run: listExports,
}

func init() {
	rootCmd.AddCommand(exportsCmd)
	stackName = exportsCmd.Flags().StringP("stackname", "n", "", "Name, ID, or wildcard filter for the stack (optional)")
}

func listExports(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	exports := lib.GetExports(stackName, awsConfig.CloudformationClient())
	keys := []string{"Export", "Description", "Stack", "Value", "Imported"}
	subtitle := "All exports"
	if *stackName != "" {
		subtitle = fmt.Sprintf("Exports for %v", *stackName)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)
	output := format.OutputArray{Keys: keys, Title: title}
	output.SortKey = "Export"
	for _, resource := range exports {
		content := make(map[string]string)
		content["Export"] = resource.ExportName
		content["Value"] = resource.OutputValue
		content["Description"] = resource.Description
		content["Stack"] = resource.StackName
		if resource.Imported {
			content["Imported"] = "Yes"
		} else {
			content["Imported"] = "No"
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write(*settings)

}
