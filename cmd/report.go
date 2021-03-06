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
var report_LatestOnly *bool

func init() {
	rootCmd.AddCommand(reportCmd)
	report_StackName = reportCmd.Flags().StringP("stackname", "n", "", "The name for the stack")
	report_Outputfile = reportCmd.Flags().String("file", "", "Optional file to save the output to")
	report_LatestOnly = reportCmd.Flags().Bool("latest", false, "Only show the latest event")
}

func stackReport(cmd *cobra.Command, args []string) {
	outputsettings = settings.NewOutputSettings()
	outputsettings.OutputFile = *report_Outputfile
	outputsettings.SeparateTables = true
	mermaidoutputsettings := settings.NewOutputSettings()
	mermaidoutputsettings.SetOutputFormat("mermaid")
	mermaidoutputsettings.MermaidSettings.ChartType = "ganttchart"
	switch outputsettings.OutputFormat {
	case "markdown":
		mermaidoutputsettings.MermaidSettings.AddMarkdown = true
	case "html":
		mermaidoutputsettings.MermaidSettings.AddHTML = true
	}
	awsConfig := config.DefaultAwsConfig(*settings)
	mainoutput := format.OutputArray{Keys: []string{}, Settings: outputsettings}
	stacks, err := lib.GetCfnStacks(report_StackName, awsConfig.CloudformationClient())
	if err != nil {
		panic(err)
	}
	if len(stacks) > 1 {
		mainoutput.Settings.HasTOC = true
	}
	stackskeys := make([]string, 0, len(stacks))
	for stackkey := range stacks {
		stackskeys = append(stackskeys, stackkey)
	}
	sort.Strings(stackskeys)

	for _, stackkey := range stackskeys {
		stack := stacks[stackkey]
		mainoutput.AddHeader(fmt.Sprintf("Stack %s", stack.Name))
		events, err := stack.GetEvents(awsConfig.CloudformationClient())
		if err != nil {
			panic(err)
		}
		for counter, event := range events {
			keys := []string{"Action", "CfnName", "Type", "ID", "Start time", "Duration", "Success"}
			if !event.Success {
				keys = append(keys, "Reason")
			}
			if *report_LatestOnly && counter+1 < len(events) {
				continue
			}
			// Create metadata table
			title := fmt.Sprintf("Metadata of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.Local().Format(time.RFC3339))
			metadatakeys := []string{"Stack", "Account", "Region", "Type", "Start time", "Duration", "Success"}
			metadataoutput := format.OutputArray{Keys: metadatakeys, Settings: outputsettings}
			metadataoutput.Settings.Title = title
			metadatacontent := make(map[string]interface{})
			metadatacontent["Stack"] = stack.Name
			metadatacontent["Account"] = awsConfig.GetAccountAliasID()
			metadatacontent["Region"] = awsConfig.Region
			metadatacontent["Type"] = event.Type
			metadatacontent["Start time"] = event.StartDate.Local().Format(time.RFC3339)
			metadatacontent["Duration"] = event.GetDuration().Round(time.Second).String()
			metadatacontent["Success"] = event.Success
			metadataholder := format.OutputHolder{Contents: metadatacontent}
			metadataoutput.AddHolder(metadataholder)
			metadataoutput.AddToBuffer()

			// Create outputarray for table
			output := format.OutputArray{Keys: keys, Settings: outputsettings}
			output.Settings.Title = fmt.Sprintf("Events of %s - %s event - Started %s", stack.Name, event.Type, event.StartDate.Local().Format(time.RFC3339))
			output.Settings.SortKey = "Start time"

			// Create OutputArray for mermaid diagram
			mermaidoutput := format.OutputArray{Keys: []string{"Start time", "Duration", "Label"}, Settings: mermaidoutputsettings}
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

			for _, resource := range event.ResourceEvents {
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
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)

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
				mermaidholder := format.OutputHolder{Contents: mermaidcontent}
				mermaidoutput.AddHolder(mermaidholder)
			}
			output.AddToBuffer()
			if mainoutput.Settings.OutputFormat == "markdown" || mainoutput.Settings.OutputFormat == "html" {
				mermaidoutput.AddToBuffer()
			}

		}
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
