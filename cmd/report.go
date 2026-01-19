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
	"sort"
	"strings"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/gosimple/slug"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	outputFormatMarkdown = "markdown"
	outputFormatHTML     = "html"
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

Note: The --file flag writes output to a local file using the exact filename provided.
For S3 output, the Outputfile parameter (used via Lambda integration) supports the
following placeholders:
- $TIMESTAMP - Current timestamp
- $REGION - AWS region
- $ACCOUNTID - AWS account ID
- $STACKNAME - CloudFormation stack name


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

	// Determine if we need Mermaid output based on format
	outputFormat := settings.GetLCString("output")
	hasMermaid := outputFormat == outputFormatMarkdown || outputFormat == outputFormatHTML
	if hasMermaid {
		reportFlags.HasMermaid = true
	}

	stacks, err := lib.GetCfnStacks(&reportFlags.StackName, awsConfig.CloudformationClient())
	if err != nil {
		failWithError(err)
	}

	// Build frontmatter if requested
	var frontMatter map[string]string
	if reportFlags.FrontMatter && outputFormat == outputFormatMarkdown {
		frontMatter = generateFrontMatter(stacks, awsConfig)
	}

	// Sort stacks by name
	stackskeys := make([]string, 0, len(stacks))
	for stackkey := range stacks {
		stackskeys = append(stackskeys, stackkey)
	}
	sort.Strings(stackskeys)

	// Build document with all tables
	doc := output.New()

	// Add document title
	latestText := ""
	if reportFlags.LatestOnly {
		latestText = "Last event only."
	}
	if len(frontMatter) != 0 {
		latestText = "Single event."
	}
	switch {
	case reportFlags.StackName == "":
		doc.Header(fmt.Sprintf("Fog report for account %s. %s", awsConfig.GetAccountAliasID(), latestText))
	case strings.Contains(reportFlags.StackName, "*"):
		doc.Header(fmt.Sprintf("Fog report for stacks matching '%s'. %s", reportFlags.StackName, latestText))
	default:
		doc.Header(fmt.Sprintf("Fog report for stack %s. %s", cleanStackName(reportFlags.StackName), latestText))
	}

	// Generate report for each stack
	for _, stackkey := range stackskeys {
		fmt.Println(stackkey)
		generateStackReport(stacks[stackkey], doc, awsConfig)
	}

	// Get output options with S3/file configuration if needed
	outputOptions := getReportOutputOptions(awsConfig, frontMatter)

	// Render the complete document to console (and file if formats match, or S3)
	builtDoc := doc.Build()
	out := output.NewOutput(outputOptions...)
	if err := out.Render(context.Background(), builtDoc); err != nil {
		failWithError(err)
	}

	// Render to file separately if file format differs from console format
	if fileOpts := settings.GetFileOutputOptions(); fileOpts != nil {
		// Add frontmatter to file output if provided
		if len(frontMatter) > 0 {
			fileOpts = append(fileOpts, output.WithFrontMatter(frontMatter))
		}
		fileOut := output.NewOutput(fileOpts...)
		if err := fileOut.Render(context.Background(), builtDoc); err != nil {
			failWithError(err)
		}
	}
}

// getReportOutputOptions creates output options with S3/file support
func getReportOutputOptions(awsConfig config.AWSConfig, frontMatter map[string]string) []output.OutputOption {
	opts := settings.GetOutputOptions()

	// Add frontmatter if provided (for markdown output)
	if len(frontMatter) > 0 {
		opts = append(opts, output.WithFrontMatter(frontMatter))
	}

	// Handle S3 bucket output if configured
	if reportFlags.TargetBucket != "" {
		// Build S3 key pattern
		keyPattern := reportFlags.Outputfile
		if keyPattern == "" {
			ext := getDefaultExtension(settings.GetLCString("output"))
			keyPattern = cleanStackName(reportFlags.StackName) + "/" + time.Now().Format(time.RFC3339) + ext
		} else {
			keyPattern = reportPlaceholderParser(keyPattern, reportFlags.StackName, awsConfig)
		}

		// Create S3 writer - v2.3.2+ supports AWS SDK v2 clients directly
		s3Writer := output.NewS3Writer(awsConfig.S3Client(), reportFlags.TargetBucket, keyPattern)
		opts = append(opts, output.WithWriter(s3Writer))
	}

	return opts
}

// getDefaultExtension returns the default file extension for a format
func getDefaultExtension(format string) string {
	switch format {
	case outputFormatMarkdown, outputFormatHTML:
		return ".md"
	case "json":
		return ".json"
	case "csv":
		return ".csv"
	default:
		return ".txt"
	}
}

func reportPlaceholderParser(value string, stackname string, awsConfig config.AWSConfig) string {
	value = strings.ReplaceAll(value, "$TIMESTAMP", time.Now().In(settings.GetTimezoneLocation()).Format("2006-01-02T15-04-05"))
	value = strings.ReplaceAll(value, "$STACKNAME", cleanStackName(stackname))
	value = strings.ReplaceAll(value, "$REGION", awsConfig.Region)
	value = strings.ReplaceAll(value, "$ACCOUNTID", awsConfig.AccountID)
	return value
}

// generateStackReport creates the report for the provided stack
func generateStackReport(stack lib.CfnStack, doc *output.Builder, awsConfig config.AWSConfig) {
	// Add stack header
	doc.Header(fmt.Sprintf("Stack %s", stack.Name))

	events, err := stack.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		failWithError(err)
	}

	for counter, event := range events {
		if reportFlags.LatestOnly && counter+1 < len(events) {
			continue
		}

		// Create metadata table
		metadataTitle, metadataData := createMetadataTable(stack, event, awsConfig)
		doc.Table(metadataTitle, metadataData, output.WithKeys("Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"))

		// Create events table (data is sorted within the helper function)
		eventTitle, eventKeys, eventData := createEventsTable(stack, event)
		doc.Table(eventTitle, eventData, output.WithKeys(eventKeys...))

		// Add Mermaid diagram if needed (uses v2 GanttChart for proper rendering)
		if reportFlags.HasMermaid {
			mermaidTitle, ganttTasks := createMermaidGanttChart(stack, event)
			doc.GanttChart(mermaidTitle, ganttTasks)
		}
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
			// Create metadata summary for frontmatter
			_, metadataData := createMetadataTable(stack, event, awsConfig)
			if len(metadataData) > 0 {
				// Build a simple HTML table representation for frontmatter
				// This is a simplified version since we can't use HtmlTableOnly() in v2
				summarytable := buildSimpleHTMLTable(metadataData[0])
				result["summary"] = "'" + summarytable + "'"
			}
		}
	}
	return result
}

// buildSimpleHTMLTable creates a simple HTML table from a data row
func buildSimpleHTMLTable(data map[string]any) string {
	var sb strings.Builder
	sb.WriteString("<table>")
	for k, v := range data {
		sb.WriteString(fmt.Sprintf("<tr><th>%s</th><td>%v</td></tr>", k, v))
	}
	sb.WriteString("</table>")
	return sb.String()
}

// createEventsTable creates the data for the resource events table
func createEventsTable(stack lib.CfnStack, event lib.StackEvent) (string, []string, []map[string]any) {
	keys := []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"}
	if !event.Success {
		keys = append(keys, "Reason")
	}

	title := fmt.Sprintf("Event details of %s - %s event - Started %s",
		stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))

	// Build data rows
	data := make([]map[string]any, 0, len(event.ResourceEvents))
	for _, resource := range event.ResourceEvents {
		row := createTableResourceData(resource, event)
		data = append(data, row)
	}

	// Sort by start time
	// Note: Using manual sorting instead of v2.4.0 transformations because:
	// 1. Small datasets (CloudFormation events) where manual sorting is efficient and clear
	// 2. Other sorting in this file (stack names, Gantt tasks) can't use transformations,
	//    so keeping consistent approach across all sorting operations
	sort.Slice(data, func(i, j int) bool {
		return data[i]["Start time"].(string) < data[j]["Start time"].(string)
	})

	return title, keys, data
}

// createMermaidGanttChart creates Gantt chart tasks for the Mermaid diagram
func createMermaidGanttChart(stack lib.CfnStack, event lib.StackEvent) (string, []output.GanttTask) {
	title := fmt.Sprintf("Visual timeline of %s - %s event - Started %s",
		stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))

	tasks := make([]output.GanttTask, 0)

	// Create a slice to hold all events for sorting
	type sortableEvent struct {
		task     output.GanttTask
		sortTime time.Time
	}
	sortableEvents := make([]sortableEvent, 0)

	// Add milestones for stack events
	for moment, status := range event.Milestones {
		task := output.GanttTask{
			ID:        fmt.Sprintf("milestone-%d", moment.Unix()),
			Title:     fmt.Sprintf("Stack %s", status),
			StartDate: moment.In(settings.GetTimezoneLocation()).Format("2006-01-02 15:04:05"),
			Duration:  "0s",
			Status:    "milestone",
		}
		sortableEvents = append(sortableEvents, sortableEvent{task: task, sortTime: moment})
	}

	// Add resource events
	for _, resource := range event.ResourceEvents {
		task := createMermaidGanttTask(resource)
		sortableEvents = append(sortableEvents, sortableEvent{task: task, sortTime: resource.StartDate})
	}

	// Sort by time
	sort.Slice(sortableEvents, func(i, j int) bool {
		return sortableEvents[i].sortTime.Before(sortableEvents[j].sortTime)
	})

	// Extract sorted tasks
	for _, se := range sortableEvents {
		tasks = append(tasks, se.task)
	}

	return title, tasks
}

func createMetadataTable(stack lib.CfnStack, event lib.StackEvent, awsConfig config.AWSConfig) (string, []map[string]any) {
	title := fmt.Sprintf("Metadata of %s - %s event - Started %s",
		stack.Name, event.Type, event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339))

	data := []map[string]any{
		{
			"Stack":      stack.Name,
			"Account":    awsConfig.GetAccountAliasID(),
			"Region":     awsConfig.Region,
			"Type":       event.Type,
			"Start time": event.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339),
			"Duration":   event.GetDuration().Round(time.Second).String(),
			"Success":    event.Success,
		},
	}

	return title, data
}

func createTableResourceData(resource lib.ResourceEvent, event lib.StackEvent) map[string]any {
	content := map[string]any{
		"Action":     resource.EventType,
		"CfnName":    resource.Resource.LogicalID,
		"Type":       resource.Resource.Type,
		"ID":         resource.Resource.ResourceID,
		"Start time": resource.StartDate.In(settings.GetTimezoneLocation()).Format(time.RFC3339),
		"Duration":   resource.GetDuration().Round(time.Second).String(),
		"Success":    resource.EndStatus == resource.ExpectedEndStatus,
	}

	if !event.Success {
		content["Reason"] = resource.EndStatusReason
	}

	return content
}

func createMermaidGanttTask(resource lib.ResourceEvent) output.GanttTask {
	task := output.GanttTask{
		ID:        resource.Resource.LogicalID,
		Title:     resource.Resource.LogicalID,
		StartDate: resource.StartDate.In(settings.GetTimezoneLocation()).Format("2006-01-02 15:04:05"),
		Duration:  resource.GetDuration().Round(time.Second).String(),
		Status:    "",
	}

	switch {
	case resource.EndStatus != resource.ExpectedEndStatus:
		task.Status = "done, crit"
	case resource.EventType == eventTypeRemove || resource.EventType == "Cleanup":
		task.Status = "crit"
	case resource.EventType == "Modify":
		task.Status = "active"
	}

	return task
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
