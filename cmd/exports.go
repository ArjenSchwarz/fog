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

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
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

var exportsFlags ExportsFlags

func init() {
	resourceGroupCmd.AddCommand(exportsCmd)
	exportsFlags.RegisterFlags(exportsCmd)
}

func listExports(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	exports := lib.GetExports(&exportsFlags.StackName, &exportsFlags.ExportName, awsConfig.CloudformationClient())

	// Build column keys based on verbose flag
	keys := []string{"Export", "Value"}
	if settings.GetBool("verbose") {
		keys = append(keys, "Imported By")
	}

	// Build title
	subtitle := "All exports"
	if exportsFlags.StackName != "" {
		subtitle = fmt.Sprintf("Exports for %v", exportsFlags.StackName)
	}
	title := fmt.Sprintf("%v in account %v for region %v", subtitle, awsConfig.AccountID, awsConfig.Region)

	// Build export data for v2 output
	data := make([]map[string]any, 0, len(exports))
	for _, resource := range exports {
		row := map[string]any{
			"Export": resource.ExportName,
			"Value":  resource.OutputValue,
		}
		if settings.GetBool("verbose") {
			// Pass array directly - v2 handles array rendering automatically
			row["Imported By"] = resource.ImportedBy
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

	// Create output with configured options
	out := output.NewOutput(settings.GetOutputOptions()...)
	if err := out.Render(context.Background(), doc); err != nil {
		failWithError(err)
	}
}
