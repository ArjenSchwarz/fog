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
)

const (
	replacementConditional = "Conditional"
	replacementTrue        = "True"
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
	buildAndRenderChangeset(changeset, deployment, awsConfig)
}

func buildBasicStackInfo(deployment lib.DeployInfo, showDryRunInfo bool, awsConfig config.AWSConfig) *output.Builder {
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
		// Use emoji directly - EmojiTransformer will handle format-specific rendering
		dryRunValue := "❌"
		if deployment.IsDryRun {
			dryRunValue = "✅"
		}
		content["Is dry run"] = dryRunValue
	}

	// Build document using v2 Builder pattern
	return output.New().
		Table(
			stacktitle,
			[]map[string]any{content},
			output.WithKeys(keys...),
		)
}

// addStackInfoSection adds stack information table to the builder
func addStackInfoSection(
	builder *output.Builder,
	deployment lib.DeployInfo,
	awsConfig config.AWSConfig,
	changeset lib.ChangesetInfo,
	showDryRunInfo bool,
) *output.Builder {
	stacktitle := "CloudFormation stack information"
	keys := []string{"StackName", "Account", "Region", "Action"}
	if showDryRunInfo {
		keys = append(keys, "Is dry run")
	}
	// Add Console URL column if not a dry run
	if !deployment.IsDryRun {
		keys = append(keys, "ConsoleURL")
	}

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
		dryRunValue := "❌"
		if deployment.IsDryRun {
			dryRunValue = "✅"
		}
		content["Is dry run"] = dryRunValue
	}
	// Add console URL as a field for programmatic access
	if !deployment.IsDryRun {
		content["ConsoleURL"] = changeset.GenerateChangesetUrl(awsConfig)
	}

	// Add table to existing builder
	return builder.Table(
		stacktitle,
		[]map[string]any{content},
		output.WithKeys(keys...),
	)
}

// buildChangesetData prepares changeset data for rendering
// Returns: (changeRows, summaryContent, dangerRows)
//
// Note: ANSI codes are applied to Remove actions for visual emphasis in table output.
// The EnhancedColorTransformer in GetOutputOptions() automatically strips these codes
// from non-terminal formats (JSON, CSV, YAML), ensuring clean structured output.
func buildChangesetData(
	changes []lib.ChangesetChanges,
	hasModule bool,
) ([]map[string]any, map[string]any, []map[string]any) {
	bold := color.New(color.Bold).SprintFunc()

	// Initialize summary
	summarykeys, summaryContent := getChangesetSummaryTable()
	_ = summarykeys // Unused in this function but kept for consistency

	// Build change rows
	changeRows := make([]map[string]any, 0, len(changes))
	dangerRows := make([]map[string]any, 0)

	for _, change := range changes {
		// Build change row
		changeContent := make(map[string]any)
		action := change.Action
		if action == eventTypeRemove {
			action = bold(action)
		}
		changeContent["Action"] = action
		changeContent["Replacement"] = change.Replacement
		changeContent["CfnName"] = change.LogicalID
		changeContent["Type"] = change.Type
		changeContent["ID"] = change.ResourceID
		if hasModule {
			changeContent["Module"] = change.Module
		}
		changeRows = append(changeRows, changeContent)

		// Update summary
		addToChangesetSummary(&summaryContent, change)

		// Add to danger rows if dangerous
		if change.Action == eventTypeRemove ||
			change.Replacement == replacementConditional ||
			change.Replacement == replacementTrue {
			dangerContent := make(map[string]any)
			dangerContent["Action"] = action // Already bolded if Remove
			dangerContent["Replacement"] = change.Replacement
			dangerContent["CfnName"] = change.LogicalID
			dangerContent["Type"] = change.Type
			dangerContent["ID"] = change.ResourceID
			dangerContent["Details"] = change.GetDangerDetails()
			if hasModule {
				dangerContent["Module"] = change.Module
			}
			dangerRows = append(dangerRows, dangerContent)
		}
	}

	return changeRows, summaryContent, dangerRows
}

// addChangesetSections adds changeset tables to the builder
func addChangesetSections(
	builder *output.Builder,
	changeset lib.ChangesetInfo,
) *output.Builder {
	changesettitle := fmt.Sprintf("%v %v", texts.DeployChangesetMessageChanges, changeset.Name)
	changesetsummarytitle := fmt.Sprintf("Summary for %v", changeset.Name)

	// Handle empty changeset case
	if len(changeset.Changes) == 0 {
		// Add appropriate empty message as text for human-readable formats
		// This will appear in table/markdown/html but be handled appropriately in JSON/YAML/CSV
		builder = builder.Text(string(texts.DeployChangesetMessageNoResourceChanges) + "\n")
		return builder
	}

	// Build changes table data
	changeRows, summaryContent, dangerRows := buildChangesetData(changeset.Changes, changeset.HasModule)

	// Add changes table
	changesetkeys := []string{"Action", "CfnName", "Type", "ID", "Replacement"}
	if changeset.HasModule {
		changesetkeys = append(changesetkeys, "Module")
	}
	builder = builder.Table(
		changesettitle,
		changeRows,
		output.WithKeys(changesetkeys...),
	)

	// Add dangerous changes section
	if len(dangerRows) == 0 {
		// Use empty table instead of text message
		// Go-output v2 will handle this appropriately per format:
		// - Table: shows title with no rows (acceptable)
		// - JSON/YAML: includes table with empty data array
		// - CSV: no rows for this section
		dangerKeys := []string{"Action", "CfnName", "Type", "ID", "Replacement", "Details"}
		if changeset.HasModule {
			dangerKeys = append(dangerKeys, "Module")
		}
		builder = builder.Table(
			"Potentially destructive changes",
			[]map[string]any{},
			output.WithKeys(dangerKeys...),
		)
	} else {
		dangerKeys := []string{"Action", "CfnName", "Type", "ID", "Replacement", "Details"}
		if changeset.HasModule {
			dangerKeys = append(dangerKeys, "Module")
		}
		builder = builder.Table(
			"Potentially destructive changes",
			dangerRows,
			output.WithKeys(dangerKeys...),
		)
	}

	// Add summary table
	summarykeys, _ := getChangesetSummaryTable()
	builder = builder.Table(
		changesetsummarytitle,
		[]map[string]any{summaryContent},
		output.WithKeys(summarykeys...),
	)

	return builder
}

// buildAndRenderChangeset creates a complete changeset document and renders it
func buildAndRenderChangeset(
	changeset lib.ChangesetInfo,
	deployment lib.DeployInfo,
	awsConfig config.AWSConfig,
) {
	// Create single builder for entire document
	builder := output.New()

	// Add stack information section (includes console URL)
	builder = addStackInfoSection(builder, deployment, awsConfig, changeset, false)

	// Add changeset sections (changes, dangerous changes, summary)
	builder = addChangesetSections(builder, changeset)

	// Build and render the complete document
	doc := builder.Build()
	if err := renderDocument(context.Background(), doc); err != nil {
		fmt.Printf("ERROR: Failed to render changeset: %v\n", err)
		os.Exit(1)
	}
}

// printBasicStackInfo renders the basic stack information table
func printBasicStackInfo(deployment lib.DeployInfo, showDryRunInfo bool, awsConfig config.AWSConfig) {
	builder := buildBasicStackInfo(deployment, showDryRunInfo, awsConfig)
	doc := builder.Build()

	// Render to stderr using createStderrOutput helper
	out := createStderrOutput()
	if err := out.Render(context.Background(), doc); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to render stack info: %v\n", err)
	}
}

func showChangeset(changeset lib.ChangesetInfo, deployment lib.DeployInfo, awsConfig config.AWSConfig, optionalBuilder ...*output.Builder) {
	changesettitle := fmt.Sprintf("%v %v", texts.DeployChangesetMessageChanges, changeset.Name)
	changesetsummarytitle := fmt.Sprintf("Summary for %v", changeset.Name)

	// Use provided builder or create a new one
	var builder *output.Builder
	if len(optionalBuilder) > 0 {
		builder = optionalBuilder[0]
	} else {
		builder = output.New()
	}

	// Add changeset tables to the builder
	builder, hasChanges := buildChangesetDocument(builder, changesettitle, changesetsummarytitle, changeset.Changes, changeset.HasModule)

	if hasChanges {
		// Render the combined document to stderr using table format
		out := createStderrOutput()
		if err := out.Render(context.Background(), builder.Build()); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to render changeset: %v\n", err)
		}
	} else {
		// No changes, just render the stack info to stderr using table format
		out := createStderrOutput()
		if err := out.Render(context.Background(), builder.Build()); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to render stack info: %v\n", err)
		}
		fmt.Fprintln(os.Stderr, texts.DeployChangesetMessageNoResourceChanges)
	}

	if !deployment.IsDryRun {
		fmt.Fprintf(os.Stderr, "\n%v %v \r\n", texts.DeployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
	}
}

// buildChangesetDocument creates a document builder with changeset tables.
// Appends changeset tables to the provided builder.
// Returns false if there are no changes.
func buildChangesetDocument(builder *output.Builder, title string, summaryTitle string, changes []lib.ChangesetChanges, hasModule bool) (*output.Builder, bool) {
	if len(changes) == 0 {
		return builder, false
	}

	bold := color.New(color.Bold).SprintFunc()
	changesetkeys := []string{"Action", "CfnName", "Type", "ID", "Replacement"}
	if hasModule {
		changesetkeys = append(changesetkeys, "Module")
	}
	summarykeys, summaryContent := getChangesetSummaryTable()

	{
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

		// Add changeset changes table to builder
		builder = builder.Table(
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
			if change.Action == eventTypeRemove || change.Replacement == replacementConditional || change.Replacement == replacementTrue {
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
			// Add text instead of header to avoid uppercase and separators
			// Use "All changes are safe" instead of "No dangerous changes" to avoid emoji replacement
			builder = builder.Text(output.StylePositive("All changes are safe") + "\n")
		} else {
			builder = builder.Table(
				destructivechanges,
				dangerRows,
				output.WithKeys(dangerKeys...),
			)
		}

		// Add summary table
		builder = builder.Table(
			summaryTitle,
			[]map[string]any{summaryContent},
			output.WithKeys(summarykeys...),
		)

		return builder, true
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
	case replacementTrue:
		addToField(summaryContent, "Replacements", 1)
	case replacementConditional:
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
