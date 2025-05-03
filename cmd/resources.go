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
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

// resourcesListCmd represents the list command
var resourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all CloudFormation managed resources",
	Long: `This command let's you see all the resources managed by your CloudFormation templates.

The standard output shows the type, resource ID, and stack it's managed by. Verbose mode adds the logical ID in the CloudFormation stack and the status.
Using the stackname argument you can limit this to a specific stack using the stack's name or ID. If you provide a wildcard filter such as "*dev*" it will match all stacks that match that pattern.

Examples:

$ fog resource list
$ fog resource list --stackname my-awesome-stack
$ fog resource list --stackname "*awesome*"`,
	Run: listResources,
}

var resource_stackname *string

func init() {
	resourceGroupCmd.AddCommand(resourcesListCmd)
	resource_stackname = resourcesListCmd.Flags().StringP("stackname", "n", "", "Name, ID, or wildcard filter for the stack (optional)")
}

func listResources(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	resources := lib.GetResources(resource_stackname, awsConfig.CloudformationClient())
	keys := []string{"Type", "ID", "Stack"}
	if settings.GetBool("verbose") {
		keys = append(keys, []string{"LogicalID", "Status"}...)
	}
	subtitle := "All resources created by CloudFormation"
	if *resource_stackname != "" {
		subtitle = fmt.Sprintf("Resources for %v", *resource_stackname)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)
	output := format.OutputArray{Keys: keys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = title
	output.Settings.SortKey = "Type"
	for _, resource := range resources {
		content := make(map[string]interface{})
		content["Type"] = resource.Type
		content["ID"] = resource.ResourceID
		content["Stack"] = resource.StackName
		if settings.GetBool("verbose") {
			content["LogicalID"] = resource.LogicalID
			content["Status"] = resource.Status
		}
		holder := format.OutputHolder{Contents: content}
		output.AddHolder(holder)
	}
	output.Write()

}
