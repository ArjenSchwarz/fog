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
	"log"
	"os"
	"sort"
	"time"

	"github.com/ArjenSchwarz/fog/config"
	"github.com/ArjenSchwarz/fog/lib"
	"github.com/ArjenSchwarz/fog/lib/format"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy a CloudFormation stack",
	Long: `deploy allows you to deploy a CloudFormation stack

It does so by creating a ChangeSet and then asking you for approval before continuing. You can automatically approve or only create or deploy a changeset by using flags.

A name for the changeset will automatically be generated based on your preferred name, but can be overwritten as well.

Examples: fog deploy mytemplate`,
	Run: deployTemplate,
}

var deploy_StackName *string
var deploy_Template *string
var deploy_Parameters *string
var deploy_Tags *string
var deploy_Dryrun *bool
var deployment lib.DeployInfo

func init() {
	rootCmd.AddCommand(deployCmd)
	deploy_StackName = deployCmd.Flags().StringP("stackname", "n", "", "The name for the stack")
	deploy_Template = deployCmd.Flags().StringP("file", "f", "", "The filename for the template")
	deploy_Parameters = deployCmd.Flags().StringP("parameters", "p", "", "The filename for the parameters")
	deploy_Tags = deployCmd.Flags().StringP("tags", "t", "", "The filename for the tags")
	deploy_Dryrun = deployCmd.Flags().Bool("dryrun", false, "Do a dry run: create the changeset and immediately delete")
}

func deployTemplate(cmd *cobra.Command, args []string) {
	settings.SeparateTables = true //Make table output stand out more
	deployment.StackName = *deploy_StackName
	awsConfig := config.DefaultAwsConfig(*settings)
	deployment.IsNew = !lib.StackExists(&deployment, awsConfig.CloudformationClient())
	if !deployment.IsNew {
		if ready, status := deployment.IsReadyForUpdate(awsConfig.CloudformationClient()); !ready {
			message := fmt.Sprintf("The stack '%v' is currently in status %v and can't be updated", *deploy_StackName, status)
			settings.PrintFailure(message)
			os.Exit(1)
		}
	}
	bold := color.New(color.Bold).SprintFunc()
	if deployment.IsNew {
		method := "Deploying"
		if *deploy_Dryrun {
			method = fmt.Sprintf("Doing a %v for", bold("dry run"))
		}
		fmt.Printf("%v new stack '%v' to region %v of account %v\n", method, bold(*deploy_StackName), awsConfig.Region, awsConfig.AccountID)
	} else {
		method := "Updating"
		if *deploy_Dryrun {
			method = fmt.Sprintf("Doing a %v for updating", bold("dry run"))
		}
		fmt.Printf("%v stack '%v' in region %v of account %v\n", method, bold(*deploy_StackName), awsConfig.Region, awsConfig.AccountID)
	}
	template, err := lib.ReadTemplate(deploy_Template)
	if err != nil {
		panic(err)
	}
	deployment.Template = template
	if *deploy_Tags != "" {
		tags, err := lib.ReadTagsfile(deploy_Tags)
		if err != nil {
			log.Fatalln(err)
		}
		deployment.TagsFile = tags
	}
	if *deploy_Parameters != "" {
		parameters, err := lib.ReadParametersfile(deploy_Parameters)
		if err != nil {
			log.Fatalln(err)
		}
		deployment.ParametersFile = parameters
	}
	deployment.ChangesetName = settings.GetString("changesetname")
	_, err = deployment.CreateChangeSet(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure(deployChangesetMessageCreationFailed)
		log.Fatalln(err)
	}
	changeset, err := deployment.WaitUntilChangesetDone(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure(deployChangesetMessageCreationFailed)
		log.Fatalln(err)
	}
	if changeset.Status != string(types.ChangeSetStatusCreateComplete) {
		settings.PrintFailure(deployChangesetMessageCreationFailed)
		fmt.Println(changeset.StatusReason)
		fmt.Println("")
		fmt.Printf("%v %v \r\n", deployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
		deployChangesetConfirmation := askForConfirmation("Do you want to delete this change set?")
		if deployChangesetConfirmation {
			deleteChangeset(deployment, awsConfig)
		}
		os.Exit(1)
	}
	showChangeset(*changeset, awsConfig)
	if *deploy_Dryrun {
		deleteChangeset(deployment, awsConfig)
		os.Exit(0)
	}
	deployChangesetConfirmation := askForConfirmation("Do you want to deploy this changeset?")
	if deployChangesetConfirmation {
		deployChangeset(deployment, awsConfig)
	} else {
		deleteChangeset(deployment, awsConfig)
		os.Exit(0)
	}
	resultStack, err := deployment.GetFreshStack(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure("Something went wrong when I tried to fetch the stack after the deployment.")
		log.Fatalln(err.Error())
	}
	switch resultStack.StackStatus {
	case types.StackStatusCreateComplete, types.StackStatusUpdateComplete:
		settings.PrintSuccess("Deployment completed successfully")
		if len(resultStack.Outputs) > 0 {
			outputkeys := []string{"Key", "Value", "Description", "ExportName"}
			outputtitle := fmt.Sprintf("Outputs for stack %v", *resultStack.StackName)
			output := format.OutputArray{Keys: outputkeys, Title: outputtitle}
			for _, outputresult := range resultStack.Outputs {
				exportName := ""
				if outputresult.ExportName != nil {
					exportName = *outputresult.ExportName
				}
				description := ""
				if outputresult.Description != nil {
					description = *outputresult.Description
				}
				content := make(map[string]string)
				content["Key"] = *outputresult.OutputKey
				content["Value"] = *outputresult.OutputValue
				content["Description"] = description
				content["ExportName"] = exportName
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
			output.Write(*settings)
		}
	case types.StackStatusRollbackComplete, types.StackStatusRollbackFailed, types.StackStatusUpdateRollbackComplete, types.StackStatusUpdateRollbackFailed:
		settings.PrintFailure("The deployment went wrong, please look at the error messages below to figure out why")
		showFailedEvents(deployment, awsConfig)
		if deployment.IsNew {
			//double verify that the stack can be deleted
			deleteStackIfNew(deployment, awsConfig)
		}
	}
}

func showChangeset(changeset lib.ChangesetInfo, awsConfig config.AWSConfig) {
	bold := color.New(color.Bold).SprintFunc()
	changesetkeys := []string{"Action", "CfnName", "Type", "ID", "Replacement"}
	changesettitle := fmt.Sprintf("Changes in change set %v", changeset.Name)
	output := format.OutputArray{Keys: changesetkeys, Title: changesettitle}
	output.SortKey = "Type"
	if len(changeset.Changes) == 0 {
		fmt.Println(deployChangesetMessageNoChanges)
	} else {
		for _, change := range changeset.Changes {
			content := make(map[string]string)
			action := change.Action
			if action == "Remove" {
				action = bold(action)
			}
			content["Action"] = action
			content["Replacement"] = change.Replacement
			content["CfnName"] = change.LogicalID
			content["Type"] = change.Type
			content["ID"] = change.ResourceID
			holder := format.OutputHolder{Contents: content}
			output.AddHolder(holder)
		}
		output.Write(*settings)
	}
	fmt.Printf("%v %v \r\n", deployChangesetMessageConsole, changeset.GenerateChangesetUrl(awsConfig))
}

func deleteChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	if *deploy_Dryrun {
		settings.PrintSuccess(deployChangesetMessageDryrunDelete)
	} else {
		settings.PrintSuccess(deployChangesetMessageWillDelete)
	}
	deleteAttempt := deployment.Changeset.DeleteChangeset(awsConfig.CloudformationClient())
	if !deleteAttempt {
		settings.PrintFailure(deployChangesetMessageDeleteFailed)
	}
	// Likely a new deployment. Check if the stack is in status REVIEW_IN_PROGRESS and offer to delete
	if deployment.IsNew {
		stack, err := deployment.GetFreshStack(awsConfig.CloudformationClient())
		if err != nil {
			log.Fatalln(err)
		}
		if stack.StackStatus == types.StackStatusReviewInProgress {
			deleteStackIfNew(deployment, awsConfig)
		}
	}
}

func deleteStackIfNew(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	fmt.Println(deployStackMessageNewStackDeleteInfo)
	var deleteStackConfirmation bool
	if *deploy_Dryrun {
		deleteStackConfirmation = true
	} else {
		deleteStackConfirmation = askForConfirmation("Do you want me to delete this empty stack for you?")
	}
	if deleteStackConfirmation {
		if !deployment.DeleteStack(awsConfig.CloudformationClient()) {
			settings.PrintFailure("Something went wrong while trying to delete the stack. Please check manually.")
		} else {
			if *deploy_Dryrun {
				settings.PrintSuccess(deployStackMessageNewStackDryrunDelete)
			} else {
				settings.PrintSuccess(deployStackMessageNewStackDeleteSuccess)
			}
		}
	} else {
		fmt.Println("No problem. I have left the stack intact, please delete it manually once you're done.")
	}
}

func deployChangeset(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	settings.PrintSuccess("OK. Deploying this Changeset.")
	err := deployment.Changeset.DeployChangeset(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure("Could not execute changeset! See details below")
		fmt.Println(err)
	}
	latest := deployment.Changeset.CreationTime
	time.Sleep(3 * time.Second)
	settings.PrintBold("Showing the events for the deployment:")
	ongoing := true
	for ongoing {
		latest = showEvents(deployment, latest, awsConfig)
		time.Sleep(3 * time.Second)
		ongoing = deployment.IsOngoing(awsConfig.CloudformationClient())
	}
	// One last time after the deployment finished in case of a timing mismatch
	showEvents(deployment, latest, awsConfig)
}

func showEvents(deployment lib.DeployInfo, latest time.Time, awsConfig config.AWSConfig) time.Time {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure("Something went wrong trying to get the events of the stack")
		fmt.Println(err)
	}
	sort.Sort(ReverseEvents(events))
	for _, event := range events {
		if event.Timestamp.After(latest) {
			latest = *event.Timestamp
			message := fmt.Sprintf("%v: %v %v in status %v", event.Timestamp.Local().Format("2006-01-02 15:04:05 MST"), *event.ResourceType, *event.LogicalResourceId, event.ResourceStatus)
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed, types.ResourceStatusImportRollbackComplete, types.ResourceStatus(types.StackStatusRollbackComplete), types.ResourceStatus(types.StackStatusUpdateRollbackComplete):
				settings.PrintWarning(message)
			case types.ResourceStatusCreateComplete, types.ResourceStatusImportComplete, types.ResourceStatusUpdateComplete, types.ResourceStatusDeleteComplete:
				settings.PrintPositive(message)
			default:
				fmt.Println(message)
			}
		}
	}
	return latest
}

func showFailedEvents(deployment lib.DeployInfo, awsConfig config.AWSConfig) {
	events, err := deployment.GetEvents(awsConfig.CloudformationClient())
	if err != nil {
		settings.PrintFailure("Something went wrong trying to get the events of the stack")
		fmt.Println(err)
	}
	changesetkeys := []string{"CfnName", "Type", "Status", "Reason"}
	changesettitle := fmt.Sprintf("Failed events in deployment of changeset %v", deployment.Changeset.Name)
	output := format.OutputArray{Keys: changesetkeys, Title: changesettitle}
	sort.Sort(ReverseEvents(events))
	for _, event := range events {
		if event.Timestamp.After(deployment.Changeset.CreationTime) {
			switch event.ResourceStatus {
			case types.ResourceStatusCreateFailed, types.ResourceStatusImportFailed, types.ResourceStatusDeleteFailed, types.ResourceStatusUpdateFailed:
				content := make(map[string]string)
				content["CfnName"] = *event.LogicalResourceId
				content["Type"] = *event.ResourceType
				content["Status"] = string(event.ResourceStatus)
				content["Reason"] = *event.ResourceStatusReason
				holder := format.OutputHolder{Contents: content}
				output.AddHolder(holder)
			}
		}
	}
	output.Write(*settings)
}

type ReverseEvents []types.StackEvent

func (a ReverseEvents) Len() int           { return len(a) }
func (a ReverseEvents) Less(i, j int) bool { return a[i].Timestamp.Before(*a[j].Timestamp) }
func (a ReverseEvents) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
