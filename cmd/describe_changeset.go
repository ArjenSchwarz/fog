/*
Copyright © 2023 Arjen Schwarz <developer@arjen.eu>

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
	"os"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/texts"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// changesetCmd represents the changeset command
var changesetCmd = &cobra.Command{
	Use:   "changeset",
	Short: "Show the details of a changeset",
	Long: `Using this command you get a tabular overview of the provided changeset.

	You can provide the changeset either as the name of the stack + the name of the changeset,
	or you can provide the url using the url parameter.`,
	Run: describeChangeset,
}

func init() {
	describeCmd.AddCommand(changesetCmd)
	changesetCmd.Flags().StringVarP(&describeFlags.ChangesetName, "changeset", "c", "", "The name of the changeset")
	changesetCmd.Flags().StringVarP(&describeFlags.ChangesetUrl, "url", "u", "", "The URL of the changeset, will be parsed to get the stack and template name")
}

func describeChangeset(cmd *cobra.Command, args []string) {
	viper.Set("output", "table") // Enforce table output for deployments
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	if describeFlags.ChangesetName != "" && describeFlags.ChangesetUrl != "" {
		fmt.Println(output.StyleNegative("You can only use one of the following flags: changeset, url"))
		os.Exit(1)
	}
	if describeFlags.ChangesetUrl != "" {
		stackid, changesetid := lib.GetStackAndChangesetFromURL(describeFlags.ChangesetUrl, awsConfig.Region)
		describeFlags.StackName = stackid
		describeFlags.ChangesetName = changesetid
	}
	deployment.StackName = describeFlags.StackName
	// Set the changeset name to what's provided, otherwise fall back on the generated value
	deployment.ChangesetName = describeFlags.ChangesetName
	// We're calling an existing change set, so it can't be a dry run. Set explicitly.
	deployment.IsDryRun = false
	rawchangeset, err := deployment.GetChangeset(awsConfig.CloudformationClient())
	if err != nil {
		message := fmt.Sprintf(string(texts.DeployChangesetMessageRetrieveFailed), deployment.ChangesetName)
		fmt.Print(output.StyleNegative(message))
		os.Exit(1)
	}
	changeset := deployment.AddChangeset(rawchangeset)
	printBasicStackInfo(deployment, false, awsConfig)
	showChangeset(changeset, deployment, awsConfig)
}

func printBasicStackInfo(deployment lib.DeployInfo, showDryRunInfo bool, awsConfig config.AWSConfig) {
	stacktitle := "CloudFormation stack information"
	keys := []string{"StackName", "Account", "Region", "Action"}
	if showDryRunInfo {
		keys = append(keys, "Is dry run")
	}
	// TODO decide if I want to include the below fields in the output
	// , "StackStatus", "StackStatusReason", "CreationTime", "StackDescription"
	content := make(map[string]any)
	content["StackName"] = deployment.GetCleanedStackName()
	content["Account"] = awsConfig.GetAccountAliasID()
	content["Region"] = awsConfig.Region
	action := "Update"
	if deployment.IsNew {
		action = "Create"
	}
	content["Action"] = action
	if showDryRunInfo {
		content["Is dry run"] = deployment.IsDryRun
	}

	// Build document using v2 Builder pattern
	doc := output.New().
		Table(
			stacktitle,
			[]map[string]any{content},
			output.WithKeys(keys...),
		).
		Build()

	// Render using v2 Output
	out := output.NewOutput(settings.GetOutputOptions()...)
	if err := out.Render(context.Background(), doc); err != nil {
		fmt.Printf("ERROR: Failed to render stack info: %v\n", err)
	}
}

func showChangeset(changeset lib.ChangesetInfo, deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	changesettitle := fmt.Sprintf("%v %v", texts.DeployChangesetMessageChanges, changeset.Name)
	changesetsummarytitle := fmt.Sprintf("Summary for %v", changeset.Name)
	printChangeset(changesettitle, changesetsummarytitle, changeset.Changes, changeset.HasModule)

	if !deployment.IsDryRun {
		fmt.Printf("%v %v \r\n", texts.DeployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
	}
}

func printChangeset(title string, summaryTitle string, changes []lib.ChangesetChanges, hasModule bool) {
	bold := color.New(color.Bold).SprintFunc()
	changesetkeys := []string{"Action", "CfnName", "Type", "ID", "Replacement"}
	if hasModule {
		changesetkeys = append(changesetkeys, "Module")
	}
	summarykeys, summaryContent := getChangesetSummaryTable()

	if len(changes) == 0 {
		fmt.Println(texts.DeployChangesetMessageNoResourceChanges)
	} else {
		// Build changeset changes rows
		changeRows := make([]map[string]any, 0, len(changes))
		for _, change := range changes {
			content := make(map[string]any)
			action := change.Action
			if action == eventTypeRemove {
				action = bold(action)
			}
			content["Action"] = action
			content["Replacement"] = change.Replacement
			content["CfnName"] = change.LogicalID
			content["Type"] = change.Type
			content["ID"] = change.ResourceID
			if hasModule {
				content["Module"] = change.Module
			}
			addToChangesetSummary(&summaryContent, change)
			changeRows = append(changeRows, content)
		}

		// Build document with multiple tables using v2 Builder pattern
		doc := output.New().
			Table(
				title,
				changeRows,
				output.WithKeys(changesetkeys...),
			)

		// Add danger table
		destructivechanges := "Potentially destructive changes"
		dangerKeys := []string{"Action", "CfnName", "Type", "ID", "Replacement", "Details"}
		if hasModule {
			dangerKeys = append(dangerKeys, "Module")
		}
		dangerRows := make([]map[string]any, 0)
		for _, change := range changes {
			if change.Action == "Remove" || change.Replacement == "Conditional" || change.Replacement == "True" {
				content := make(map[string]any)
				action := change.Action
				if action == eventTypeRemove {
					action = bold(action)
				}
				content["Action"] = action
				content["Replacement"] = change.Replacement
				content["CfnName"] = change.LogicalID
				content["Type"] = change.Type
				content["ID"] = change.ResourceID
				content["Details"] = change.GetDangerDetails()
				if hasModule {
					content["Module"] = change.Module
				}
				dangerRows = append(dangerRows, content)
			}
		}

		if len(dangerRows) == 0 {
			// Add header instead of table
			doc = doc.Header(output.StylePositive("No dangerous changes"))
		} else {
			doc = doc.Table(
				destructivechanges,
				dangerRows,
				output.WithKeys(dangerKeys...),
			)
		}

		// Add summary table
		doc = doc.Table(
			summaryTitle,
			[]map[string]any{summaryContent},
			output.WithKeys(summarykeys...),
		)

		// Render all tables
		out := output.NewOutput(settings.GetOutputOptions()...)
		if err := out.Render(context.Background(), doc.Build()); err != nil {
			fmt.Printf("ERROR: Failed to render changeset: %v\n", err)
		}
	}
}

func addToChangesetSummary(summaryContent *map[string]any, change lib.ChangesetChanges) {
	addToField(summaryContent, "Total", 1)
	switch change.Action {
	case "Add":
		addToField(summaryContent, "Added", 1)
	case "Remove":
		addToField(summaryContent, "Removed", 1)
	case "Modify":
		addToField(summaryContent, "Modified", 1)
	}
	switch change.Replacement {
	case "True":
		addToField(summaryContent, "Replacements", 1)
	case "Conditional":
		addToField(summaryContent, "Conditionals", 1)
	}
}

func getChangesetSummaryTable() ([]string, map[string]any) {
	summarykeys := []string{"Total", "Added", "Removed", "Modified", "Replacements", "Conditionals"}
	summaryContent := make(map[string]any)
	for _, key := range summarykeys {
		summaryContent[key] = 0
	}
	return summarykeys, summaryContent
}
