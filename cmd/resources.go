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
	"context"
	"fmt"
	"slices"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

const (
	sortKeyType = "Type"
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

var resourcesFlags ResourcesFlags

func init() {
	resourceGroupCmd.AddCommand(resourcesListCmd)
	resourcesFlags.RegisterFlags(resourcesListCmd)
}

func listResources(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	resources := lib.GetResources(&resourcesFlags.StackName, awsConfig.CloudformationClient())

	// Build column keys based on verbose flag
	keys := []string{"Type", "ID", "Stack"}
	if settings.GetBool("verbose") {
		keys = append(keys, []string{"LogicalID", "Status"}...)
	}

	// Build title
	subtitle := "All resources created by CloudFormation"
	if resourcesFlags.StackName != "" {
		subtitle = fmt.Sprintf("Resources for %v", resourcesFlags.StackName)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)

	// Sort resources by Type
	slices.SortFunc(resources, func(a, b lib.CfnResource) int {
		if a.Type < b.Type {
			return -1
		}
		if a.Type > b.Type {
			return 1
		}
		return 0
	})

	// Build resource data for v2 output
	data := make([]map[string]any, 0, len(resources))
	for _, resource := range resources {
		row := map[string]any{
			"Type":  resource.Type,
			"ID":    resource.ResourceID,
			"Stack": resource.StackName,
		}
		if settings.GetBool("verbose") {
			row["LogicalID"] = resource.LogicalID
			row["Status"] = resource.Status
		}
		data = append(data, row)
	}

	// Create document using v2 Builder pattern
	doc := output.New().
		Table(
			title,
			data,
			output.WithKeys(keys...),
		).
		Build()

	// Render to console and file (if configured)
	if err := renderDocument(context.Background(), doc); err != nil {
		failWithError(err)
	}
}
