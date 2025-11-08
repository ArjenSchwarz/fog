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
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	output "github.com/ArjenSchwarz/go-output/v2"
	"github.com/spf13/cobra"
)

var historyFlags HistoryFlags

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
	stackCmd.AddCommand(historyCmd)
	historyFlags.RegisterFlags(historyCmd)
}

func history(cmd *cobra.Command, args []string) {
	awsConfig, err := config.DefaultAwsConfig(*settings)
	if err != nil {
		failWithError(err)
	}
	logs := lib.ReadAllLogs()
	for _, log := range logs {
		// Only show logs from the selected account and region
		if awsConfig.Region != log.Region || awsConfig.AccountID != log.Account {
			continue
		}
		// If filtering by a stack, only show that stack
		if historyFlags.StackName != "" {
			if historyFlags.StackName != log.StackName {
				continue
			}
		}
		printLog(log, false)
	}

	// Create final header document
	var title string
	if historyFlags.StackName == "" {
		title = fmt.Sprintf("Deployments in account %s for region %s", awsConfig.GetAccountAliasID(), awsConfig.Region)
	} else {
		title = fmt.Sprintf("Deployments for stack(s) %s in account %s for region %s", historyFlags.StackName, awsConfig.GetAccountAliasID(), awsConfig.Region)
	}

	// Build and render the final document with header only
	doc := output.New().Header(title).Build()
	out := output.NewOutput(settings.GetOutputOptions()...)
	err = out.Render(context.Background(), doc)
	if err != nil {
		failWithError(err)
	}
}

func printLog(log lib.DeploymentLog, useStderr bool, prefixMessage ...string) {
	header := fmt.Sprintf("%v - %v", log.StartedAt.In(settings.GetTimezoneLocation()).Format(time.RFC3339), log.StackName)

	// Create styled header based on status
	var styledHeader string
	if log.Status == lib.DeploymentLogStatusSuccess {
		styledHeader = output.StylePositive("📋 " + header)
	} else {
		styledHeader = output.StyleWarning("📋 " + header)
	}

	// Build deployment log table
	logData := []map[string]any{
		{
			"Account":    log.Account,
			"Region":     log.Region,
			"Deployer":   log.Deployer,
			"Type":       string(log.DeploymentType),
			"Prechecks":  string(log.PreChecks),
			"Started At": log.StartedAt.In(settings.GetTimezoneLocation()).Format(time.RFC3339),
			"Duration":   log.UpdatedAt.Sub(log.StartedAt).Round(time.Second).String(),
		},
	}

	builder := output.New()

	// Add optional prefix message
	if len(prefixMessage) > 0 && prefixMessage[0] != "" {
		// Add spacing before the prefix to separate from any previous output
		builder = builder.Text("\n" + prefixMessage[0])
	}

	// Add styled header as text (not Header to avoid uppercase and separators)
	builder = builder.Text(styledHeader + "\n")

	// Add table
	builder = builder.Table(
		"Details about the deployment",
		logData,
		output.WithKeys("Account", "Region", "Deployer", "Type", "Prechecks", "Started At", "Duration"),
	)

	// Add changeset tables if there are changes
	changesettitle := "Deployed change set"
	summaryTitle := "Summary of changes"
	hasModule := false
	for _, change := range log.Changes {
		if change.Module != "" {
			hasModule = true
			break
		}
	}

	builder, _ = buildChangesetDocument(builder, changesettitle, summaryTitle, log.Changes, hasModule)

	doc := builder.Build()

	// Render deployment log - use stderr for deploy context, stdout for history command
	var out *output.Output
	if useStderr {
		out = createStderrOutput()
	} else {
		out = output.NewOutput(settings.GetOutputOptions()...)
	}
	err := out.Render(context.Background(), doc)
	if err != nil {
		failWithError(err)
	}

	// print error info if failed
	if log.Status == lib.DeploymentLogStatusFailed {
		// Prepare failed events data
		failedEventsData := append([]map[string]any(nil), log.Failures...)

		// Build failed events document
		failedDoc := output.New().
			Header(output.StyleWarning("Failed with below errors")).
			Table(
				"Failed events in deployment of change set",
				failedEventsData,
				output.WithKeys("CfnName", "Type", "Status", "Reason"),
			).
			Build()

		// Render failed events
		failedOut := output.NewOutput(settings.GetOutputOptions()...)
		err := failedOut.Render(context.Background(), failedDoc)
		if err != nil {
			failWithError(err)
		}
	}
}
