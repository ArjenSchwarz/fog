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
	"sort"
	"strings"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/ArjenSchwarz/go-output/mermaid"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var reportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate a report about CloudFormation deployments",
	Long: `Generates a report displaying a summary of the events taking place during
a CloudFormation deployment. Events are grouped by action taken on the stack and the
affected resource. When choosing either html or markdown format a mermaid diagram
showing the timelines is added.

You can provide a file to be written to using --file, or ask for the result to be
stored on an S3 bucket with --s3bucket. When providing the s3bucket parameter it
will automatically generate a filename consisting of a prefix with the provided
stackname (or all-stacks if not provided) and a datestamp of when it was taken.
When you use --file, it supports the below placeholders. Make sure you use single
quotes around the filename to ensure it doesn't get substituted.
- $TIMESTAMP
- $REGION
- $ACCOUNTID
- $STACKNAME


Example:

  $ fog report --stackname sg-example --output markdown
# Fog report for stack sg-example

## Stack sg-example
#### Metadata of sg-example - Create event - Started 2021-09-13T17:33:46+10:00

| Stack | Account | Region | Type | Start time | Duration | Success |
| --- | --- | --- | --- | --- | --- | --- |
| sg-example | ignoreme-demo (1234567890) | ap-southeast-2 | Create | 2021-09-13T17:33:46+10:00 | 18s | ✅ |

#### Events of sg-example - Create event - Started 2021-09-13T17:33:46+10:00

| Action | CfnName | Type | ID | Start time | Duration | Success |
| --- | --- | --- | --- | --- | --- | --- |
| Add | Examplegroup | AWS::EC2::SecurityGroup |  | 2021-09-13T17:33:57+10:00 | 6s | ✅ |

` + "```mermaid" + `
gantt
	title Visual timeline of sg-example - Create event - Started 2021-09-13T17:33:46+10:00
	dateFormat HH:mm:ss
	axisFormat %H:%M:%S
	Examplegroup	:17:33:57 , 6s
` + "```",
	Run: stackReport,
}

var report_StackName *string
var report_Outputfile *string
var report_TargetBucket *string
var report_LatestOnly *bool
var report_FrontMatter *bool

func init() {
	rootCmd.AddCommand(reportCmd)
	report_StackName = reportCmd.Flags().StringP("stackname", "n", "", "The name for the stack")
	report_Outputfile = reportCmd.Flags().String("file", "", "Optional file to save the output to. Supports placeholders, see --help for details")
	report_LatestOnly = reportCmd.Flags().Bool("latest", false, "Only show the latest event")
	report_TargetBucket = reportCmd.Flags().String("s3bucket", "", "Optional s3 bucket the output should be saved to. Filename is autogenerated (providedStackname/date.extension) unless specified by --file")
	report_FrontMatter = reportCmd.Flags().Bool("frontmatter", false, "Add frontmatter to the report. Only works for single events. Markdown only.")
}

func stackReport(cmd *cobra.Command, args []string) {
	awsConfig := config.DefaultAwsConfig(*settings)
	outputsettings = getReportOutputSettingsFromCli(awsConfig)
	generateReport(awsConfig)
}

// generateReport creates the complete report
func generateReport(awsConfig config.AWSConfig) {
	mainoutput := format.OutputArray{Keys: []string{}, Settings: outputsettings}
	stacks, err := lib.GetCfnStacks(report_StackName, awsConfig.CloudformationClient())
	if *report_FrontMatter && outputsettings.OutputFormat == "markdown" {
		mainoutput.Settings.FrontMatter = generateFrontMatter(stacks, awsConfig)
	}

	if err != nil {
		panic(err)
	}
	if len(stacks) > 1 {
		mainoutput.Settings.HasTOC = true
	}
	// Hacky way to sort the stack
	stackskeys := make([]string, 0, len(stacks))
	for stackkey := range stacks {
		stackskeys = append(stackskeys, stackkey)
	}
	sort.Strings(stackskeys)

	for _, stackkey := range stackskeys {
		generateStackReport(stacks[stackkey], mainoutput, awsConfig)
	}
	// Set the title for the output file that we actually want
	latestText := ""
	if *report_LatestOnly {
		latestText = "Last event only."
	}
	if *report_StackName == "" {
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for account %s. %s", awsConfig.GetAccountAliasID(), latestText)
	} else if strings.Contains(*report_StackName, "*") {
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for stacks matching '%s'. %s", *report_StackName, latestText)
	} else {
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for stack %s. %s", *report_StackName, latestText)
	}
	mainoutput.Write()
}

func getReportOutputSettingsFromCli(awsConfig config.AWSConfig) *format.OutputSettings {
	settings := settings.NewOutputSettings()
	settings.OutputFile = reportPlaceholderParser(*report_Outputfile, *report_StackName, awsConfig)
	if *report_TargetBucket != "" {
		targetpath := settings.OutputFile
		if targetpath == "" {
			targetpath = cleanStackName(*report_StackName) + "/" + time.Now().Format(time.RFC3339) + settings.GetDefaultExtension()
		}
		settings.SetS3Bucket(awsConfig.S3Client(), *report_TargetBucket, targetpath)
	}
	settings.SeparateTables = true
	return settings
}

func reportPlaceholderParser(value string, stackname string, awsConfig config.AWSConfig) string {
	value = strings.Replace(value, "$TIMESTAMP", time.Now().Local().Format("2006-01-02T15-04-05"), -1)
	value = strings.Replace(value, "$STACKNAME", cleanStackName(stackname), -1)
	value = strings.Replace(value, "$REGION", awsConfig.Region, -1)
	value = strings.Replace(value, "$ACCOUNTID", awsConfig.AccountID, -1)
	return value
}

// getReportMermaidSettings generates the outputsettings we want for the report
func getReportMermaidSettings() *format.OutputSettings {
	outputSettings := settings.NewOutputSettings()
	outputSettings.SetOutputFormat("mermaid")
	outputSettings.MermaidSettings.ChartType = "ganttchart"
	switch outputsettings.OutputFormat {
	case "markdown":
		outputSettings.MermaidSettings.AddMarkdown = true
	case "html":
		outputSettings.MermaidSettings.AddHTML = true
	}
	return outputSettings
}

// generateStackReport creates the report for the provided stack
func generateStackReport(stack lib.CfnStack, mainoutput format.OutputArray, awsConfig config.AWSConfig) {
	mainoutput.AddHeader(fmt.Sprintf("Stack %s", stack.Name))
	events, err := stack.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		panic(err)
	}
	for counter, event := range events {
		if *report_LatestOnly && counter+1 < len(events) {
			continue
		}
		// Create metadata table
		metadataoutput := createMetadataTable(stack, event, awsConfig, true)
		metadataoutput.AddToBuffer()

		// Create outputarray for table
		output := createTableOutput(stack, event)
		// Create OutputArray for mermaid diagram
		mermaidoutput := createMermaidOutput(stack, event)

		for _, resource := range event.ResourceEvents {
			// Add resource to both the table and mermaid
			output.AddHolder(createTableResourceHolder(resource, event))
			mermaidoutput.AddHolder(createMermaidResourceHolder(resource))
		}
		output.AddToBuffer()
		// Only add mermaid to the buffer if the output needs it
		if mainoutput.Settings.OutputFormat == "markdown" || mainoutput.Settings.OutputFormat == "html" {
			mermaidoutput.AddToBuffer()
		}
	}
}

func generateFrontMatter(stacks map[string]lib.CfnStack, awsConfig config.AWSConfig) map[string]string {
	result := make(map[string]string)
	for _, stack := range stacks {
		events, err := stack.GetEvents(awsConfig.CloudformationClient())
		if err != nil {
			panic(err)
		}
		for _, event := range events {
			result["account"] = awsConfig.AccountID
			result["accountalias"] = awsConfig.GetAccountAliasID()
			result["region"] = awsConfig.Region
			result["stack"] = stack.Name
			result["date"] = event.StartDate.Local().Format(time.RFC3339)
			result["duration"] = event.GetDuration().Round(time.Second).String()
			result["eventtype"] = event.Type
			if event.Success {
				result["success"] = "true"
			} else {
				result["success"] = "false"
			}
			metadataoutput := createMetadataTable(stack, event, awsConfig, false)
			summarytable := string(metadataoutput.HtmlTableOnly())
			result["summary"] = "'" + summarytable + "'"
		}
	}
	return result
}

// createTableOutput creates the outputArray for the resource table
func createTableOutput(stack lib.CfnStack, event lib.StackEvent) format.OutputArray {
	keys := []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"}
	if !event.Success {
		keys = append(keys, "Reason")
	}
	output := format.OutputArray{Keys: keys, Settings: outputsettings}
	output.Settings.Title = fmt.Sprintf("Events of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.Local().Format(time.RFC3339))
	output.Settings.SortKey = "Start time"
	return output
}

// createMermaidOutput creates the outputArray for the mermaid graph
func createMermaidOutput(stack lib.CfnStack, event lib.StackEvent) format.OutputArray {
	mermaidoutput := format.OutputArray{Keys: []string{"Start time", "Duration", "Label"}, Settings: getReportMermaidSettings()}
	mermaidoutput.Settings.Title = fmt.Sprintf("Visual timeline of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.Local().Format(time.RFC3339))
	mermaidoutput.Settings.MermaidSettings.GanttSettings = &mermaid.GanttSettings{
		LabelColumn:     "Label",
		StartDateColumn: "Start time",
		DurationColumn:  "Duration",
		StatusColumn:    "Status",
	}
	mermaidoutput.Settings.SortKey = "Sorttime"
	// Add milestones for stack events
	for moment, status := range event.Milestones {
		mermaidcontent := make(map[string]interface{})
		mermaidcontent["Label"] = fmt.Sprintf("Stack %s", status)
		mermaidcontent["Start time"] = moment.Local().Format("15:04:05")
		mermaidcontent["Duration"] = "0s"
		mermaidcontent["Sorttime"] = moment.Local().Format(time.RFC3339)
		mermaidcontent["Status"] = "milestone"
		mermaidholder := format.OutputHolder{Contents: mermaidcontent}
		mermaidoutput.AddHolder(mermaidholder)
	}
	return mermaidoutput
}

func createMetadataTable(stack lib.CfnStack, event lib.StackEvent, awsConfig config.AWSConfig, usetitle bool) format.OutputArray {
	title := fmt.Sprintf("Metadata of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.Local().Format(time.RFC3339))
	metadatakeys := []string{"Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"}
	output := format.OutputArray{Keys: metadatakeys, Settings: outputsettings}
	if usetitle {
		output.Settings.Title = title
	}
	contents := make(map[string]interface{})
	contents["Stack"] = stack.Name
	contents["Account"] = awsConfig.GetAccountAliasID()
	contents["Region"] = awsConfig.Region
	contents["Type"] = event.Type
	contents["Start time"] = event.StartDate.Local().Format(time.RFC3339)
	contents["Duration"] = event.GetDuration().Round(time.Second).String()
	contents["Success"] = event.Success
	metadataholder := format.OutputHolder{Contents: contents}
	output.AddHolder(metadataholder)
	return output
}

func createTableResourceHolder(resource lib.ResourceEvent, event lib.StackEvent) format.OutputHolder {
	// Add row to table OutputArray
	content := make(map[string]interface{})
	content["Action"] = resource.EventType
	content["CfnName"] = resource.Resource.LogicalID
	content["Type"] = resource.Resource.Type
	content["ID"] = resource.Resource.ResourceID
	content["Start time"] = resource.StartDate.Local().Format(time.RFC3339)
	content["Duration"] = resource.GetDuration().Round(time.Second).String()
	content["Success"] = resource.EndStatus == resource.ExpectedEndStatus
	if !event.Success {
		content["Reason"] = resource.EndStatusReason
	}
	return format.OutputHolder{Contents: content}
}

func createMermaidResourceHolder(resource lib.ResourceEvent) format.OutputHolder {
	// Add row to mermaid OutputArray
	mermaidcontent := make(map[string]interface{})
	mermaidcontent["Label"] = resource.Resource.LogicalID
	mermaidcontent["Start time"] = resource.StartDate.Local().Format("15:04:05")
	mermaidcontent["Duration"] = resource.GetDuration().Round(time.Second).String()
	mermaidcontent["Sorttime"] = resource.StartDate.Local().Format(time.RFC3339)
	mermaidcontent["Status"] = ""
	if resource.EndStatus != resource.ExpectedEndStatus {
		mermaidcontent["Status"] = "done, crit"
	} else if resource.EventType == "Remove" || resource.EventType == "Cleanup" {
		mermaidcontent["Status"] = "crit"
	} else if resource.EventType == "Modify" {
		mermaidcontent["Status"] = "active"
	}
	return format.OutputHolder{Contents: mermaidcontent}
}

func cleanStackName(stackname string) string {
	if stackname == "" {
		return "all-stacks"
	}
	// If it contains a /, that means it's likely an arn. Split it up
	if strings.Contains(stackname, "/") {
		stackslice := strings.Split(stackname, "/")
		stackname = stackslice[1]
	}
	return slug.Make(stackname)
}
