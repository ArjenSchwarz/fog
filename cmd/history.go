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
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	format "github.com/ArjenSchwarz/go-output"
	"github.com/spf13/cobra"
)

var history_StackName *string

// historyCmd represents the history command
var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show the deployment history of a stack",
	Long: `This looks at the logs to show you the history of stacks in the account and region

By default it will show an overview of all stacks, but you can filter by a specific stack. In addition it should be noted that it only shows deployment logs for deployments carried out with fog. For a report that isn't dependent on fog deployments, see fog report.

All output formats are supported for this, but best results are with those supporting tables natively (table, markdown, and html).
`,
	Run: history,
}

func init() {
	rootCmd.AddCommand(historyCmd)
	history_StackName = historyCmd.Flags().StringP("stackname", "n", "", "(Optional) The name of the stack to filter by")
}

func history(cmd *cobra.Command, args []string) {
	outputsettings = settings.NewOutputSettings()
	outputsettings.SeparateTables = true
	awsConfig := config.DefaultAwsConfig(*settings)
	logs := lib.ReadAllLogs()
	for _, log := range logs {
		// Only show logs from the selected account and region
		if awsConfig.Region != log.Region || awsConfig.AccountID != log.Account {
			continue
		}
		// If filtering by a stack, only show that stack
		if *history_StackName != "" {
			if *history_StackName != log.StackName {
				continue
			}
		}
		printLog(log)
	}
	output := format.OutputArray{Keys: []string{}, Settings: settings.NewOutputSettings()}
	if *history_StackName == "" {
		output.Settings.Title = fmt.Sprintf("Deployments in account %s for region %s", awsConfig.GetAccountAliasID(), awsConfig.Region)
	} else {
		output.Settings.Title = fmt.Sprintf("Deployments for stack(s) %s in account %s for region %s", *history_StackName, awsConfig.GetAccountAliasID(), awsConfig.Region)
	}
	output.Write()
}

func printLog(log lib.DeploymentLog) {
	header := fmt.Sprintf("%v - %v", log.StartedAt.In(settings.GetTimezoneLocation()).Format(time.RFC3339), log.StackName)

	//print log entry info
	logkeys := []string{"Account", "Region", "Deployer", "Type", "Prechecks", "Started At", "Duration"}
	logtitle := "Details about the deployment"
	output := format.OutputArray{Keys: logkeys, Settings: settings.NewOutputSettings()}
	output.Settings.Title = logtitle
	if log.Status == lib.DeploymentLogStatusSuccess {
		output.AddHeader(outputsettings.StringPositive("📋 " + header))
	} else {
		output.AddHeader(outputsettings.StringWarning("📋 " + header))
	}
	contents := make(map[string]interface{})
	contents["Account"] = log.Account
	contents["Region"] = log.Region
	contents["Deployer"] = log.Deployer
	contents["Type"] = string(log.DeploymentType)
	contents["Prechecks"] = string(log.PreChecks)
	contents["Started At"] = log.StartedAt.In(settings.GetTimezoneLocation()).Format(time.RFC3339)
	contents["Duration"] = log.UpdatedAt.Sub(log.StartedAt).Round(time.Second).String()
	holder := format.OutputHolder{Contents: contents}
	output.AddHolder(holder)
	output.AddToBuffer()

	//print change set info
	changesettitle := "Deployed change set"
	summaryTitle := "Summary of changes"
	hasModule := false
	for _, change := range log.Changes {
		if change.Module != "" {
			hasModule = true
			break
		}
	}
	printChangeset(changesettitle, summaryTitle, log.Changes, hasModule)

	if log.Status == lib.DeploymentLogStatusFailed {
		//print error info
		output.AddHeader(outputsettings.StringWarning("Failed with below errors"))
		eventskeys := []string{"CfnName", "Type", "Status", "Reason"}
		eventstitle := "Failed events in deployment of change set "
		output := format.OutputArray{Keys: eventskeys, Settings: settings.NewOutputSettings()}
		output.Settings.Title = eventstitle
		for _, event := range log.Failures {
			holder := format.OutputHolder{Contents: event}
			output.AddHolder(holder)
		}
		output.AddToBuffer()
	}
}
