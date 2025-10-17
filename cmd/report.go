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
	format "github.com/ArjenSchwarz/go-output/v2"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	outputFormatMarkdown = "markdown"
	eventTypeRemove      = "Remove"
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

var reportFlags ReportFlags

func init() {
	stackCmd.AddCommand(reportCmd)
	reportFlags.RegisterFlags(reportCmd)
}

func stackReport(cmd *cobra.Command, args []string) {
	generateReport()
}

// GenerateReportFromLambda generates a CloudFormation deployment report for Lambda execution
func GenerateReportFromLambda(stackname string, bucketname string, outputfilename string, outputformat string, timezone string) {
	// Default settings for Lambda output: only latest, markdown, with frontmatter
	reportFlags.LatestOnly = true // The Lambda always only retrieves the latest report
	reportFlags.FrontMatter = true
	viper.Set("output", outputformat)
	viper.Set("timezone", timezone)
	reportFlags.StackName = stackname
	reportFlags.TargetBucket = bucketname
	reportFlags.Outputfile = outputfilename
	generateReport()
}

// generateReport creates the complete report
func generateReport() {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	outputsettings = getReportOutputSettingsFromCli(awsConfig)
	mainoutput := format.OutputArray{Keys: []string{}, Settings: outputsettings}
	if mainoutput.Settings.OutputFormat == outputFormatMarkdown || mainoutput.Settings.OutputFormat == "html" {
		reportFlags.HasMermaid = true
	}
	stacks, err := lib.GetCfnStacks(&reportFlags.StackName, awsConfig.CloudformationClient())
	if reportFlags.FrontMatter && outputsettings.OutputFormat == outputFormatMarkdown {
		mainoutput.Settings.FrontMatter = generateFrontMatter(stacks, awsConfig)
	}

	if err != nil {
		failWithError(err)
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
		fmt.Println(stackkey)
		generateStackReport(stacks[stackkey], mainoutput, awsConfig)
	}
	// Set the title for the output file that we actually want
	latestText := ""
	if reportFlags.LatestOnly {
		latestText = "Last event only."
	}
	if len(mainoutput.Settings.FrontMatter) != 0 {
		latestText = "Single event."
	}
	switch {
	case reportFlags.StackName == "":
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for account %s. %s", awsConfig.GetAccountAliasID(), latestText)
	case strings.Contains(reportFlags.StackName, "*"):
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for stacks matching '%s'. %s", reportFlags.StackName, latestText)
	default:
		mainoutput.Settings.Title = fmt.Sprintf("Fog report for stack %s. %s", cleanStackName(reportFlags.StackName), latestText)
	}
	mainoutput.Write()
}

func getReportOutputSettingsFromCli(awsConfig config.AWSConfig) *format.OutputSettings {
	settings := settings.NewOutputSettings()
	settings.OutputFile = reportPlaceholderParser(reportFlags.Outputfile, reportFlags.StackName, awsConfig)
	if reportFlags.TargetBucket != "" {
		targetpath := settings.OutputFile
		if targetpath == "" {
			targetpath = cleanStackName(reportFlags.StackName) + "/" + time.Now().Format(time.RFC3339) + settings.GetDefaultExtension()
		}
		settings.SetS3Bucket(awsConfig.S3Client(), reportFlags.TargetBucket, targetpath)
	}
	settings.SeparateTables = true
	return settings
}

func reportPlaceholderParser(value string, stackname string, awsConfig config.AWSConfig) string {
	value = strings.ReplaceAll(value, "$TIMESTAMP", time.Now().In(settings.GetTimezoneLocation()).Format("2006-01-02T15-04-05"))
	value = strings.ReplaceAll(value, "$STACKNAME", cleanStackName(stackname))
	value = strings.ReplaceAll(value, "$REGION", awsConfig.Region)
	value = strings.ReplaceAll(value, "$ACCOUNTID", awsConfig.AccountID)
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
		failWithError(err)
	}
	for counter, event := range events {
		if reportFlags.LatestOnly && counter+1 < len(events) {
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
		// Only add mermaid to the buffer if the output needs it
		if reportFlags.HasMermaid {
			mermaidoutput.AddToBuffer()
		}
		output.AddToBuffer()
	}
}

func generateFrontMatter(stacks map[string]lib.CfnStack, awsConfig config.AWSConfig) map[string]string {
	result := make(map[string]string)
	for _, stack := range stacks {
		events, err := stack.GetEvents(awsConfig.CloudformationClient())
		if err != nil {
			failWithError(err)
		}
		for _, event := range events {
			result["account"] = awsConfig.AccountID
			result["accountalias"] = awsConfig.GetAccountAliasID()
			result["region"] = awsConfig.Region
			result["stack"] = stack.Name
			result["date"] = event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
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
	output.Settings.Title = fmt.Sprintf("Event details of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))
	output.Settings.SortKey = "Start time"
	return output
}

// createMermaidOutput creates the outputArray for the mermaid graph
func createMermaidOutput(stack lib.CfnStack, event lib.StackEvent) format.OutputArray {
	mermaidoutput := format.OutputArray{Keys: []string{"Start time", "Duration", "Label"}, Settings: getReportMermaidSettings()}
	mermaidoutput.Settings.Title = fmt.Sprintf("Visual timeline of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))
	mermaidoutput.Settings.MermaidSettings.GanttSettings = &mermaid.GanttSettings{
		LabelColumn:     "Label",
		StartDateColumn: "Start time",
		DurationColumn:  "Duration",
		StatusColumn:    "Status",
	}
	mermaidoutput.Settings.SortKey = "Sorttime"
	// Add milestones for stack events
	for moment, status := range event.Milestones {
		mermaidcontent := make(map[string]any)
		mermaidcontent["Label"] = fmt.Sprintf("Stack %s", status)
		mermaidcontent["Start time"] = moment.In(settings.GetTimezoneLocation()).Format("15:04:05")
		mermaidcontent["Duration"] = "0s"
		mermaidcontent["Sorttime"] = moment.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
		mermaidcontent["Status"] = "milestone"
		mermaidholder := format.OutputHolder{Contents: mermaidcontent}
		mermaidoutput.AddHolder(mermaidholder)
	}
	return mermaidoutput
}

func createMetadataTable(stack lib.CfnStack, event lib.StackEvent, awsConfig config.AWSConfig, usetitle bool) format.OutputArray {
	title := fmt.Sprintf("Metadata of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))
	metadatakeys := []string{"Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"}
	output := format.OutputArray{Keys: metadatakeys, Settings: outputsettings}
	if usetitle {
		output.Settings.Title = title
	}
	contents := make(map[string]any)
	contents["Stack"] = stack.Name
	contents["Account"] = awsConfig.GetAccountAliasID()
	contents["Region"] = awsConfig.Region
	contents["Type"] = event.Type
	contents["Start time"] = event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
	contents["Duration"] = event.GetDuration().Round(time.Second).String()
	contents["Success"] = event.Success
	metadataholder := format.OutputHolder{Contents: contents}
	output.AddHolder(metadataholder)
	return output
}

func createTableResourceHolder(resource lib.ResourceEvent, event lib.StackEvent) format.OutputHolder {
	// Add row to table OutputArray
	content := make(map[string]any)
	content["Action"] = resource.EventType
	content["CfnName"] = resource.Resource.LogicalID
	content["Type"] = resource.Resource.Type
	content["ID"] = resource.Resource.ResourceID
	content["Start time"] = resource.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
	content["Duration"] = resource.GetDuration().Round(time.Second).String()
	content["Success"] = resource.EndStatus == resource.ExpectedEndStatus
	if !event.Success {
		content["Reason"] = resource.EndStatusReason
	}
	return format.OutputHolder{Contents: content}
}

func createMermaidResourceHolder(resource lib.ResourceEvent) format.OutputHolder {
	// Add row to mermaid OutputArray
	mermaidcontent := make(map[string]any)
	mermaidcontent["Label"] = resource.Resource.LogicalID
	mermaidcontent["Start time"] = resource.StartDate.In(settings.GetTimezoneLocation()).Format("15:04:05")
	mermaidcontent["Duration"] = resource.GetDuration().Round(time.Second).String()
	mermaidcontent["Sorttime"] = resource.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
	mermaidcontent["Status"] = ""
	switch {
	case resource.EndStatus != resource.ExpectedEndStatus:
		mermaidcontent["Status"] = "done, crit"
	case resource.EventType == eventTypeRemove || resource.EventType == "Cleanup":
		mermaidcontent["Status"] = "crit"
	case resource.EventType == "Modify":
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
